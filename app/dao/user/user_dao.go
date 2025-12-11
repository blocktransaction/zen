package user

import (
	"github.com/blocktransaction/zen/app/handler/api/httpreq"
	"github.com/blocktransaction/zen/app/model/entity"
)

type UserDao interface {
	//创建
	Create(*entity.User) (bool, error)
	//查找
	Find(*httpreq.FindReq) ([]entity.User, int64, error)
}
