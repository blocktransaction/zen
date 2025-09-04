package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blocktransaction/zen/app/router"
	"github.com/blocktransaction/zen/config"
	"github.com/blocktransaction/zen/internal/i18n"
	"github.com/blocktransaction/zen/internal/logx"
	"github.com/spf13/cobra"
)

var (
	configPath  string
	autoMigrate bool
	StartCmd    = &cobra.Command{
		Use:          "server",
		Short:        "Start API server",
		Example:      "zen server -c config/",
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			setup()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return run()
		},
	}
)

// init
func init() {
	// 配置文件路径
	StartCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config/", "Start server with provided configuration file")
	// 数据库合并迁移
	StartCmd.PersistentFlags().BoolVarP(&autoMigrate, "autoMigrate", "a", false, "database auto migrate")
}

// 初始化相关
func setup() {
	config.Setup(
		configPath,
		i18n.Setup,
		// mysql.Setup,
		// redis.Setup,
	)
}

// 运行
func run() error {
	//开启自动迁移模式
	autoDatabaseMigrate()
	//初始化日志
	zapLog := logx.NewLogger(
		logx.WithLogFileName(config.ApplicationConfig.LogFileName),
		logx.WithLogFilePath(config.ApplicationConfig.LogFilePath),
		logx.WithSerivceName(config.ApplicationConfig.LogName),
		logx.WithLogFileMaxSize(config.ApplicationConfig.LogFileMaxSize),
		logx.WithLogLogFileMaxAge(config.ApplicationConfig.LogFileMaxAge),
	)

	//server配置
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.ServerConfig.Host, config.ServerConfig.Port),
		Handler: router.InitRouter(zapLog),
	}

	go func() {
		//启动api服务
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("ListenAndServe error: %s\n", err)
		}
	}()

	// 等待信号以关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Shutting down server...")

	// 创建一个5秒的超时上下文，等待未完成的请求完成
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("Server shutdown error: %s\n", err)
	}

	fmt.Println("Server stopped.")

	return nil
}

// 自动迁移数据库
func autoDatabaseMigrate() {
	if autoMigrate {

	}
}
