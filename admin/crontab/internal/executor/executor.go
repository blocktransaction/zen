package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/blocktransaction/zen/admin/crontab/internal/store"
)

type Executor struct {
	store      store.Store
	httpClient *http.Client
}

func NewExecutor(s store.Store) *Executor {
	return &Executor{
		store:      s,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Execute tries to run job (supports http POST or cmd://shell)
func (e *Executor) Execute(ctx context.Context, j *store.Job) error {
	start := time.Now()
	rec := &store.ExecRecord{
		JobID:     j.ID,
		StartTime: start,
		Attempt:   0,
	}
	// ensure record persisted at end
	defer func() {
		
		_ = e.store.InsertExec(context.Background(), rec)
	}()

	var lastErr error
	for attempt := 0; attempt <= j.RetryCount; attempt++ {
		rec.Attempt = attempt + 1
		if attempt > 0 {
			backoff := time.Duration(attempt) * 500 * time.Millisecond
			time.Sleep(backoff)
		}
		if isHTTP(j.Target) {
			err := e.callHTTP(ctx, j.Target, j.TimeoutSec)
			if err == nil {
				rec.Success = true
				rec.EndTime = ptrTime(time.Now())
				rec.Output = "http ok"
				return nil
			}
			lastErr = err
			rec.Output = err.Error()
		} else if isCmd(j.Target) {
			cmd := j.Target[len("cmd://"):]
			err := e.execCmd(ctx, cmd, j.TimeoutSec)
			if err == nil {
				rec.Success = true
				rec.EndTime = ptrTime(time.Now())
				rec.Output = "cmd ok"
				return nil
			}
			lastErr = err
			rec.Output = err.Error()
		} else {
			lastErr = errors.New("unknown target scheme")
			rec.Output = lastErr.Error()
			break
		}
	}
	// final
	rec.Success = false
	rec.EndTime = ptrTime(time.Now())
	return lastErr
}

// 调用 http
func (e *Executor) callHTTP(ctx context.Context, url string, timeout int) error {
	tctx, cancel := context.WithTimeout(ctx, time.Duration(maxInt(1, timeout))*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(tctx, http.MethodPost, url, bytes.NewBuffer(nil))
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("http status %d", resp.StatusCode)
}

// 执行cmd
func (e *Executor) execCmd(ctx context.Context, cmdStr string, timeout int) error {
	tctx, cancel := context.WithTimeout(ctx, time.Duration(maxInt(1, timeout))*time.Second)
	defer cancel()
	cmd := exec.CommandContext(tctx, "sh", "-c", cmdStr)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cmd fail: %v output=%s", err, out.String())
	}
	return nil
}

func isHTTP(u string) bool { return len(u) > 4 && (u[:4] == "http") }
func isCmd(u string) bool  { return len(u) > 6 && u[:6] == "cmd://" }

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func ptrTime(t time.Time) *time.Time { return &t }
