package routes

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"logo_api/logger"
	"logo_api/settings"
	"logo_api/util"
)

func getFromShortName(c *gin.Context) {
	shortName := c.Param("shortName")
	client, err := util.NewClient(settings.Config.CosConfig)
	if err != nil {
		zap.L().Error("util.NewClient() err:", zap.Error(err))
	}
	if err = client.UploadObject(shortName); err != nil {
		zap.L().Error("client.UploadObject() err:", zap.Error(err))
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "success",
		"data": nil,
	})

}

func Setup() *gin.Engine {
	router := gin.New()
	router.Use(logger.GinLogger(), logger.GinRecovery(true))
	router.GET("/:shortName", getFromShortName)

	return router
}
