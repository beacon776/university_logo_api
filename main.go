package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
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
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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
	runMode := viper.GetString("RUN_MODE") // 新引入的环境变量
	// 临时调试代码
	fmt.Printf("[DEBUG] Detected RUN_MODE: '%s'\n", runMode)
	// 2. 本地模式
	if runMode == "local" {
		zap.L().Info("starting server")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		// 定时任务 Goroutine
		go func() {
			// 确保服务器已经启动并监听，使用 settings 中的配置或默认值
			host := viper.GetString("app.host") // 假设 host 在 settings 中配置
			port := viper.GetString("app.port") // 假设 port 在 settings 中配置
			if host == "" {
				host = "localhost"
			}
			if port == "" {
				port = "9000"
			} // 确保使用实际监听的端口

			clearCacheURL := fmt.Sprintf("http://%s:%s/clearCache", host, port)

			ticker := time.NewTicker(1 * time.Minute) // 本体调试，每1min删除一次缓存
			defer ticker.Stop()

			// 立即执行一次清理（可选）
			zap.L().Info("Starting initial cache cleanup request.")
			triggerClearCache(clearCacheURL)

			for {
				select {
				case <-ticker.C:
					zap.L().Info("Sending request to clearCache route.")
					triggerClearCache(clearCacheURL)
				case <-ctx.Done():
					return
				}
			}
		}()

		// 优雅关机
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
	}
	// 3. 云函数模式
	cloudfunction.Start(Handler)

}

// 辅助函数：发送 HTTP POST 请求到 /clearCache
func triggerClearCache(url string) {
	// 创建一个新的 POST 请求
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		zap.L().Error("Failed to create clearCache request", zap.Error(err))
		return
	}

	// 设置超时客户端
	client := http.Client{
		Timeout: 1 * time.Minute, // 同样给 1 分钟的超时时间
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		zap.L().Error("Failed to send clearCache request", zap.String("url", url), zap.Error(err))
		return
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		zap.L().Error("clearCache route returned non-200 status",
			zap.String("url", url),
			zap.Int("status", resp.StatusCode))
	} else {
		zap.L().Info("Successfully triggered clearCache route", zap.String("url", url))
	}
}
