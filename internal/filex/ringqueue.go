package filex

import (
	"runtime"
	"sync/atomic"
	"unsafe"
)

// MPMC ring buffer (Dmitry Vyukov bounded MPMC queue inspired)
// Stored type: string (file path). You can adapt to any type by changing item type.

type ringSlot struct {
	seq  uint64         // sequence for this slot
	data unsafe.Pointer // *string stored
}

type RingQueue struct {
	mask  uint64
	size  uint64
	slots []ringSlot
	// head and tail index (monotonic counters)
	head uint64 // producer index (next to produce)
	tail uint64 // consumer index (next to consume)
}

// NewRingQueue creates a ring queue with capacity >= n and power-of-two size.
func NewRingQueue(n int) *RingQueue {
	// make it power of two
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
			// try claim by CAS head -> head+1
			if atomic.CompareAndSwapUint64(&q.head, pos, pos+1) {
				// store data pointer
				ptr := unsafe.Pointer(&s)
				atomic.StorePointer(&slot.data, ptr)
				// set seq = pos+1 to mark ready (producer finished)
				atomic.StoreUint64(&slot.seq, pos+1)
				return true
			}
			// CAS failed: retry
			runtime.Gosched()
			continue
		}
		// if seq < pos => slot not yet consumed (full)
		if seq < pos {
			return false
		}
		// otherwise spin a bit
		runtime.Gosched()
		idx++
		if idx > 8 {
			runtime.Gosched()
			idx = 0
		}
	}
}

// Poll pops value from queue. Non-blocking; returns (value,true) on success, ("",false) if empty.
func (q *RingQueue) Poll() (string, bool) {
	var pos, idx uint64
	for {
		pos = atomic.LoadUint64(&q.tail)
		slot := &q.slots[pos&q.mask]
		seq := atomic.LoadUint64(&slot.seq)

		if seq == pos+1 {
			// try claim by CAS tail -> tail+1
			if atomic.CompareAndSwapUint64(&q.tail, pos, pos+1) {
				// load pointer
				ptr := atomic.LoadPointer(&slot.data)
				var val string
				if ptr != nil {
					// copy string value to local (because original may be from caller stack)
					sp := (*string)(ptr)
					val = *sp
				}
				// mark slot as free: set seq = pos + size
				atomic.StoreUint64(&slot.seq, pos+q.size)
				// clear data pointer (avoid memory leak)
				atomic.StorePointer(&slot.data, nil)
				return val, true
			}
			runtime.Gosched()
			continue
		}
		// empty if seq < pos+1
		if seq < pos+1 {
			return "", false
		}
		// otherwise spin
		runtime.Gosched()
		idx++
		if idx > 8 {
			runtime.Gosched()
			idx = 0
		}
	}
}
