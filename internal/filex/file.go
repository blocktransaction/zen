package filex

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerResult collects results safely
type WorkerResult struct {
	mu    sync.Mutex
	files []string
}

func (wr *WorkerResult) Append(p string) {
	wr.mu.Lock()
	wr.files = append(wr.files, p)
	wr.mu.Unlock()
}

// 并发扫描目录，使用无锁 RingQueue
// pattern：通配符，如 "*.go"
// ignoreCase：匹配是否忽略大小写
// recursive：是否扫描子目录
// workers：工作协程数量
// ringSize：队列容量(容量选择：ringSize 建议 2^k（如 1<<14 = 16384）。若太小会频繁满而触发 busy-wait；太大占内存。)
func ListFilesWithRingMPMC(
	root, pattern string,
	ignoreCase, recursive bool,
	workers int,
	ringSize int,
) ([]string, error) {

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(absRoot)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", absRoot)
	}
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if ringSize <= 0 {
		ringSize = 1 << 14 // default 16384
	}

	// prepare pattern lower-case if ignoreCase
	if ignoreCase {
		pattern = strings.ToLower(pattern)
	}

	matchName := func(name string) (bool, error) {
		if ignoreCase {
			name = strings.ToLower(name)
		}
		return filepath.Match(pattern, name)
	}

	fileQ := NewRingQueue(ringSize)
	result := &WorkerResult{}
	var producerErr atomic.Value // store error
	var producersDone int32
	var consumersWg sync.WaitGroup
	consumersWg.Add(workers)

	// consumers (workers) - pop file paths, test and append results
	for i := 0; i < workers; i++ {
		go func() {
			defer consumersWg.Done()
			backoff := 0
			for {
				// try poll
				p, ok := fileQ.Poll()
				if !ok {
					// check if producers finished and queue empty
					if atomic.LoadInt32(&producersDone) == 1 {
						// final drain: if empty -> exit
						if _, still := fileQ.Poll(); !still {
							return
						}
						// else continue to consume
						continue
					}
					// backoff: spin a few times then sleep briefly
					backoff++
					if backoff < 4 {
						runtime.Gosched()
					} else {
						time.Sleep(time.Microsecond * 50)
					}
					continue
				}
				backoff = 0
				// got file path p
				name := filepath.Base(p)
				okm, merr := matchName(name)
				if merr != nil {
					// store error and stop
					producerErr.Store(merr)
					return
				}
				if okm {
					result.Append(p)
				}
			}
		}()
	}

	// producer: WalkDir and push file paths into ring buffer
	walkErr := filepath.WalkDir(absRoot, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if !recursive && p != absRoot {
				return filepath.SkipDir
			}
			return nil
		}
		// push into ring queue; non-blocking with spin+sleep fallback
		backoff := 0
		for {
			if fileQ.Offer(p) {
				break
			}
			// queue full; spin and then sleep
			backoff++
			if backoff < 4 {
				runtime.Gosched()
			} else {
				time.Sleep(time.Microsecond * 100)
			}
			// also check if any producer error stored
			if pe := producerErr.Load(); pe != nil {
				return pe.(error)
			}
		}
		return nil
	})

	// mark producers done
	atomic.StoreInt32(&producersDone, 1)

	// if walk encountered error, save
	if walkErr != nil {
		producerErr.Store(walkErr)
	}

	// wait consumers to finish processing remaining items
	consumersWg.Wait()

	// check producerErr
	if pe := producerErr.Load(); pe != nil {
		return nil, pe.(error)
	}

	return result.files, nil
}
