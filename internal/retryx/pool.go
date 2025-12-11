package retryx

import (
	"context"
	"sync"
)

// ---------------- Pool ----------------
type Task[T any] struct {
	Fn      func() (T, error)
	Retrier *Retrier[T]
	Ctx     context.Context // optional task-level ctx
}

type Result[T any] struct {
	Value T
	Err   error
}

type Future[T any] struct {
	once       sync.Once
	result     Result[T]
	resultChan <-chan Result[T]
}

func (f *Future[T]) Get() (T, error) {
	f.once.Do(func() {
		f.result = <-f.resultChan
	})
	return f.result.Value, f.result.Err
}

func (f *Future[T]) GetContext(ctx context.Context) (T, error) {
	var zero T
	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case r := <-f.resultChan:
		f.once.Do(func() { f.result = r })
		return r.Value, r.Err
	}
}

func (f *Future[T]) TryGet() (value T, err error, ok bool) {
	select {
	case r := <-f.resultChan:
		f.once.Do(func() { f.result = r })
		return f.result.Value, f.result.Err, true
	default:
		var zero T
		return zero, nil, false
	}
}

func (f *Future[T]) IsDone() bool {
	_, _, ok := f.TryGet()
	return ok
}

type Pool[T any] struct {
	workerCount int
	tasks       chan poolTask[T]
	wg          sync.WaitGroup

	ctx    context.Context
	cancel context.CancelFunc

	mu     sync.Mutex
	closed bool

	replaceWorkerOnPanic bool
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
		workerCount:          workerCount,
		tasks:                make(chan poolTask[T], workerCount*2),
		ctx:                  ctx,
		cancel:               cancel,
		replaceWorkerOnPanic: true,
	}
	p.start()
	return p
}

func (p *Pool[T]) start() {
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

func (p *Pool[T]) worker() {
	defer func() {
		if r := recover(); r != nil && p.replaceWorkerOnPanic {
			// 自愈：保持 worker 数量不变
			go p.worker()
		}
		p.wg.Done()
	}()

	for {
		select {
		case <-p.ctx.Done():
			return
		case pt, ok := <-p.tasks:
			if !ok {
				return
			}

			// 使用 task 自己的 ctx 或 pool 的 ctx
			ctx := pt.task.Ctx
			if ctx == nil {
				ctx = p.ctx
			}

			r := pt.task.Retrier
			if r == nil {
				r = NewRetrier[T]()
			}

			value, err := r.Do(ctx, pt.task.Fn)

			// 安全发送结果
			select {
			case pt.resultChan <- Result[T]{Value: value, Err: err}:
			default:
			}
			close(pt.resultChan)
		}
	}
}

func (p *Pool[T]) Submit(task Task[T]) *Future[T] {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		panic("submit on closed pool")
	}
	p.mu.Unlock()

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

	p.cancel()
	p.wg.Wait()
}
