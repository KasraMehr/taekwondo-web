package athletes

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.RouterGroup, handler *Handler) {
	routes := router.Group("/athletes")

	routes.POST("", handler.Create)
	routes.GET("", handler.List)
	routes.GET("/:id", handler.GetByID)
	routes.PUT("/:id", handler.Update)
	routes.DELETE("/:id", handler.Delete)
}
