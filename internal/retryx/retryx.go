package retryx

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// ---------------- Retrier ----------------

type Retrier[T any] struct {
	maxRetries     int
	maxElapsedTime time.Duration
	initialDelay   time.Duration
	maxDelay       time.Duration
	backoffFactor  float64
	jitterFactor   float64
	errorFilter    func(error) bool
	onRetry        func(err error, attempt int, nextDelay time.Duration)
	rand           *rand.Rand
}

type Option[T any] func(*Retrier[T])

func NewRetrier[T any](opts ...Option[T]) *Retrier[T] {
	r := &Retrier[T]{
		maxRetries:     5,
		maxElapsedTime: 30 * time.Second,
		initialDelay:   100 * time.Millisecond,
		maxDelay:       5 * time.Second,
		backoffFactor:  2.0,
		jitterFactor:   0.1,
		rand:           rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *Retrier[T]) Do(ctx context.Context, task func() (T, error)) (T, error) {
	var zero T
	start := time.Now()
	delay := r.initialDelay
	var lastErr error

	for attempt := 1; attempt <= r.maxRetries; attempt++ {
		// -------------------- 执行业务 --------------------
		result, err := task()
		if err == nil {
			return result, nil
		}
		lastErr = err

		// -------------------- 错误过滤 --------------------
		if r.errorFilter != nil && !r.errorFilter(err) {
			return zero, err
		}

		// -------------------- 终止条件 --------------------
		if attempt == r.maxRetries {
			return zero, fmt.Errorf("retry failed after %d attempts: %w", attempt, err)
		}
		if r.maxElapsedTime > 0 && time.Since(start) >= r.maxElapsedTime {
			return zero, fmt.Errorf("retry elapsed time exceeded: %w", err)
		}

		// -------------------- 计算下次延迟（带抖动） --------------------
		next := time.Duration(float64(delay) * r.backoffFactor)
		if next > r.maxDelay {
			next = r.maxDelay
		}
		if r.jitterFactor > 0 {
			j := 1 + r.rand.Float64()*r.jitterFactor // 增加 0%~jitterFactor
			next = time.Duration(float64(next) * j)
		}

		if r.onRetry != nil {
			r.onRetry(err, attempt, next)
		}

		// -------------------- 等待（带 ctx 控制） --------------------
		timer := time.NewTimer(next)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, fmt.Errorf("retry aborted: %w, last error: %v", ctx.Err(), lastErr)
		case <-timer.C:
		}

		delay = next
	}

	// 逻辑上不会走到这里
	return zero, lastErr
}

// --- Retrier Options ---
func WithMaxRetries[T any](n int) Option[T] {
	return func(r *Retrier[T]) { r.maxRetries = n }
}
func WithInitialDelay[T any](d time.Duration) Option[T] {
	return func(r *Retrier[T]) { r.initialDelay = d }
}
func WithMaxDelay[T any](d time.Duration) Option[T] {
	return func(r *Retrier[T]) { r.maxDelay = d }
}
func WithMaxElapsedTime[T any](d time.Duration) Option[T] {
	return func(r *Retrier[T]) { r.maxElapsedTime = d }
}
func WithErrorFilter[T any](f func(error) bool) Option[T] {
	return func(r *Retrier[T]) { r.errorFilter = f }
}
func WithOnRetry[T any](f func(err error, attempt int, nextDelay time.Duration)) Option[T] {
	return func(r *Retrier[T]) { r.onRetry = f }
}
func WithRandSource[T any](src rand.Source) Option[T] {
	return func(r *Retrier[T]) { r.rand = rand.New(src) }
}
