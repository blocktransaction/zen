package service

import (
	"context"
	"sync"

	"github.com/blocktransaction/zen/common/constant"
)

// BaseService 提供通用字段和方法
type BaseService struct {
	mtx sync.Mutex
	Ctx context.Context
}

func (s *BaseService) TraceId() string {
	if val := s.Ctx.Value(constant.CtxKeyTrace); val != nil {
		return val.(string)
	}
	return ""
}

func (s *BaseService) UserId() int64 {
	if val := s.Ctx.Value(constant.CtxKeyUserId); val != nil {
		return val.(int64)
	}
	return 0
}

func (s *BaseService) Env() string {
	if val := s.Ctx.Value(constant.CtxKeyEnv); val != nil {
		return val.(string)
	}
	return ""
}

func (s *BaseService) Lang() string {
	if val := s.Ctx.Value(constant.CtxKeyLang); val != nil {
		return val.(string)
	}
	return ""
}

func (s *BaseService) Lock() {
	s.mtx.Lock()
}

func (s *BaseService) Unlock() {
	s.mtx.Unlock()
}
