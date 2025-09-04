package user

import (
	"github.com/blocktransaction/zen/app/handler/api/common"
	"github.com/blocktransaction/zen/app/service/user"
	"github.com/gin-gonic/gin"
)

type UserApi struct {
	common.Api
}

// 获取用户信息
func (api UserApi) GetUserInfo(c *gin.Context) {

	api.WithContext(c).
		WithLogger()

	userService := user.NewUserService(api.CommonContext, nil) // userdao.NewUserImplDao(api.CommonContext, mysql.GetOrm(api.Env)))

	api.Success("success", gin.H{
		"ok": userService.GetUserInfo(),
	})
}
