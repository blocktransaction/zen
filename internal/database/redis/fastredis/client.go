package fastredis

import (
	"bufio"
	"errors"
	"net"
	"strconv"
	"sync"
	"time"
)

var (
	ErrProtocol = errors.New("redis protocol error")
)

// ---- Buffer Pool ----
var bufPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 0, 256)
		return &b
	},
}

// ---- Map Pool ----
var mapPool = sync.Pool{
	New: func() interface{} {
		return make(map[string]string, 16)
	},
}

// ---- Fast Client ----

type Client struct {
	conn   net.Conn
	reader *bufio.Reader
}

func Dial(addr string, timeout time.Duration) (*Client, error) {
	c, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}
	return &Client{
		conn:   c,
		reader: bufio.NewReaderSize(c, 32*1024),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

/* -------------------------------------------------------
 * Internal: Write command
 * -----------------------------------------------------*/
func (c *Client) writeCommand(cmd string, args ...string) error {
	// 使用 buffer 池提升性能
	bptr := bufPool.Get().(*[]byte)
	b := (*bptr)[:0]

	// 构建 RESP 数组头
	b = append(b, '*')
	b = strconv.AppendInt(b, int64(1+len(args)), 10)
	b = append(b, '\r', '\n')

	// 写命令
	b = appendBulk(b, cmd)
	for _, arg := range args {
		b = appendBulk(b, arg)
	}

	// 发送
	_, err := c.conn.Write(b)
	// 释放 buffer
	*bptr = b[:0]
	bufPool.Put(bptr)
	return err
}

func appendBulk(b []byte, s string) []byte {
	b = append(b, '$')
	b = strconv.AppendInt(b, int64(len(s)), 10)
	b = append(b, '\r', '\n')
	b = append(b, s...)
	b = append(b, '\r', '\n')
	return b
}

/* -------------------------------------------------------
 * Simple fast GET
 * -----------------------------------------------------*/
func (c *Client) Get(key string) (string, error) {
	if err := c.writeCommand("GET", key); err != nil {
		return "", err
	}
	return c.readBulk()
}

func (c *Client) readBulk() (string, error) {
	t, err := c.reader.ReadByte()
	if err != nil {
		return "", err
	}

	switch t {
	case '$':
		// bulk
		lenBytes, err := c.reader.ReadBytes('\n')
		if err != nil {
			return "", err
		}
		n, _ := strconv.Atoi(string(lenBytes[:len(lenBytes)-2]))
		if n < 0 {
			return "", nil // nil bulk
		}
		buf := make([]byte, n+2)
		_, err = c.reader.Read(buf)
		if err != nil {
			return "", err
		}
		return string(buf[:n]), nil

	case '-': // error
		line, _ := c.reader.ReadBytes('\n')
		return "", errors.New(string(line[:len(line)-2]))

	default:
		return "", ErrProtocol
	}
}

/* -------------------------------------------------------
 * Fast HGETALL (map 池复用)
 * -----------------------------------------------------*/
func (c *Client) HGetAll(key string) (map[string]string, error) {
	if err := c.writeCommand("HGETALL", key); err != nil {
		return nil, err
	}

	t, err := c.reader.ReadByte()
	if err != nil {
		return nil, err
	}
	if t != '*' {
		return nil, ErrProtocol
	}

	// array length
	lenBytes, err := c.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	n, _ := strconv.Atoi(string(lenBytes[:len(lenBytes)-2]))
	if n%2 != 0 {
		return nil, ErrProtocol
	}

	// map 池获取对象（无锁）
	m := mapPool.Get().(map[string]string)
	// 清空旧内容
	for k := range m {
		delete(m, k)
	}

	// 解析 key-value
	for i := 0; i < n/2; i++ {
		f, err := c.readBulk()
		if err != nil {
			mapPool.Put(m)
			return nil, err
		}
		v, err := c.readBulk()
		if err != nil {
			mapPool.Put(m)
			return nil, err
		}
		m[f] = v
	}

	return m, nil
}

// 释放 map 返回池
func ReleaseMap(m map[string]string) {
	for k := range m {
		delete(m, k)
	}
	mapPool.Put(m)
}
