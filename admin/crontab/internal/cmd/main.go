package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blocktransaction/zen/admin/crontab/internal/api"
	"github.com/blocktransaction/zen/admin/crontab/internal/executor"
	"github.com/blocktransaction/zen/admin/crontab/internal/lock"
	"github.com/blocktransaction/zen/admin/crontab/internal/scheduler"
	"github.com/blocktransaction/zen/admin/crontab/internal/store"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	dsn       = flag.String("dsn", "root:123456@tcp(127.0.0.1:3307)/scheduler?charset=utf8mb4&parseTime=True&loc=Local", "mysql dsn")
	redisAddr = flag.String("redis", "127.0.0.1:6379", "redis addr")
	addr      = flag.String("addr", ":8080", "http listen")
)

func main() {
	flag.Parse()

	// gorm
	db, err := gorm.Open(mysql.Open(*dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("gorm open: %v", err)
	}

	// run automigrate
	if err := store.AutoMigrate(db); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}

	// store
	st := store.NewGormStore(db)

	// redis client
	rdb := redis.NewClient(&redis.Options{Addr: *redisAddr})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping: %v", err)
	}
	lockMgr := lock.NewRedisLock(rdb)

	// executor
	ex := executor.NewExecutor(st)

	// scheduler
	sched := scheduler.New(scheduler.Config{
		Store:    st,
		Lock:     lockMgr,
		Executor: ex,
	})
	if err := sched.Start(ctx); err != nil {
		log.Fatalf("scheduler start: %v", err)
	}

	// api
	a := api.NewAPI(st, sched, ex)
	a.MountRoutes()

	srv := &http.Server{Addr: *addr}

	go func() {
		log.Printf("http listening %s", *addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http serve: %v", err)
		}
	}()

	// graceful
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down")
	sched.Stop(ctx)
	_ = srv.Shutdown(ctx)
	// close redis
	_ = rdb.Close()
	// allow some time
	time.Sleep(300 * time.Millisecond)
	fmt.Println("bye")
}
