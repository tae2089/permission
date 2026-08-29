package apikey

import "github.com/gin-gonic/gin"

func RegisterRoutes(group *gin.RouterGroup, handler *Handler) {
	keys := group.Group("/projects/:project_id/api-keys")
	keys.POST("", handler.Issue)
	keys.GET("", handler.List)
	keys.POST("/:key_id/rotate", handler.Rotate)
	keys.POST("/:key_id/revoke", handler.Revoke)
}
