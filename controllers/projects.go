// controllers.go
package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go/task_management/backend/utils"
	"github.com/gocql/gocql"
)

// InitTaskRoutes initializes routes for projects
func InitProjectRoutes(router *gin.RouterGroup) {
	router.GET("/projects", GetProjects)
	router.POST("/project", CreateProject)
	router.GET("/project/:id", GetProjectDetails)
	router.POST("/project/add-task/:id", AddTaskToProject)
	router.GET("/project/tasks/:projectId", GetProjectTasks)
	router.DELETE("/project/:id", deleteProject)
	router.PUT("/project/:id", UpdateProject)
	router.PUT("/projects/:projectId/tasks/:taskId", UpdateTask)
}

// CreateProject creates a new project
func CreateProject(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")
	var project utils.Project
	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := getUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	project.ID = gocql.TimeUUID()

	if err := utils.Session.Query("INSERT INTO projects (id, name, description, start_date, due_date, status, owner_id) VALUES (?, ?, ?, ?, ?, ?, ?)",
		project.ID, project.Name, project.Description, project.StartDate, project.DueDate, project.Status, userID).Exec(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, project)
}

// GetProjects gets all projects
func GetProjects(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")

	var projects []utils.Project

	userID, err := getUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	iter := utils.Session.Query("SELECT id, name, description, start_date, due_date,status FROM projects where owner_id = ? ", userID).Iter()
	for {
		var project utils.Project

		if !iter.Scan(&project.ID, &project.Name, &project.Description, &project.StartDate, &project.DueDate, &project.Status) {
			break
		}

		projects = append(projects, project)
	}

	if err := iter.Close(); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, projects)
}
