package gormx

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync/atomic"
	"time"

	"github.com/blocktransaction/zen/internal/database"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/gorm/logger"
)

//
// =======================
//    BatchWriteSyncer
// =======================
//

type BatchWriteSyncer struct {
	writer    zapcore.WriteSyncer
	buf       []byte
	capacity  int
	flushSize int
	ch        chan []byte
	closing   int32
}

func NewBatchWriteSyncer(w zapcore.WriteSyncer, capacity, flushSize int) *BatchWriteSyncer {
	b := &BatchWriteSyncer{
		writer:    w,
		capacity:  capacity,
		flushSize: flushSize,
		ch:        make(chan []byte, 1024),
	}
	go b.loop()
	return b
}

func (b *BatchWriteSyncer) loop() {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case data, ok := <-b.ch:
			if !ok {
				b.doFlush()
				return
			}
			b.buf = append(b.buf, data...)
			if len(b.buf) >= b.flushSize {
				b.doFlush()
			}
		case <-ticker.C:
			if len(b.buf) > 0 {
				b.doFlush()
			}
		}
	}
}

func (b *BatchWriteSyncer) doFlush() {
	if len(b.buf) == 0 {
		return
	}
	_, _ = b.writer.Write(b.buf)
	b.buf = b.buf[:0]
}

func (b *BatchWriteSyncer) Write(p []byte) (int, error) {
	cp := make([]byte, len(p))
	copy(cp, p)
	b.ch <- cp
	return len(p), nil
}

func (b *BatchWriteSyncer) Sync() error {
	if atomic.CompareAndSwapInt32(&b.closing, 0, 1) {
		close(b.ch)
	}
	return b.writer.Sync()
}

//
// =======================
//    Config
// =======================
//

type Config struct {
	Level         string
	File          string
	Rotate        bool
	MaxSize       int
	MaxAge        int
	MaxBackups    int
	Compress      bool
	EnableMasking bool
	SlowThreshold time.Duration
}

var sensitivePattern = regexp.MustCompile(`(?i)(password|token|secret|mobile|phone)\s*=\s*'[^']*'`)

func maskSQL(sql string) string {
	return sensitivePattern.ReplaceAllString(sql, `$1='***'`)
}

//
// =======================
//    Main Logger
// =======================
//

func NewLogger(cfg Config) (*zap.Logger, *GormLogger) {
	level := zapcore.InfoLevel
	_ = level.Set(cfg.Level)

	// Encoder (JSON for production)
	encCfg := zapcore.EncoderConfig{
		TimeKey:        "@timestamp",
		LevelKey:       "level",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stack",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.RFC3339TimeEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
	}
	encoder := zapcore.NewJSONEncoder(encCfg)

	// Ensure log dir
	_ = os.MkdirAll(filepath.Dir(cfg.File), 0755)

	// file writer
	var fileWriter zapcore.WriteSyncer
	if cfg.Rotate {
		fileWriter = zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		})
	} else {
		f, _ := os.OpenFile(cfg.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY|os.O_SYNC, 0644)
		fileWriter = zapcore.AddSync(f)
	}

	// batch wrappers
	fileBatch := NewBatchWriteSyncer(fileWriter, 4*1024*1024, 512*1024)
	consoleBatch := NewBatchWriteSyncer(zapcore.AddSync(os.Stdout), 2*1024*1024, 64*1024)

	multiWS := zapcore.NewMultiWriteSyncer(fileBatch, consoleBatch)

	core := zapcore.NewCore(encoder, multiWS, level)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))

	// GORM Logger
	gormLogger := NewGormLogger(logger, cfg)

	return logger, gormLogger
}

//
// =======================
//    GORM Logger
// =======================
//

type GormLogger struct {
	Zap           *zap.Logger
	SlowThreshold time.Duration
	EnableMasking bool
	LogLevel      logger.LogLevel
}

func NewGormLogger(z *zap.Logger, cfg Config) *GormLogger {
	return &GormLogger{
		Zap:           z,
		SlowThreshold: cfg.SlowThreshold,
		EnableMasking: cfg.EnableMasking,
		LogLevel:      logger.Info,
	}
}

func (l *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
	l.LogLevel = level
	return l
}

func (l *GormLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	l.Zap.Sugar().Infow(msg, args...)
}

func (l *GormLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	l.Zap.Sugar().Warnw(msg, args...)
}

func (l *GormLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	l.Zap.Sugar().Errorw(msg, args...)
}

func (l *GormLogger) traceID(ctx context.Context) string {
	return database.ExtractTraceID(ctx)
}

func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	if l.EnableMasking {
		sql = maskSQL(sql)
	}

	traceID := l.traceID(ctx)
	elapsedMsFloat := float64(time.Since(begin)) / float64(time.Millisecond)
	latency := fmt.Sprintf("%.4fms", elapsedMsFloat)
	switch {
	case err != nil && l.LogLevel >= logger.Error:
		l.Zap.Error("SQL ERROR",
			zap.String("traceId", traceID),
			zap.String("latency", latency),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
			zap.Error(err),
		)

	case l.SlowThreshold > 0 && elapsed > l.SlowThreshold:
		l.Zap.Warn("SLOW SQL",
			zap.String("traceId", traceID),
			zap.String("latency", latency),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
		)

	case l.LogLevel == logger.Info:
		l.Zap.Info("SQL",
			zap.String("traceId", traceID),
			zap.String("latency", latency),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
		)
	}
}
