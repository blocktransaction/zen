package logx

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 特性总结
// 1、无锁环形缓冲区 (RingBuffer)：用于高吞吐量写入。
// 2、MPSC 安全性：Push 中使用了 CAS 循环来支持多生产者。
// 3、避免伪共享：使用了 cacheLinePad。
// 4、精确的 Caller 捕获：解决了异步日志的调用方错位问题。
// 5、高效的消费者退避：run 方法避免了 CPU 空转。
// 6、GC 友好：Pop 中清空了槽位。
type AsyncLogData struct {
	Level      zapcore.Level
	Msg        string
	ErrWrapper *ErrorWrapper
	Fields     []zap.Field
	Caller     zapcore.EntryCaller // **新增**
}

type AsyncLogger struct {
	rb     *RingBuffer
	logger *zap.Logger
	closed atomic.Bool
	wg     sync.WaitGroup // **新增**
}

// cacheLinePad 用于填充，确保 write 和 read 字段不共享缓存行
type cacheLinePad [8]uint64
type RingBuffer struct {
	size uint64
	mask uint64
	// 确保 write 索引位于独立缓存行
	_     cacheLinePad
	write uint64 // atomic, MPSC 必须使用 CAS 保护
	_     cacheLinePad
	read  uint64 // atomic, SPSC 模型下 run() 中使用是安全的
	_     cacheLinePad
	data  []AsyncLogData
}

func NewRingBuffer(n uint64) *RingBuffer {
	// n 必须是 2^k
	size := uint64(1)
	for size < n {
		size <<= 1
	}

	return &RingBuffer{
		size: size,
		mask: size - 1,
		data: make([]AsyncLogData, size),
	}
}

func (rb *RingBuffer) Push(v AsyncLogData) bool {
	spin := 0
	for {
		w := atomic.LoadUint64(&rb.write)
		r := atomic.LoadUint64(&rb.read)

		// ring 满（留一个槽区分 full / empty）
		if w-r >= rb.size-1 {
			return false
		}

		// --- 优化点：CAS失败时的自旋退避 ---
		if spin > 1000 { // 适当的阈值
			runtime.Gosched()
		}
		spin++
		rb.data[w&rb.mask] = v

		// 尝试原子性更新 write 索引
		if atomic.CompareAndSwapUint64(&rb.write, w, w+1) {
			return true // 写入成功
		}
		// CAS 失败，继续循环重试
	}
}

// 修正后的 Pop (清空槽位)
func (rb *RingBuffer) Pop(v *AsyncLogData) bool {
	r := atomic.LoadUint64(&rb.read)
	w := atomic.LoadUint64(&rb.write)

	if r == w {
		return false
	}

	index := r & rb.mask

	*v = rb.data[index]

	// **清空槽位，帮助 GC 回收底层对象的内存**
	rb.data[index] = AsyncLogData{}

	atomic.StoreUint64(&rb.read, r+1)
	return true
}

func NewAsyncLogger(z *zap.Logger, buffer uint64) *AsyncLogger {
	a := &AsyncLogger{
		logger: z,
		rb:     NewRingBuffer(buffer),
	}
	a.wg.Add(1) // **新增**
	go a.run()
	return a
}

// 运行
func (a *AsyncLogger) run() {
	defer a.wg.Done() // **新增**

	var item AsyncLogData
	spin := 0            // 计数器，用于控制退避策略
	const maxSpin = 1000 // 最大自旋次数
	for {
		// 1. 优先消费队列
		for a.rb.Pop(&item) {
			a.write(item)
			spin = 0 // 成功消费，重置自旋计数
		}

		// 2. 队列为空时的退出条件
		if a.closed.Load() {
			// 确保队列确实已经清空
			if !a.rb.Pop(&item) {
				return // 队列已空且已关闭，安全退出
			}
			// 如果 pop 成功，则写入，继续消费，不退出
			a.write(item)
			continue
		}

		// 3. 队列为空时的退避策略 (取代简单的 Gosched)
		if spin < maxSpin {
			spin++
			// 短暂自旋，通常比 Gosched 更高效
			runtime.Gosched()
		} else {
			// 长时间无数据，进行短暂休眠，彻底释放 CPU
			time.Sleep(time.Microsecond)
		}
	}
}

// 写操作
func (a *AsyncLogger) write(data AsyncLogData) {
	n := len(data.Fields)
	if data.ErrWrapper != nil {
		n += 3
	}
	fs := make([]zap.Field, 0, n)

	fs = append(fs, data.Fields...)

	if data.ErrWrapper != nil {
		fs = append(fs,
			zap.String("err", data.ErrWrapper.Err.Error()),
			zap.Int("code", data.ErrWrapper.Code),
			zap.String("stack", data.ErrWrapper.Stack),
		)
		if data.ErrWrapper.Metadata != nil {
			fs = append(fs, zap.Any("metadata", data.ErrWrapper.Metadata))
		}
	}

	if ce := a.logger.Check(data.Level, data.Msg); ce != nil {
		// **应用手动捕获的 Caller**
		if data.Caller.Defined {
			ce.Caller = data.Caller
		}
		ce.Write(fs...)
	}
}

// Log 方法 (捕获 Caller)
func (a *AsyncLogger) Log(level zapcore.Level, msg string,
	errWrapper *ErrorWrapper, fields ...zap.Field) {

	// 捕获调用方信息：跳过 Log() 和 runtime.Caller 所在的帧
	caller := zapcore.NewEntryCaller(runtime.Caller(1))
	entry := AsyncLogData{
		Level:      level,
		Msg:        msg,
		ErrWrapper: errWrapper,
		Fields:     fields,
		Caller:     caller, // **传递 Caller**
	}

	if !a.rb.Push(entry) {
		a.fallback(entry)
	}
}

func (a *AsyncLogger) fallback(data AsyncLogData) {
	// 1. 记录通道满的警告（同步写入）
	a.logger.Warn("Async log ring buffer full, logging synchronously.",
		zap.String("original_msg", data.Msg),
		zap.Uint64("buffer_size", a.rb.size),
		zap.Int("level", int(data.Level)),
	)
	// 2. 执行原始日志的同步写入
	a.write(data)
}

func (a *AsyncLogger) Close() {
	// a.closed.Store(true)
	// 1. 标记关闭
	if a.closed.CompareAndSwap(false, true) {
		// 2. 等待 run Goroutine 退出，确保所有排队的日志都被写入
		a.wg.Wait()
		// 确保 zap Logger 的 Core 被同步（例如，将日志写入文件）
		// 如果 zap Logger 配置了自动同步，这步可选
		a.logger.Sync()
	}
}
