package user

import (
	"github.com/blocktransaction/zen/app/handler/api/httpreq"
	"github.com/blocktransaction/zen/app/model"
)

type UserDao interface {
	//创建
	Create(*model.User) (bool, error)
	//查找
	Find(*httpreq.FindReq) ([]model.User, int64, error)
}
