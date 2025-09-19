package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/blocktransaction/zen/admin/crontab/internal/executor"
	"github.com/blocktransaction/zen/admin/crontab/internal/lock"
	"github.com/blocktransaction/zen/admin/crontab/internal/store"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

type Config struct {
	Store    store.Store
	Lock     *lock.RedisLock
	Executor *executor.Executor
}

type Scheduler struct {
	cfg  Config
	cron *cron.Cron
	jobs map[uint]cron.EntryID
	mu   sync.Mutex
	wg   sync.WaitGroup
}

func New(cfg Config) *Scheduler {
	return &Scheduler{
		cfg:  cfg,
		cron: cron.New(cron.WithSeconds()),
		jobs: make(map[uint]cron.EntryID),
	}
}

// 开始
func (s *Scheduler) Start(ctx context.Context) error {
	jobs, err := s.cfg.Store.ListJobs(ctx)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if err := s.addJob(j); err != nil {
			log.Printf("add job err: %v", err)
		}
	}
	s.cron.Start()
	return nil
}

// 停止
func (s *Scheduler) Stop(ctx context.Context) {
	s.cron.Stop()
	s.wg.Wait()
}

// AddJob 动态注册新任务到 cron
func (s *Scheduler) AddJob(j *store.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.addJob(*j)
}

// 添加job
func (s *Scheduler) addJob(j store.Job) error {
	job := j // capture
	entryID, err := s.cron.AddFunc(job.CronExpr, func() {
		go s.runWithLock(context.Background(), &job)
	})
	if err != nil {
		return err
	}
	s.jobs[job.ID] = entryID
	log.Printf("scheduled job %d %s", job.ID, job.CronExpr)
	return nil
}

// runWithLock 支持锁续租
func (s *Scheduler) runWithLock(ctx context.Context, j *store.Job) {
	key := fmt.Sprintf("task-lock:%d", j.ID)
	val := uuid.New().String()
	ttl := 30 * time.Second

	ok, err := s.cfg.Lock.Acquire(ctx, key, val, ttl)
	if err != nil {
		log.Printf("lock acquire err: %v", err)
		return
	}
	if !ok {
		log.Printf("job %d locked by other", j.ID)
		return
	}

	ctxCancel, cancel := context.WithCancel(ctx)
	defer cancel()
	defer func() {
		// 尝试释放锁
		if err := s.cfg.Lock.SafeRelease(context.Background(), key, val); err != nil {
			log.Printf("unlock err: %v", err)
		}
	}()

	// 启动锁续租协程
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(ttl / 2)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := s.cfg.Lock.Extend(ctxCancel, key, val, ttl); err != nil {
					log.Printf("lock extend err: %v", err)
					cancel()
					return
				}
			case <-done:
				return
			case <-ctxCancel.Done():
				return
			}
		}
	}()

	s.wg.Add(1)
	defer s.wg.Done()
	if err := s.cfg.Executor.Execute(ctxCancel, j); err != nil {
		log.Printf("job %d exec err: %v", j.ID, err)
	}

	close(done)
}
