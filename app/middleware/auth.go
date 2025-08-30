package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// 用户授权中间件
func UserAuthMiddleware(skippers ...SkipperFunc) gin.HandlerFunc {
	return func(c *gin.Context) {

		if SkipHandler(c, skippers...) || strings.Contains(c.Request.URL.Path, "swagger/") {
			c.Next()
			return
		}

		c.Next()
	}
}
