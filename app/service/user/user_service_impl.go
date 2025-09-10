package user

import (
	"context"
	"time"

	"github.com/blocktransaction/zen/app/dao/user"
	"github.com/blocktransaction/zen/app/handler/api/httpreq"
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
	info := model.User{
		Name:      "zorro",
		CreatedAt: time.Now().Unix(),
	}

	return s.userDao.Create(&info)
}

func (s *userServiceImpl) ListUser(req *httpreq.FindReq) ([]model.User, int64, error) {
	return s.userDao.Find(req)
}
