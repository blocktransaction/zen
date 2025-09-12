package retryx

import (
	"context"
	"sync"
)

// ---------------- Pool ----------------
type Task[T any] struct {
	Fn      func() (T, error)
	Retrier *Retrier[T]
}

type Result[T any] struct {
	Value T
	Err   error
}

type Future[T any] struct {
	resultChan <-chan Result[T]
}

func (f *Future[T]) Get() (T, error) {
	res := <-f.resultChan
	return res.Value, res.Err
}

func (f *Future[T]) GetContext(ctx context.Context) (T, error) {
	var zero T
	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case res := <-f.resultChan:
		return res.Value, res.Err
	}
}

type Pool[T any] struct {
	workerCount int
	tasks       chan poolTask[T]
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	closed      bool
	mu          sync.Mutex
}

type poolTask[T any] struct {
	task       Task[T]
	resultChan chan<- Result[T]
}

func NewPool[T any](workerCount int) *Pool[T] {
	if workerCount <= 0 {
		workerCount = 10
	}
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool[T]{
		workerCount: workerCount,
		tasks:       make(chan poolTask[T]),
		ctx:         ctx,
		cancel:      cancel,
	}
	p.start()
	return p
}

func (p *Pool[T]) start() {
	p.wg.Add(p.workerCount)
	for i := 0; i < p.workerCount; i++ {
		go func() {
			defer p.wg.Done()
			for {
				select {
				case <-p.ctx.Done():
					return
				case task, ok := <-p.tasks:
					if !ok {
						return
					}
					retrier := task.task.Retrier
					if retrier == nil {
						retrier = NewRetrier[T]()
					}
					val, err := retrier.Do(p.ctx, task.task.Fn)
					task.resultChan <- Result[T]{Value: val, Err: err}
					close(task.resultChan)
				}
			}
		}()
	}
}

func (p *Pool[T]) Submit(task Task[T]) *Future[T] {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		panic("submit on closed pool")
	}
	resultChan := make(chan Result[T], 1)
	p.tasks <- poolTask[T]{task: task, resultChan: resultChan}
	return &Future[T]{resultChan: resultChan}
}

func (p *Pool[T]) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	close(p.tasks)
	p.mu.Unlock()

	p.wg.Wait()
	p.cancel()
}
