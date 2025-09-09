package user

import (
	"context"
	"time"

	"github.com/blocktransaction/zen/app/dao/user"
	"github.com/blocktransaction/zen/app/model"
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

func (s *userServiceImpl) CreateUser() (bool, error) {
	// redis.
	// redisCli := redis.NewRedisCli(s.base.Ctx, s.base.Env())
	// redisCli.Get("test00111")
	// return s.base.TraceId()

	info := model.User{
		Name:      "zorro",
		CreatedAt: time.Now().Unix(),
	}

	return s.userDao.Create(&info)
}
