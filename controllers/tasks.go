// tasks.go
package controllers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go/task_management/backend/utils"
	"github.com/gocql/gocql"
)

// InitTaskRoutes initializes routes for tasks
func InitTaskRoutes(router *gin.RouterGroup) {
	router.GET("/tasks", getTasks)
	router.DELETE("/tasks/:taskId", deleteTask)
}

func getTasks(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")
	var tasks []utils.Task

	userID, err := getUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	iter := utils.Session.Query(`
	SELECT id, name
	FROM projects where owner_id = ?`,
		userID).Iter()

	for {
		var projectID gocql.UUID
		var projectName string
		if !iter.Scan(&projectID, &projectName) {
			if err := iter.Close(); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			break
		}

		fmt.Printf("Project ID: %v, Project Name: %v\n", projectID, projectName)

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
			var task utils.Task

			task.ProjectName = projectName
			task.Title = taskTitle
			task.Progress = taskProgress
			task.Status = taskStatus

			tasks = append(tasks, task)

			fmt.Printf("Project name : %v,Task ID: %v, Task Title: %v, Progress: %v, Status: %v\n", projectName, taskID, taskTitle, taskProgress, taskStatus)
		}

		// Close the task iterator
		if err := taskIter.Close(); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
	}
	// Print the response before sending it
	fmt.Println("Response:", tasks)
	c.JSON(200, tasks)

	// iter := utils.Session.Query("SELECT task_id, task_name, progress, status FROM tasks").Iter()
	// for {
	// 	var task utils.Task

	// 	if !iter.Scan(&task.ID, &task.Title, &task.Progress, &task.status) {
	// 		break
	// 	}

	// 	tasks = append(tasks, task)
	// }

	// if err := iter.Close(); err != nil {
	// 	c.JSON(500, gin.H{"error": err.Error()})
	// 	return
	// }
	// // Print the response before sending it
	// fmt.Println("Response:", tasks)
	// c.JSON(200, tasks)
}

func AddTaskToProject(c *gin.Context) {
	projectIDStr := c.Param("id")
	var task utils.Task
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	// Parse the project ID
	projectId, err := gocql.ParseUUID(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	task.ID = gocql.TimeUUID()

	if err := utils.Session.Query("INSERT INTO tasks (task_id, project_id, task_name, description, progress, status) VALUES (?, ?, ?,?,?,?)", task.ID, projectId, task.Title, task.Description, task.Progress, task.Status).Exec(); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, task)
}

// Update task
func UpdateTask(c *gin.Context) {
	projectID, err := gocql.ParseUUID(c.Param("projectId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}
	taskID, err := gocql.ParseUUID(c.Param("taskId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	var updatedTask utils.Task // Assuming you have a User struct in utils package

	if err := c.ShouldBindJSON(&updatedTask); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Perform the update logic,  using CQL UPDATE statement
	query := "UPDATE tasks SET task_name=?, description=?, progress=?, status=? WHERE project_id=? AND task_id=?"
	err = utils.Session.Query(query, updatedTask.Title, updatedTask.Description, updatedTask.Progress, updatedTask.Status, projectID, taskID).Exec()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedTask)
}

// Delete task
func deleteTask(c *gin.Context) {
	taskIDStr := c.Param("taskId")

	// Parse the user ID
	taskID, err := gocql.ParseUUID(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	query := "DELETE FROM tasks WHERE task_id = ?"
	err = utils.Session.Query(query, taskID).Exec()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task deleted successfully"})
}
