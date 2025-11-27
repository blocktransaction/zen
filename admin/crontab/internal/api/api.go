package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/blocktransaction/zen/admin/crontab/internal/executor"
	"github.com/blocktransaction/zen/admin/crontab/internal/scheduler"
	"github.com/blocktransaction/zen/admin/crontab/internal/store"
)

type API struct {
	store store.Store
	sched *scheduler.Scheduler
	ex    *executor.Executor
}

func NewAPI(s store.Store, sch *scheduler.Scheduler, ex *executor.Executor) *API {
	return &API{store: s, sched: sch, ex: ex}
}

func (a *API) MountRoutes() {
	http.HandleFunc("/api/jobs/add", a.handleAdd)
	http.HandleFunc("/api/jobs/list", a.handleList)
	http.HandleFunc("/api/jobs/trigger", a.handleTrigger)
	http.HandleFunc("/api/jobs/stop", a.handleTrigger)
}

// 新增
func (a *API) handleAdd(w http.ResponseWriter, r *http.Request) {
	var j store.Job
	if err := json.NewDecoder(r.Body).Decode(&j); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if j.Name == "" || j.CronExpr == "" || j.Target == "" {
		http.Error(w, "name/cron/target required", 400)
		return
	}
	if err := a.store.CreateJob(context.Background(), &j); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	// 动态热加载
	if err := a.sched.AddJob(&j); err != nil {
		log.Printf("hot load job err: %v", err)
	}
	fmt.Fprint(w, "ok")
}

// 列表
func (a *API) handleList(w http.ResponseWriter, r *http.Request) {
	jobs, _ := a.store.ListJobs(context.Background())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jobs)
}

// 触发
func (a *API) handleTrigger(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "id required", 400)
		return
	}
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", 400)
		return
	}
	j, err := a.store.FindJob(context.Background(), uint(id64))
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	go func() {
		if err := a.ex.Execute(context.Background(), j); err != nil {
			log.Printf("trigger exec err: %v", err)
		}
	}()
	fmt.Fprint(w, "triggered")
}
