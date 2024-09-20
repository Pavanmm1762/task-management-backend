// reportDetails.go
package controllers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go/task_management/backend/utils"
	"github.com/gocql/gocql"
)

// InitTaskRoutes initializes the task-related routes
func InitReportRoutes(router *gin.RouterGroup) {
	router.GET("/report-lists", getReports)
}

// getTasks fetches report details from the database and returns them as a JSON response
func getReports(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")
	var reports []utils.Reports

	userID, err := getUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	iter := utils.Session.Query(`
	SELECT id, name, status
	FROM projects where owner_id=?`,
		userID).Iter()

	for {
		var projectID gocql.UUID
		var projectName, status string

		if !iter.Scan(&projectID, &projectName, &status) {
			if err := iter.Close(); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			break
		}

		fmt.Printf("Project ID: %v, Project Name: %v, Status: %v\n", projectID, projectName, status)

		// Declare these variables outside the task loop
		var totalTasks, completedTasks, progressSum int

		// Fetch tasks for each project
		taskIter := utils.Session.Query(`
			SELECT task_id, task_name, progress, status
			FROM tasks
			WHERE project_id = ?
			`, projectID).Iter()
		var taskID gocql.UUID
		var taskTitle, taskStatus string
		var taskProgress int
		for taskIter.Scan(&taskID, &taskTitle, &taskProgress, &taskStatus) {
			totalTasks++
			progressSum += taskProgress

			if taskStatus == "completed" {
				completedTasks++
			}
			fmt.Printf("Task ID: %v, Task Title: %v, Progress: %v, Status: %v\n", taskID, taskTitle, taskProgress, taskStatus)
		}

		var report utils.Reports
		report.ProjectName = projectName
		report.TotalTasks = totalTasks
		report.CompletedTasks = completedTasks
		if totalTasks > 0 {
			report.Progress = float64(progressSum) / float64(totalTasks)
		} else {
			report.Progress = 0.0
		}
		report.Status = status

		reports = append(reports, report)

		// Close the task iterator
		if err := taskIter.Close(); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
	}

	// Print the response before sending it
	fmt.Println("Response:", reports)
	c.JSON(200, reports)
}
