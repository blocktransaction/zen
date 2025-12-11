package fastredis

import (
	"fmt"
	"testing"
	"time"
)

func TestFastRedis(t *testing.T) {
	pool, err := NewPool("127.0.0.1:6379", 128, time.Second)
	if err != nil {
		panic(err)
	}

	cli, err := pool.Get()
	if err != nil {
		// fallback: 也可以新建一个 fast-path conn
		panic(err)
	}

	v, _ := cli.Get("foo")
	fmt.Println("foo =", v)

	m, err := cli.HGetAll("user:infos")
	if err != nil {
		panic(err)
	}

	fmt.Println(m["2"], m["3"], m["4"])

	// 用完归还
	ReleaseMap(m)
}
