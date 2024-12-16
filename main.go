// main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go/task_management/backend/controllers"
	"github.com/go/task_management/backend/middleware"
	"github.com/go/task_management/backend/routes"
	"github.com/go/task_management/backend/utils"
	"github.com/go/task_management/backend/webSocket"
)

func main() {

	// Initialize database connection
	utils.ConnectToDB()
	defer utils.Close() // Ensure the connection pool is closed on exit
	utils.CreateTables()
	// Check if the database is reachable
	if err := utils.Ping(); err != nil {
		log.Fatalf("Database is unreachable: %v", err)
	}

	// c := cron.New()

	// // Schedule the job to run every hour
	// c.AddFunc("@hourly", func() {
	// 	err := controllers.UpdateOverdueTasks()
	// 	if err != nil {
	// 		log.Printf("Error updating overdue tasks: %v", err)
	// 	}
	// })

	// c.Start()

	// Start WebSocket server for real-time updates
	go webSocket.StartWebSocketServer()

	server := gin.Default()

	// Enable CORS middleware
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Authorization", "Content-Type"}
	config.AllowCredentials = true
	server.Use(cors.New(config))

	server.OPTIONS("/*path", func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "http://localhost:3000")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Origin, Accept")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.AbortWithStatus(204)
	})

	authGroup := server.Group("/auth")
	{
		authGroup.POST("/register", controllers.RegisterAdmin)
		authGroup.POST("/login", controllers.LoginAdmin)
	}

	adminAPIGroup := server.Group("/api")
	adminAPIGroup.Use(middleware.AuthMiddleware("admin")) //Middleware for authentication
	{
		controllers.InitProjectRoutes(adminAPIGroup)
		controllers.InitTaskRoutes(adminAPIGroup)
		controllers.InitUserRoutes(adminAPIGroup)
		controllers.InitReportRoutes(adminAPIGroup)
		controllers.InitDashboardRoutes(adminAPIGroup, "admin")
		routes.InitNotificationRoutes(adminAPIGroup)
	}

	// Member Authentication and Routes
	memberAuthGroup := server.Group("/auth/member")
	{
		memberAuthGroup.POST("/login", controllers.LoginMember)
	}

	// Member API Group
	memberAPIGroup := server.Group("/api/member")
	memberAPIGroup.Use(middleware.AuthMiddleware("member"))
	{
		// controllers.InitMemberProjectRoutes(memberAPIGroup)
		controllers.InitMemberTaskRoutes(memberAPIGroup)
		controllers.InitDashboardRoutes(memberAPIGroup, "member")
		routes.MemberProjectRoutes(memberAPIGroup)
		routes.InitNotificationRoutes(memberAPIGroup)
	}

	//controllers.InitCommentRoutes(apiGroup)
	//controllers.InitNotificationRoutes(apiGroup)

	//server.GET("/ws", controllers.HandleWebSocket)
	//server.GET("/ws", controllers.HandleWebSocket)

	// // Serve static files
	// server.Static("/static", "./static")
	// server.NoRoute(func(c *gin.Context) {
	// 	c.File("./static/index.html")
	// })

	// Configure HTTP server with Gin
	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: server,
	}
	// Start server in a separate goroutine to handle graceful shutdown
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	log.Println("Server is running on port 8080")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Block until a signal is received

	log.Println("Shutting down server...")

	// Gracefully shutdown the server with a timeout of 5 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}
