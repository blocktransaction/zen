package redis

import (
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

// Logger 统一日志接口
type Logger interface {
	Info(msg string, fields map[string]interface{})
	Error(msg string, fields map[string]interface{})
}

// ----------------- zap 适配 -----------------

type ZapAdapter struct {
	logger *zap.Logger
}

func NewZapAdapter(logger *zap.Logger) *ZapAdapter {
	return &ZapAdapter{logger: logger}
}

func (z *ZapAdapter) Info(msg string, fields map[string]interface{}) {
	z.logger.Info(msg, toZapFields(fields)...)
}

func (z *ZapAdapter) Error(msg string, fields map[string]interface{}) {
	z.logger.Error(msg, toZapFields(fields)...)
}

func toZapFields(fields map[string]interface{}) []zap.Field {
	zf := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zf = append(zf, zap.Any(k, v))
	}
	return zf
}

// ----------------- logrus 适配 -----------------

type LogrusAdapter struct {
	logger *logrus.Logger
}

func NewLogrusAdapter(logger *logrus.Logger) *LogrusAdapter {
	return &LogrusAdapter{logger: logger}
}

func (l *LogrusAdapter) Info(msg string, fields map[string]interface{}) {
	l.logger.WithFields(fields).Info(msg)
}

func (l *LogrusAdapter) Error(msg string, fields map[string]interface{}) {
	l.logger.WithFields(fields).Error(msg)
}
