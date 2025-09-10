package user

import (
	"github.com/blocktransaction/zen/app/handler/api/httpreq"
	"github.com/blocktransaction/zen/app/model"
)

// interface
type UserService interface {
	//创建
	CreateUser() (bool, error)
	//列表
	ListUser(*httpreq.FindReq) ([]model.User, int64, error)
}
