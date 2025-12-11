package filex

import (
	"runtime"
	"sync/atomic"
	"unsafe"
)

type ringSlot struct {
	seq  uint64         // sequence for this slot
	data unsafe.Pointer // *string stored
}

type RingQueue struct {
	mask  uint64
	size  uint64
	slots []ringSlot
	head  uint64 // producer index (next to produce)
	tail  uint64 // consumer index (next to consume)
}

// 初始化 固定容量（通常为 2^n）
func NewRingQueue(n int) *RingQueue {
	cap := uint64(1)
	for cap < uint64(n) {
		cap <<= 1
	}
	slots := make([]ringSlot, cap)
	// initialize sequence numbers
	for i := range slots {
		slots[i].seq = uint64(i)
	}
	return &RingQueue{
		mask:  cap - 1,
		size:  cap,
		slots: slots,
	}
}

func (q *RingQueue) Offer(s string) bool {
	var pos, idx uint64
	for {
		pos = atomic.LoadUint64(&q.head)
		slot := &q.slots[pos&q.mask]
		seq := atomic.LoadUint64(&slot.seq)
		if seq == pos {
			if atomic.CompareAndSwapUint64(&q.head, pos, pos+1) {
				ptr := unsafe.Pointer(&s)
				atomic.StorePointer(&slot.data, ptr)
				atomic.StoreUint64(&slot.seq, pos+1)
				return true
			}
			runtime.Gosched()
			continue
		}
		if seq < pos {
			return false
		}
		runtime.Gosched()
		idx++
		if idx > 8 {
			runtime.Gosched()
			idx = 0
		}
	}
}

func (q *RingQueue) Poll() (string, bool) {
	var pos, idx uint64
	for {
		pos = atomic.LoadUint64(&q.tail)
		slot := &q.slots[pos&q.mask]
		seq := atomic.LoadUint64(&slot.seq)

		if seq == pos+1 {
			if atomic.CompareAndSwapUint64(&q.tail, pos, pos+1) {
				ptr := atomic.LoadPointer(&slot.data)
				var val string
				if ptr != nil {
					sp := (*string)(ptr)
					val = *sp
				}
				atomic.StoreUint64(&slot.seq, pos+q.size)
				atomic.StorePointer(&slot.data, nil)
				return val, true
			}
			runtime.Gosched()
			continue
		}

		if seq < pos+1 {
			return "", false
		}
		runtime.Gosched()
		idx++
		if idx > 8 {
			runtime.Gosched()
			idx = 0
		}
	}
}
