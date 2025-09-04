package mysql

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	// 日志级别
	Silent logger.LogLevel = iota + 1
	Error
	Warn
	Info
)

const DEFAULT_LOGFILE = "mysql.log"

type LogConfig struct {
	logger.Config
	LogFile          string
	Rotate           bool // 是否启用日志轮转
	MaxSize          int  // MB
	MaxBackups       int
	MaxAge           int // days
	Compress         bool
	AdditionalLogger Loggers
}

type Loggers []*log.Logger

// FileLogger 文件日志器
type FileLogger struct {
	LogConfig
	Loggers                             Loggers
	infoStr, warnStr, errStr            string
	traceStr, traceErrStr, traceWarnStr string
}

// 设置默认值
func applyDefaults(cfg *LogConfig) {
	if cfg.LogFile == "" {
		cfg.LogFile = DEFAULT_LOGFILE
	}
	if cfg.Rotate {
		if cfg.MaxSize == 0 {
			cfg.MaxSize = 100 // 默认 100 MB
		}
		if cfg.MaxBackups == 0 {
			cfg.MaxBackups = 7
		}
		if cfg.MaxAge == 0 {
			cfg.MaxAge = 30 // 默认 30 天
		}
		// 默认开启压缩
		if !cfg.Compress {
			cfg.Compress = true
		}
	}
}

// NewLogger 创建 FileLogger
func NewLogger(config LogConfig) *FileLogger {
	applyDefaults(&config)

	loggers := make([]*log.Logger, 0)

	// stdout logger
	stdout := os.NewFile(uintptr(syscall.Stdout), "/dev/stdout")
	consoleLogger := log.New(stdout, "", log.LstdFlags)
	loggers = append(loggers, consoleLogger)

	// 确保目录存在
	logDir := filepath.Dir(config.LogFile)
	if logDir != "." {
		_ = os.MkdirAll(logDir, 0755)
	}

	// 文件 logger
	var fileOutput *log.Logger
	if config.Rotate {
		// 使用 lumberjack 日志轮转
		rotater := &lumberjack.Logger{
			Filename:   config.LogFile,
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   config.Compress,
		}
		fileOutput = log.New(rotater, "", log.LstdFlags)
	} else {
		// 普通文件日志
		logfile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY|os.O_SYNC, 0644)
		if err != nil {
			panic(fmt.Sprintf("open logfile error: %v", err))
		}
		fileOutput = log.New(logfile, "", log.LstdFlags)
	}
	loggers = append(loggers, fileOutput)

	// 附加日志器
	if len(config.AdditionalLogger) > 0 {
		loggers = append(loggers, config.AdditionalLogger...)
	}

	return &FileLogger{
		LogConfig:    config,
		Loggers:      loggers,
		infoStr:      "%s\n[info] ",
		warnStr:      "%s\n[warn] ",
		errStr:       "%s\n[error] ",
		traceStr:     "%s\n[%.3fms] [rows:%v] %s",
		traceWarnStr: "%s %s\n[%.3fms] [rows:%v] %s",
		traceErrStr:  "%s %s\n[%.3fms] [rows:%v] %s",
	}
}

func (l *FileLogger) printf(msg string, data ...interface{}) {
	for _, logger := range l.Loggers {
		logger.Printf(msg, data...)
	}
}

func (l *FileLogger) LogMode(level logger.LogLevel) logger.Interface {
	l.LogLevel = level
	return l
}

func (l *FileLogger) logf(level logger.LogLevel, tmpl string, msg string, data ...interface{}) {
	if l.LogLevel >= level {
		l.printf(tmpl+msg, append([]interface{}{utils.FileWithLineNum()}, data...)...)
	}
}

func (l *FileLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.logf(Info, l.infoStr, msg, data...)
}

func (l *FileLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.logf(Warn, l.warnStr, msg, data...)
}

func (l *FileLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.logf(Error, l.errStr, msg, data...)
}

// Trace 打印 SQL
func (l *FileLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= Silent {
		return
	}

	elapsed := time.Since(begin)
	ms := float64(elapsed.Nanoseconds()) / 1e6

	switch {
	case err != nil && l.LogLevel >= Error &&
		(!errors.Is(err, logger.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):

		sql, rows := fc()
		if rows == -1 {
			l.printf(l.traceErrStr, utils.FileWithLineNum(), err, ms, "-", sql)
		} else {
			l.printf(l.traceErrStr, utils.FileWithLineNum(), err, ms, rows, sql)
		}

	case l.SlowThreshold != 0 && elapsed > l.SlowThreshold && l.LogLevel >= Warn:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		if rows == -1 {
			l.printf(l.traceWarnStr, utils.FileWithLineNum(), slowLog, ms, "-", sql)
		} else {
			l.printf(l.traceWarnStr, utils.FileWithLineNum(), slowLog, ms, rows, sql)
		}

	case l.LogLevel == Info:
		sql, rows := fc()
		if rows == -1 {
			l.printf(l.traceStr, utils.FileWithLineNum(), ms, "-", sql)
		} else {
			l.printf(l.traceStr, utils.FileWithLineNum(), ms, rows, sql)
		}
	}
}
