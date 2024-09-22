// controllers.go
package controllers

import (
	"net/http"
	"strconv"

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
	router.DELETE("/projects/:projectId/tasks/:taskId", deleteTask)
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
	limitStr := c.Query("limit")

	// Default values
	limit := 5
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
	}

	// Get the last token if provided
	lastTokenStr := c.Query("last_token")
	var lastToken gocql.UUID
	var err error
	if lastTokenStr != "" {
		lastToken, err = gocql.ParseUUID(lastTokenStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid last_token"})
			return
		}
	}

	var projects []utils.Project
	var iter *gocql.Iter

	userID, err := getUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	if lastToken == (gocql.UUID{}) {
		// Fetch first page
		iter = utils.Session.Query("SELECT id, name, description, start_date, due_date,status FROM projects where owner_id = ? LIMIT ?", userID, limit).Iter()
	} else {
		// Fetch next page
		iter = utils.Session.Query("SELECT id, name, description, start_date, due_date,status FROM projects where owner_id = ? AND token(id) > token(?) LIMIT ? ", userID, lastToken, limit).Iter()

	}
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

	// Get total count of items (for pagination)
	var total int
	if err := utils.Session.Query("SELECT COUNT(*) FROM projects").Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	totalPages := (total + limit - 1) / limit

	// Return the response with the last project's ID for the next token
	var nextToken gocql.UUID
	if len(projects) > 0 {
		nextToken = projects[len(projects)-1].ID
	}
	// Return the response
	response := gin.H{
		"projects":   projects,
		"next_token": nextToken,
		"totalPages": totalPages,
	}
	c.JSON(http.StatusOK, response)
}
