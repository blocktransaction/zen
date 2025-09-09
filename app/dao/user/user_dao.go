package user

import "github.com/blocktransaction/zen/app/model"

type UserDao interface {
	//创建
	Create(*model.User) (bool, error)
}
