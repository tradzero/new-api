package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"

	"github.com/gin-gonic/gin"
)

func SetAudioTaskRouter(router *gin.Engine) {
	audioTaskRouter := router.Group("/v1")
	audioTaskRouter.Use(middleware.TokenAuth(), middleware.Distribute())
	{
		audioTaskRouter.POST("/audio/custom", controller.RelayTask)
		audioTaskRouter.GET("/audio/tasks/:task_id", controller.RelayTaskFetch)
	}
}
