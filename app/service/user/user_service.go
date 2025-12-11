package user

import (
	"github.com/blocktransaction/zen/app/handler/api/httpreq"
	"github.com/blocktransaction/zen/app/model/entity"
)

// interface
type UserService interface {
	//创建
	CreateUser() (bool, error)
	//列表
	ListUser(*httpreq.FindReq) ([]entity.User, int64, error)
}
