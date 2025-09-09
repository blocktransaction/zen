package database

import (
	"context"

	"github.com/blocktransaction/zen/common/constant"
)

// 设置 traceId
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, constant.TraceIdKey, traceID)
}

// 提取 traceId
func ExtractTraceID(ctx context.Context) string {
	if v := ctx.Value(constant.TraceIdKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
