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
	pageSize := c.Query("page_size")
	pageState := c.Query("page_state") // This will hold the paging state for the next page

	// Convert pageSize to an integer
	size, err := strconv.Atoi(pageSize)
	if err != nil || size <= 0 {
		size = 10 // Default size
	}

	var projects []utils.Project

	userID, err := getUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	query := "SELECT id, name, description, start_date, due_date,status FROM projects where owner_id = ?"
	stmt := utils.Session.Query(query, userID).PageSize(size)
	// If a page state is provided, set it
	if pageState != "" {
		stmt = stmt.PageState([]byte(pageState))
	}

	iter := stmt.Iter()

	for {
		var project utils.Project

		if !iter.Scan(&project.ID, &project.Name, &project.Description, &project.StartDate, &project.DueDate, &project.Status) {
			break
		}

		projects = append(projects, project)
	}
	// Get the next paging state from the iterator
	nextPageState := iter.PageState()

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
	totalPages := (total + size - 1) / size

	// Return the response
	response := gin.H{
		"projects":   projects,
		"next_token": nextPageState,
		"totalPages": totalPages,
	}
	c.JSON(http.StatusOK, response)
}
