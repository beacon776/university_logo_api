package main

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"logo_api/dao/mysql"
	"logo_api/logger"
	"logo_api/routes"
	"logo_api/service"
	"logo_api/settings"
	"logo_api/util"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	// 1.加载配置
	if err := settings.Init(); err != nil {
		fmt.Printf("settings.Init() failed, err: %s", err.Error())
		os.Exit(1) // 立刻退出程序，避免后续空指针
	}
	// 2.初始化日志
	if err := logger.Init(settings.Config.LogConfig); err != nil {
		fmt.Printf("logger.Init() failed, err: %s", err.Error())
		os.Exit(1)
	}
	zap.L().Debug("init logger success...")
	// 3.初始化mysql连接
	if err := mysql.Init(settings.Config.MysqlConfig); err != nil {
		fmt.Printf("mysql.Init() failed, err: %s", err.Error())
	}
	zap.L().Debug("init mysql success...")
	// 4.定时清除缓存
	client, err := util.NewClient(settings.Config.CosConfig)
	if err != nil {
		zap.L().Fatal("util.NewClient", zap.Error(err))
	}
	resourceService := service.NewResourceService(client)

	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				resourceService.ClearExpiredCache(time.Hour)
			}
		}
	}()
	// 5.注册路由
	r := routes.Setup()
	zap.L().Info("Starting HTTP server",
		zap.String("address", fmt.Sprintf("%s:%s", settings.Config.AppSettings.Host, settings.Config.AppSettings.Port)))
	// 6.优雅关机
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", settings.Config.AppSettings.Host, settings.Config.AppSettings.Port),
		Handler: r,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Fatal("listen failed", zap.Error(err))
		}
	}()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	zap.L().Info("Shutdown Server ...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		zap.L().Fatal("Server Shutdown", zap.Error(err))
	}
	zap.L().Info("Server exiting")

}
