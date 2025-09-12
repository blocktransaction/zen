package user

import (
	"context"

	"github.com/blocktransaction/zen/app/dao/dao"
	"github.com/blocktransaction/zen/app/handler/api/httpreq"
	"github.com/blocktransaction/zen/app/model"
	"github.com/blocktransaction/zen/common/constant"
	"github.com/blocktransaction/zen/internal/database/mysql"
)

type userImplDao struct {
	dao *dao.DAO[model.User]
}

func NewUserImplDao(ctx context.Context) UserDao {
	env := ctx.Value(constant.EnvKey).(string)
	return &userImplDao{
		dao: dao.NewDAO[model.User](ctx, mysql.GetOrm(env)),
	}
}

// 创建用户
func (d *userImplDao) Create(user *model.User) (bool, error) {
	if err := d.dao.Create(user); err != nil {
		return false, err
	}
	return true, nil
}

// 查找
func (d *userImplDao) Find(req *httpreq.FindReq) ([]model.User, int64, error) {
	var (
		list  []model.User
		count int64
	)

	if err := d.dao.Eq("name", "zorro").
		Count(&count).
		OrderBy("created_at DESC").
		Paginate(req.PageIndex, req.PageSize).
		Find(&list); err != nil {
		return nil, 0, err
	}
	return list, count, nil
}
