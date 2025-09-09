package user

import (
	"context"

	"github.com/blocktransaction/zen/app/model"
	"github.com/blocktransaction/zen/internal/database/mysql"
	"gorm.io/gorm"
)

type userImplDao struct {
	db *gorm.DB
}

func NewUserImplDao(ctx context.Context, env string) UserDao {
	return &userImplDao{
		db: mysql.GetOrm(env).WithContext(ctx),
	}
}

// 创建用户
func (dao *userImplDao) Create(user *model.User) (bool, error) {
	if err := dao.db.Create(&user).Error; err != nil {
		return false, err
	}
	return true, nil
}
