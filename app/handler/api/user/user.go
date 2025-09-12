package user

import (
	userdao "github.com/blocktransaction/zen/app/dao/user"
	"github.com/blocktransaction/zen/app/handler/api/common"
	"github.com/blocktransaction/zen/app/handler/api/httpreq"
	"github.com/blocktransaction/zen/app/service/user"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.uber.org/zap"
)

type UserApi struct {
	common.Api
}

// 获取用户信息
func (api UserApi) GetUserInfo(c *gin.Context) {
	var req httpreq.FindReq

	if err := api.WithLogger().
		WithContext(c).
		Bind(&req, binding.Query).Errors; err != nil {
		api.Error("1000000")
		return
	}

	userService := user.NewUserService(api.GetContext(), userdao.NewUserImplDao(api.GetContext()))
	list, count, err := userService.ListUser(&req)
	if err != nil {
		api.Logger().Error("ListUser error", zap.Error(err))
		api.Error("2000002")
		return
	}

	api.SuccessWithPagination("success", count, list, req.PageSize, req.PageIndex)
}
