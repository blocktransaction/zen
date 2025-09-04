package user

import (
	"context"

	"gorm.io/gorm"
)

type userImplDao struct {
	db *gorm.DB
}

func NewUserImplDao(ctx context.Context, db *gorm.DB) UserDao {
	return &userImplDao{db: db.WithContext(ctx)}
}

func (dao *userImplDao) Create() (bool, error) {
	return true, nil
}
