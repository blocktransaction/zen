package user

import (
	"context"

	"github.com/blocktransaction/zen/app/dao/user"
	"github.com/blocktransaction/zen/app/service"
)

// impl
type userServiceImpl struct {
	base    *service.BaseService
	userDao user.UserDao
}

// new
func NewUserService(ctx context.Context, dao user.UserDao) UserService {
	return &userServiceImpl{
		base: &service.BaseService{
			Ctx: ctx,
		},
		userDao: dao,
	}
}

func (s *userServiceImpl) GetUserInfo() string {
	return s.base.TraceId()
}
