package router

import (
	"github.com/blocktransaction/zen/app/middleware"
	"github.com/blocktransaction/zen/app/middleware/logger"
	"github.com/blocktransaction/zen/config"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"     // swagger embed files
	ginSwagger "github.com/swaggo/gin-swagger" // gin-swagger middleware
	"go.uber.org/zap"
)

var (
	routerGroupsV1 = make([]func(*gin.RouterGroup), 0)
	routerGroupsV2 = make([]func(*gin.RouterGroup), 0)
)

// 路由配置
func InitRouter(zapLogger *zap.Logger) *gin.Engine {
	engine := gin.New()
	//检查是否是prod
	if config.ApplicationConfig.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	//跨域处理
	engine.Use(middleware.Cors(), middleware.Secure())

	//日志处理,捕捉crash日志
	engine.Use(logger.GinzapWithBody(zapLogger,
		logger.WithEnableReqBody(true),
		logger.WithEnableRespBody(true),
		logger.WithGuessJSON(true),
		logger.WithMaxBodySize(2048),
		logger.WithOnlyJSONBody(true),
		logger.WithSensitiveKeys([]string{"password", "token"}),
	), logger.RecoveryWithZap(zapLogger, true))
	//路由过滤处理
	engine.Use(middleware.UserAuthMiddleware(middleware.AllowPathPrefixSkipper(config.ApiConfig.AllowPathPrefixSkipper)))
	//swagger处理
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	registerAllRoutes(engine)
	return engine
}

// 注册所有路由
func registerAllRoutes(r *gin.Engine) {
	// 可根据业务需求来设置接口版本
	v1 := r.Group("/api/v1")
	for _, f := range routerGroupsV1 {
		f(v1)
	}

	v2 := r.Group("/api/v2")
	for _, f := range routerGroupsV2 {
		f(v2)
	}
}
