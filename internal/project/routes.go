package project

import "github.com/gin-gonic/gin"

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	group.POST("", handler.Create)
	group.GET("", handler.List)
}
