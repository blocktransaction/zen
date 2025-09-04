package router

import (
	"github.com/blocktransaction/zen/app/handler/api/user"
	"github.com/gin-gonic/gin"
)

func init() {
	routerGroupsV1 = append(routerGroupsV1, registerUserRouterV1)
}

// user v1路由集合
func registerUserRouterV1(v1 *gin.RouterGroup) {
	userApi := new(user.UserApi)

	userGroup := v1.Group("/user")
	{
		//获取用户信息
		userGroup.GET("/info", userApi.GetUserInfo)
	}
}
