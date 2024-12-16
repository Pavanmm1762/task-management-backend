package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/go/task_management/backend/controllers"
)

func InitNotificationRoutes(r *gin.RouterGroup) {
	r.GET("/notifications", controllers.GetNotifications)
	r.POST("/notifications/:id/mark-read", controllers.MarkAsRead)
}

func MemberProjectRoutes(r *gin.RouterGroup) {
	r.GET("/projects-list", controllers.GetMemberProjects)
}
