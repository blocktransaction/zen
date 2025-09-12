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

// Do executes task with retry/backoff.
func (r *Retrier[T]) Do(ctx context.Context, task func() (T, error)) (T, error) {
	var zero T
	var lastErr error
	start := time.Now()
	delay := r.initialDelay

	for attempt := 1; ; attempt++ {
		result, err := task()
		if err == nil {
			return result, nil
		}
		lastErr = err

		if r.errorFilter != nil && !r.errorFilter(err) {
			return zero, err
		}

		if attempt >= r.maxRetries {
			lastErr = fmt.Errorf("retry failed after %d attempts: %w", r.maxRetries, err)
			break
		}
		if r.maxElapsedTime > 0 && time.Since(start) >= r.maxElapsedTime {
			lastErr = fmt.Errorf("retry failed due to elapsed time limit: %w", err)
			break
		}

		// backoff + jitter
		nextDelay := time.Duration(float64(delay) * r.backoffFactor)
		if nextDelay > r.maxDelay {
			nextDelay = r.maxDelay
		}
		if r.jitterFactor > 0 {
			jitter := r.rand.Float64() * r.jitterFactor // [0, jitterFactor]
			nextDelay += time.Duration(float64(nextDelay) * jitter)
		}
		delay = nextDelay

		if r.onRetry != nil {
			r.onRetry(err, attempt, delay)
		}

		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(delay):
		}
	}
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
