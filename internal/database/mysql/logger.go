package mysql

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/blocktransaction/zen/internal/database"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

const defaultMysqlLog = "mysql.log"

// 正则脱敏
var sensitivePattern = regexp.MustCompile(`(?i)(password|token|secret|mobile|phone)\s*=\s*'[^']*'`)

type LogConfig struct {
	logger.Config
	LogFile          string
	Rotate           bool // 是否启用日志轮转
	MaxSize          int  // MB
	MaxBackups       int
	MaxAge           int // days
	Compress         bool
	AdditionalLogger Loggers
	EnableMasking    bool // 是否开启参数脱敏
}

type Loggers []*log.Logger

// FileLogger 实现了 gorm logger.Interface
type FileLogger struct {
	LogConfig
	Loggers                             Loggers
	infoStr, warnStr, errStr            string
	traceStr, traceErrStr, traceWarnStr string
}

func applyDefaults(cfg *LogConfig) {
	if cfg.LogFile == "" {
		cfg.LogFile = defaultMysqlLog
	}
	if cfg.Rotate {
		if cfg.MaxSize == 0 {
			cfg.MaxSize = 100
		}
		if cfg.MaxBackups == 0 {
			cfg.MaxBackups = 7
		}
		if cfg.MaxAge == 0 {
			cfg.MaxAge = 30
		}
		if !cfg.Compress {
			cfg.Compress = true
		}
	}
}

// 创建 FileLogger
func NewLogger(config LogConfig) *FileLogger {
	applyDefaults(&config)

	loggers := make([]*log.Logger, 0)

	// stdout
	stdout := os.NewFile(uintptr(syscall.Stdout), "/dev/stdout")
	consoleLogger := log.New(stdout, "", log.LstdFlags)
	loggers = append(loggers, consoleLogger)

	// 确保目录存在
	logDir := filepath.Dir(config.LogFile)
	if logDir != "." {
		_ = os.MkdirAll(logDir, 0755)
	}

	// 文件输出
	var fileOutput *log.Logger
	if config.Rotate {
		rotater := &lumberjack.Logger{
			Filename:   config.LogFile,
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   config.Compress,
		}
		fileOutput = log.New(rotater, "", log.LstdFlags)
	} else {
		logfile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY|os.O_SYNC, 0644)
		if err != nil {
			panic(fmt.Sprintf("open logfile error: %v", err))
		}
		fileOutput = log.New(logfile, "", log.LstdFlags)
	}
	loggers = append(loggers, fileOutput)

	// 附加 logger
	if len(config.AdditionalLogger) > 0 {
		loggers = append(loggers, config.AdditionalLogger...)
	}

	return &FileLogger{
		LogConfig:    config,
		Loggers:      loggers,
		infoStr:      "[INFO] caller：%s | traceId：%s | %s",
		warnStr:      "[WARN] caller：%s | traceId：%s | %s",
		errStr:       "[ERROR] caller：%s | traceId：%s| %s",
		traceStr:     "[SQL] caller：%s | traceId：%s | latency：%.4fms | rows：%v | sql：%s",
		traceWarnStr: "[SQL-WARN] caller：%s | traceId：%s | latency：%.4fms | rows：v | error：%s | sql：%s",
		traceErrStr:  "[SQL-ERR] caller：%s | traceId：%s | latency：%.4fms | rows：%v | error： %s | sql：%s",
	}
}

// --- 通用方法 ---

// 参数脱敏
func (l *FileLogger) maskSQL(sql string) string {
	if l.EnableMasking {
		return sensitivePattern.ReplaceAllString(sql, "$1='***'")
	}
	return sql
}

func (l *FileLogger) printf(ctx context.Context, msg string, data ...interface{}) {
	traceId := database.ExtractTraceID(ctx)
	lineFile := utils.FileWithLineNum()

	args := append([]interface{}{lineFile[strings.LastIndex(lineFile, "/"):], traceId}, data...)
	for _, logger := range l.Loggers {
		logger.Printf(msg, args...)
	}
}

// --- gorm.Logger 接口 ---

func (l *FileLogger) LogMode(level logger.LogLevel) logger.Interface {
	l.LogLevel = level
	return l
}

func (l *FileLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.printf(ctx, l.infoStr, append([]interface{}{msg}, data...)...)
}

func (l *FileLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.printf(ctx, l.warnStr, append([]interface{}{msg}, data...)...)
}

func (l *FileLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.printf(ctx, l.errStr, append([]interface{}{msg}, data...)...)
}

// Trace 打印 SQL
func (l *FileLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}
	elapsed := time.Since(begin)
	ms := float64(elapsed.Nanoseconds()) / 1e6

	switch {
	case err != nil && l.LogLevel >= logger.Error &&
		(!errors.Is(err, logger.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):

		sql, rows := fc()
		l.printf(ctx, l.traceErrStr, ms, rows, err, l.maskSQL(sql))

	case l.SlowThreshold != 0 && elapsed > l.SlowThreshold && l.LogLevel >= logger.Warn:
		sql, rows := fc()
		l.printf(ctx, l.traceWarnStr, ms, rows, "SLOW SQL", l.maskSQL(sql))

	case l.LogLevel == logger.Info:
		sql, rows := fc()
		l.printf(ctx, l.traceStr, ms, rows, l.maskSQL(sql))
	}
}

// --- 业务日志接口（方便手动调用） ---

func (l *FileLogger) Infof(ctx context.Context, format string, args ...interface{}) {
	l.printf(ctx, l.infoStr, fmt.Sprintf(format, args...))
}

func (l *FileLogger) Errorf(ctx context.Context, format string, args ...interface{}) {
	l.printf(ctx, l.errStr, fmt.Sprintf(format, args...))
}

func (l *FileLogger) Warnf(ctx context.Context, format string, args ...interface{}) {
	l.printf(ctx, l.warnStr, fmt.Sprintf(format, args...))
}
