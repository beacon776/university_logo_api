package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/tencentyun/scf-go-lib/cloudfunction"
	"github.com/tencentyun/scf-go-lib/events" // 这里才有 APIGatewayRequest
	"go.uber.org/zap"
	"logo_api/dao/mysql"
	"logo_api/dao/redis"
	"logo_api/logger"
	"logo_api/routes"
	"logo_api/service"
	"logo_api/settings"
	"logo_api/util"
	"os"
)

var (
	r      *gin.Engine
	client *util.CosClient
	svc    *service.ResourceService
)

func init() {

	// 1.加载配置
	if err := settings.Init(); err != nil {
		panic(fmt.Sprintf("settings.Init() failed: %s", err))
	}

	// 2.初始化日志
	if err := logger.Init(settings.Config.LogConfig); err != nil {
		panic(fmt.Sprintf("logger.Init() failed: %s", err))
	}
	zap.L().Info("Logger init success")

	// 3.初始化 MySQL
	if err := mysql.Init(settings.Config.MysqlConfig); err != nil {
		panic(fmt.Sprintf("mysql.Init() failed: %s", err))
	}

	// 4.初始化 Redis
	if err := redis.Init(settings.Config.RedisConfig); err != nil {
		panic(fmt.Sprintf("redis.Init() failed: %s", err))
	}

	// 5.初始化 COS
	var err error
	client, err = util.NewClient(settings.Config.CosConfig)
	if err != nil {
		panic(fmt.Sprintf("util.NewClient failed: %s", err))
	}
	// 6.初始化 ResourceService（全局）
	svc = service.NewResourceService(client)
	// 7.注册路由
	r = routes.Setup(svc)

	// 8. 统一监听端口，启动 Gin 服务
	port := os.Getenv("PORT") // SCF 会自动注入 PORT 环境变量（一般是 9000）
	if port == "" {
		port = "9000"
	}
	go func() {
		if err := r.Run(":" + port); err != nil {
			zap.L().Fatal("server start failed", zap.Error(err))
		}
	}()
}

// Handler 是云函数的入口, 它就是个空壳，Web 函数只走 Gin 路由，Handler 只是占位 + Timer 清理
func Handler(ctx context.Context, evt json.RawMessage) (interface{}, error) {
	return events.APIGatewayResponse{}, nil
}

func main() {
	cloudfunction.Start(Handler)
	/*
		if err := localRun(); err != nil {
			zap.L().Fatal("localRun failed", zap.Error(err))
		}

	*/

	/*
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			ticker := time.NewTicker(20 * time.Minute)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					err := svc.CleanExpiredCOSObjects(ctx)
					if err != nil {
						zap.L().Error("CleanExpiredCOSObjects error", zap.Error(err))
					}
				case <-ctx.Done():
					return
				}
			}
		}()

		// 7.优雅关机
		server := &http.Server{
			Addr:    fmt.Sprintf("%s:%d", settings.Config.AppSettings.Host, settings.Config.AppSettings.Port),
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
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			zap.L().Fatal("Server Shutdown", zap.Error(err))
		}
		zap.L().Info("Server exiting")

	*/
}
func localRun() error {
	// 本地开发调试
	// 1. 加载配置
	if err := settings.Init(); err != nil {
		panic(fmt.Sprintf("settings.Init() failed: %s", err))
	}

	// 2. 初始化日志
	if err := logger.Init(settings.Config.LogConfig); err != nil {
		panic(fmt.Sprintf("logger.Init() failed: %s", err))
	}

	// 3. 初始化 MySQL
	if err := mysql.Init(settings.Config.MysqlConfig); err != nil {
		panic(fmt.Sprintf("mysql.Init() failed: %s", err))
	}

	// 4. 初始化 Redis
	if err := redis.Init(settings.Config.RedisConfig); err != nil {
		panic(fmt.Sprintf("redis.Init() failed: %s", err))
	}

	// 5. 初始化 COS
	client, err := util.NewClient(settings.Config.CosConfig)
	if err != nil {
		panic(fmt.Sprintf("util.NewClient failed: %s", err))
	}
	svc := service.NewResourceService(client)

	// 6. 注册路由
	r := routes.Setup(svc)

	// 7. 本地监听端口
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // 本地就用 8080
	}
	r.Run(":" + port)
	return nil
}
