package middleware

import (
	"github.com/blocktransaction/zen/common/constant"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// 设置traceid
func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		spanCtx := trace.SpanContextFromContext(c.Request.Context())
		if spanCtx.HasTraceID() {
			traceId := spanCtx.TraceID().String()
			c.Writer.Header().Set("X-Trace-ID", traceId)
			c.Set(constant.TraceId, traceId)
		}
		c.Next()
	}
}
