package fastredis

import (
	"errors"
	"sync/atomic"
	"time"
)

var ErrPoolFull = errors.New("fastredis: pool full")
var ErrNoConn = errors.New("fastredis: no connection available")

// ---- Lock-Free RingBuffer Pool ----

type Pool struct {
	conns   []*Client
	cap     uint32
	head    uint32 // pop index（CAS）
	tail    uint32 // push index（CAS）
	addr    string
	timeout time.Duration
}

// NewPool 创建池（固定大小）
func NewPool(addr string, size int, timeout time.Duration) (*Pool, error) {
	p := &Pool{
		conns:   make([]*Client, size),
		cap:     uint32(size),
		addr:    addr,
		timeout: timeout,
	}

	// 预建满所有连接（可选：懒加载）
	for i := 0; i < size; i++ {
		c, err := Dial(addr, timeout)
		if err != nil {
			return nil, err
		}
		p.conns[i] = c
	}

	// head = 0, tail = size 表示队列 full
	atomic.StoreUint32(&p.tail, uint32(size))

	return p, nil
}

/* -------------------------------------------------------
 * 非阻塞 Pop（取连接）
 * -----------------------------------------------------*/
func (p *Pool) Get() (*Client, error) {
	for {
		h := atomic.LoadUint32(&p.head)
		t := atomic.LoadUint32(&p.tail)

		if h == t { // 空
			return nil, ErrNoConn
		}

		index := h % p.cap

		conn := p.conns[index]
		if conn == nil {
			return nil, ErrNoConn
		}

		// CAS 移动 head
		if atomic.CompareAndSwapUint32(&p.head, h, h+1) {
			return conn, nil
		}
		// 失败则重试（无锁）
	}
}

/* -------------------------------------------------------
 * 非阻塞 Push（还连接）
 * -----------------------------------------------------*/
func (p *Pool) Put(c *Client) error {
	for {
		h := atomic.LoadUint32(&p.head)
		t := atomic.LoadUint32(&p.tail)

		if t-h == p.cap { // pool 满
			return ErrPoolFull
		}

		index := t % p.cap

		if atomic.CompareAndSwapUint32(&p.tail, t, t+1) {
			p.conns[index] = c
			return nil
		}
	}
}

/* -------------------------------------------------------
 * 关闭池
 * -----------------------------------------------------*/
func (p *Pool) Close() {
	for i := 0; i < int(p.cap); i++ {
		if p.conns[i] != nil {
			p.conns[i].Close()
		}
	}
}
