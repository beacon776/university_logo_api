package routes

import (
	"context"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"logo_api/auth"
	"logo_api/logger"
	"logo_api/routes/handler"
	"logo_api/service"
	"net/http"
)

// Setup 注册接口
func Setup(svc *service.ResourceService) *gin.Engine {
	router := gin.New()
	router.Use(logger.GinLogger(), logger.GinRecovery(true))
	r1 := router.Group("/")
	{
		r1.POST("/user/register", handler.RegisterFunc())
		r1.POST("/user/login", handler.UserLogin())

		r1.POST("/clearCache", clearCache(svc))
	}
	user := router.Group("/user")
	user.Use(auth.AuthRequired(svc))
	{
		user.POST("/list", handler.GetUserList())
		user.POST("/logout", handler.UserLogout())
		/*
			user.POST("/update/:id", userUpdate(svc))
			user.POST("/delete/:id", userDelete(svc))*/
	}

	university := router.Group("/university")
	university.Use(auth.AuthRequired(svc))
	{
		university.POST("/list", handler.GetUniversityList())
		// 后台管理路由：增、删、改、查、登录
		university.GET("/:name", handler.GetUniversityFromName())
		university.POST("/insert", handler.InsertUniversity())
		university.POST("/update/:name", handler.UpdateUniversities())
	}
	resource := router.Group("/resource")
	resource.Use(auth.AuthRequired(svc))
	{
		resource.GET("/getLogo/:fullName", handler.GetLogoFromNameHandler(svc))
		resource.GET("/:name", handler.GetResource())
		resource.POST("/list", handler.GetResourceList())
	}
	return router
}

// respondWithError 用户传参失误时，code为400，err为nil; 系统内部错误时，code为500，err为真实err
func respondWithError(c *gin.Context, code int, msg string, err error, location string) {
	if err != nil {
		zap.L().Error(
			location,
			zap.String("msg", msg),
			zap.Error(err))
	}
	c.JSON(code, gin.H{
		"code": code,
		"msg":  msg,
	})
}

func clearCache(svc *service.ResourceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		result, err := svc.CleanExpiredCOSObjects(ctx)
		if err != nil {
			zap.L().Error("CleanExpiredCOSObjects error", zap.Error(err))
			respondWithError(c, 500, err.Error(), nil, "CleanExpiredCOSObjects")
			return
		}
		zap.L().Info("CleanExpiredCOSObjects success")
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"msg":  "cache clean success",
			"data": result,
		})
	}
}
