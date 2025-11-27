package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"
	"time"
	"unicode"

	"github.com/blocktransaction/zen/common/constant"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ResponseWriter 包装器
type bodyLogWriter struct {
	gin.ResponseWriter
	body       *bytes.Buffer
	maxBodyLen int
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	// 1. 缓存响应体（仅缓存到 maxBodyLen）
	currentLen := w.body.Len()
	if currentLen < w.maxBodyLen {
		remain := w.maxBodyLen - currentLen
		// 只有当前剩余的字节需要被缓存
		bytesToCache := b
		if len(b) > remain {
			bytesToCache = b[:remain]
		}
		w.body.Write(bytesToCache) // 只调用一次 Write
	}

	// 2. 将数据写入实际的 ResponseWriter
	return w.ResponseWriter.Write(b)
}

// 日志记录
func GinzapWithBody(logger *zap.Logger, opts ...Option) gin.HandlerFunc {
	options := &logConfig{}
	for _, o := range opts {
		o(options)
	}
	return func(c *gin.Context) {
		start := time.Now()

		var reqBody string
		var peekReq []byte
		contentType := c.Request.Header.Get("Content-Type")

		if options.enableReqBody && c.Request.Body != nil {
			// 先读取受限长度（MaxBodySize + 1 用于判断是否被截断）
			limit := int64(options.maxBodySize + 1)
			bodyBytes, _ := io.ReadAll(io.LimitReader(c.Request.Body, limit))
			// 还原到请求体，保证后续 handler 能正常读取
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			peekReq = bodyBytes // 用于 GuessJSON

			// 决定是否记录请求体
			if shouldLogReqBody(*options, contentType, peekReq) {
				if len(bodyBytes) > options.maxBodySize {
					reqBody = string(bodyBytes[:options.maxBodySize]) + "...(truncated)"
				} else {
					reqBody = string(bodyBytes)
				}
				reqBody = maskSensitive(reqBody, options.sensitiveKeys)
			}
		}

		blw := &bodyLogWriter{
			body:           bytes.NewBufferString(""),
			ResponseWriter: c.Writer,
			maxBodyLen:     options.maxBodySize,
		}
		c.Writer = blw

		c.Next()

		//计算用时（毫秒）
		elapsedMsFloat := float64(time.Since(start)) / float64(time.Millisecond)
		var respBody string
		if options.enableRespBody {
			respCT := c.Writer.Header().Get("Content-Type")
			// 决定是否记录响应体：优先根据 header 判定；若 header 缺失且允许猜测，可 peek
			peekResp := blw.body.Bytes()
			shouldLogResp := !options.onlyJSONBody || isJSONContentType(respCT)
			if !shouldLogResp && options.guessJSON && len(peekResp) > 0 {
				shouldLogResp = looksLikeJSONBody(peekResp)
			}

			if shouldLogResp {
				if len(peekResp) > options.maxBodySize {
					respBody = string(peekResp[:options.maxBodySize]) + "...(truncated)"
				} else {
					respBody = blw.body.String()
				}
				respBody = maskSensitive(respBody, options.sensitiveKeys)
			}
		}

		if len(c.Errors) > 0 {
			for _, e := range c.Errors.Errors() {
				logger.Error(e)
			}
		} else {
			logger.Info("http request log",
				zap.String("traceId", c.GetString(constant.TraceId)),
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.String("query", c.Request.URL.RawQuery),
				zap.Int("status", c.Writer.Status()),
				zap.String("ip", c.ClientIP()),
				zap.String("latency", fmt.Sprintf("%.4fms", elapsedMsFloat)),
				zap.String("reqBody", reqBody),
				zap.String("respBody", respBody),
			)
		}
	}
}

// 捕获 panic 并记录错误
func RecoveryWithZap(logger *zap.Logger, stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				// 识别 broken pipe
				var brokenPipe bool
				if ne, ok := rec.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						errStr := strings.ToLower(se.Error())
						if strings.Contains(errStr, "broken pipe") ||
							strings.Contains(errStr, "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false)

				if brokenPipe {
					logger.Error("broken pipe",
						zap.Any("error", rec),
						zap.ByteString("request", httpRequest),
					)
					c.Error(rec.(error)) // 加入 gin context
					c.Abort()
					return
				}

				fields := []zap.Field{
					zap.Time("time", time.Now()),
					zap.Any("error", rec),
					zap.ByteString("request", httpRequest),
				}
				if stack {
					fields = append(fields, zap.ByteString("stack", debug.Stack()))
				}

				logger.Error("panic recovery", fields...)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}

// 脱敏处理
func maskSensitive(jsonStr string, sensitiveKeys []string) string {
	if len(jsonStr) == 0 {
		return jsonStr
	}

	// 尝试解析为 JSON
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		// 不是 JSON，直接返回原始
		return jsonStr
	}

	// 递归处理
	maskMap(data, sensitiveKeys)

	marshaled, _ := json.Marshal(data)
	return string(marshaled)
}

// 递归脱敏数据
func maskMap(v interface{}, sensitiveKeys []string) {
	switch val := v.(type) {
	case map[string]interface{}:
		for k, v2 := range val {
			if isSensitive(k, sensitiveKeys) {
				// 如果键是敏感的，直接脱敏整个值
				val[k] = "***"
			} else {
				// 键不敏感，但值可能是结构体/数组，继续递归处理
				maskMap(v2, sensitiveKeys)
			}
		}
	case []interface{}:
		for _, v2 := range val {
			// 递归处理数组中的每个元素
			maskMap(v2, sensitiveKeys)
		}
	// 增加对基本类型的 case，避免不必要的递归
	case nil, bool, string, float64:
		// 基本类型不需要处理
		return
	}
}

func isSensitive(key string, sensitiveKeys []string) bool {
	keyLower := strings.ToLower(key)
	for _, sk := range sensitiveKeys {
		if keyLower == strings.ToLower(sk) {
			return true
		}
	}
	return false
}

func isJSONContentType(ct string) bool {
	if ct == "" {
		return false
	}
	ct = strings.ToLower(strings.TrimSpace(ct))
	// 去掉 charset 等参数
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	if ct == "application/json" || ct == "text/json" {
		return true
	}
	// vendor types like application/ld+json, application/problem+json
	if strings.HasPrefix(ct, "application/") && strings.HasSuffix(ct, "+json") {
		return true
	}
	return false
}

func looksLikeJSONBody(b []byte) bool {
	for _, ch := range b {
		if unicode.IsSpace(rune(ch)) {
			continue
		}
		return ch == '{' || ch == '['
	}
	return false
}

func shouldLogReqBody(cfg logConfig, contentType string, peekBody []byte) bool {
	if !cfg.onlyJSONBody {
		return true
	}
	if isJSONContentType(contentType) {
		return true
	}
	if cfg.guessJSON && len(peekBody) > 0 {
		return looksLikeJSONBody(peekBody)
	}
	return false
}
