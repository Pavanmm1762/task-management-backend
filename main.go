// main.go
package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go/task_management/backend/controllers"
	"github.com/go/task_management/backend/middleware"
	"github.com/go/task_management/backend/utils"
)

func main() {
	utils.InitCassandra()
	//defer utils.Session.Close()

	server := gin.Default()
	// Enable CORS middleware
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000"} // Add your frontend origin here
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE"}
	config.AllowHeaders = []string{"Authorization", "Content-Type"}
	server.Use(cors.New(config))

	authGroup := server.Group("/auth")
	{
		authGroup.POST("/register", controllers.Register)
		authGroup.POST("/login", controllers.Login)
	}

	apiGroup := server.Group("/api")
	apiGroup.Use(middleware.AuthMiddleware()) //Middleware for authentication
	{
		controllers.InitProjectRoutes(apiGroup)
		controllers.InitTaskRoutes(apiGroup)
		controllers.InitUserRoutes(apiGroup)
		controllers.InitReportRoutes(apiGroup)
		controllers.InitDashboardRoutes(apiGroup)
	}
	//controllers.InitCommentRoutes(apiGroup)
	//controllers.InitNotificationRoutes(apiGroup)

	//server.GET("/ws", controllers.HandleWebSocket)
	server.GET("/ws", controllers.HandleWebSocket)

	// // Serve static files
	// server.Static("/static", "./static")
	// server.NoRoute(func(c *gin.Context) {
	// 	c.File("./static/index.html")
	// })

	server.Run(":8080")
}
