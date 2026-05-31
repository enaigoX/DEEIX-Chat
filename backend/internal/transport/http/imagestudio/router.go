package imagestudio

import "github.com/gin-gonic/gin"

// RegisterRoutes registers protected Image Studio routes.
func (m *Module) RegisterRoutes(authRequired *gin.RouterGroup) {
	group := authRequired.Group("/image-studio")
	group.POST("/tasks/stream", m.Handler.StreamTask)
	group.GET("/tasks", m.Handler.ListTasks)
	group.GET("/tasks/:task_id", m.Handler.GetTask)
	group.PATCH("/tasks/:task_id/favorite", m.Handler.SetFavorite)
	group.DELETE("/tasks/:task_id", m.Handler.DeleteTask)
}
