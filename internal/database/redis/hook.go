package redis

import (
	"context"
	"net"
	"time"

	"github.com/blocktransaction/zen/internal/database"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisLogger struct {
	logger    Logger
	logResult bool // 是否打印 Redis 命令结果
}

func NewRedisLogger(logger Logger, logResult bool) *RedisLogger {
	return &RedisLogger{
		logger:    logger,
		logResult: logResult,
	}
}

func (l *RedisLogger) traceID(ctx context.Context) string {
	return database.ExtractTraceID(ctx)
}

// 连接 Redis 时日志
func (l *RedisLogger) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		start := time.Now()
		conn, err := next(ctx, network, addr)

		traceID := l.traceID(ctx)

		fields := []zap.Field{
			zap.String("traceId", traceID),
			zap.String("network", network),
			zap.String("addr", addr),
			zap.Duration("latency", time.Since(start)),
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			l.logger.Error("Redis dial failed", fields...)
		} else {
			l.logger.Info("Redis dial success", fields...)
		}
		return conn, err
	}
}

// 单条命令日志
func (l *RedisLogger) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmd)
		traceID := l.traceID(ctx)

		// 预估字段数量，减少扩容
		fields := make([]zap.Field, 0, 6)
		fields = append(fields,
			zap.String("traceId", traceID),
			zap.String("cmd", cmd.Name()),
			zap.Any("args", cmd.Args()),
			zap.Duration("latency", time.Since(start)),
		)

		if l.logResult {
			fields = append(fields, zap.String("result", cmd.String()))
		}
		if err != nil {
			fields = append(fields, zap.Error(err))
			l.logger.Error("Redis command failed", fields...)
		} else {
			l.logger.Info("Redis command executed", fields...)
		}

		return err
	}
}

// pipeline 日志
func (l *RedisLogger) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {

		start := time.Now()
		err := next(ctx, cmds)
		traceID := l.traceID(ctx)

		var (
			errCount int
			results  []string
		)

		if l.logResult {
			results = make([]string, 0, len(cmds))
			for _, c := range cmds {
				results = append(results, c.String())
				if c.Err() != nil {
					errCount++
				}
			}
		} else {
			for _, c := range cmds {
				if c.Err() != nil {
					errCount++
				}
			}
		}

		fields := make([]zap.Field, 0, 6)
		fields = append(fields,
			zap.String("traceId", traceID),
			zap.Int("cmds_count", len(cmds)),
			zap.Int("failed", errCount),
			zap.Duration("latency", time.Since(start)),
		)

		if l.logResult {
			fields = append(fields, zap.Any("results", results))
		}
		if err != nil {
			fields = append(fields, zap.Error(err))
			l.logger.Error("Redis pipeline failed", fields...)
		} else {
			l.logger.Info("Redis pipeline executed", fields...)
		}
		return err
	}
}
