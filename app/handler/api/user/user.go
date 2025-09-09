package user

import (
	userdao "github.com/blocktransaction/zen/app/dao/user"
	"github.com/blocktransaction/zen/app/handler/api/common"
	"github.com/blocktransaction/zen/app/service/user"
	"github.com/gin-gonic/gin"
)

type UserApi struct {
	common.Api
}

// 获取用户信息
func (api UserApi) GetUserInfo(c *gin.Context) {
	api.WithContext(c).WithLogger()

	userService := user.NewUserService(api.CommonContext, userdao.NewUserImplDao(api.CommonContext, api.Env))
	ok, err := userService.CreateUser()
	if err != nil {
		api.Error("2000002")
		return
	}
	api.Success("success", gin.H{
		"success": ok,
	})
}
