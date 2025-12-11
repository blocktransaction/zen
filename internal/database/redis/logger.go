package redis

import (
	"go.uber.org/zap"
)

// Logger 统一日志接口
type Logger interface {
	Info(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

// ----------------- zap 适配 -----------------

type ZapAdapter struct {
	logger *zap.Logger
}

func NewZapAdapter(logger *zap.Logger) *ZapAdapter {
	return &ZapAdapter{logger: logger}
}

func (z *ZapAdapter) Info(msg string, fields ...zap.Field) {
	z.logger.Info(msg, fields...)
}

func (z *ZapAdapter) Error(msg string, fields ...zap.Field) {
	z.logger.Error(msg, fields...)
}
