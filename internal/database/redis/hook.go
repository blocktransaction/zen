package redis

import (
	"context"
	"net"
	"time"

	"github.com/blocktransaction/zen/internal/database"
	"github.com/redis/go-redis/v9"
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
		fields := map[string]interface{}{
			"traceId": l.traceID(ctx),
			"network": network,
			"addr":    addr,
			"latency": time.Since(start).String(),
		}
		if err != nil {
			fields["error"] = err
			l.logger.Error("Redis dial failed", fields)
		} else {
			l.logger.Info("Redis dial success", fields)
		}
		return conn, err
	}
}

// 单条命令日志
func (l *RedisLogger) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmd)
		fields := map[string]interface{}{
			"traceId": l.traceID(ctx),
			"cmd":     cmd.Name(),
			"args":    cmd.Args(),
			"latency": time.Since(start).String(),
		}

		if l.logResult {
			fields["result"] = cmd.String()
		}

		if err != nil {
			fields["error"] = err
			l.logger.Error("Redis command failed", fields)
		} else {
			l.logger.Info("Redis command executed", fields)
		}
		return err
	}
}

// pipeline 日志
func (l *RedisLogger) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmds)

		results := make([]string, 0, len(cmds))
		errCount := 0
		if l.logResult {
			for _, cmd := range cmds {
				results = append(results, cmd.String())
				if cmd.Err() != nil {
					errCount++
				}
			}
		}

		fields := map[string]interface{}{
			"traceId":    l.traceID(ctx),
			"cmds_count": len(cmds),
			"failed":     errCount,
			"latency":    time.Since(start).String(),
		}

		if l.logResult {
			fields["results"] = results
		}

		if errCount > 0 {
			fields["failed"] = errCount
		}

		if err != nil {
			fields["error"] = err
			l.logger.Error("Redis pipeline failed", fields)
		} else {
			l.logger.Info("Redis pipeline executed", fields)
		}
		return err
	}
}
