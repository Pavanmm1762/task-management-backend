package controllers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go/task_management/backend/utils"
	"github.com/gocql/gocql"
)

// get project details
func GetProjectDetails(c *gin.Context) {
	projectIDStr := c.Param("id")
	var project utils.Project

	// Parse the user ID
	projectId, err := gocql.ParseUUID(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	iter := utils.Session.Query("SELECT name, description, start_date, due_date, status FROM projects where id = ? ", projectId).Iter()

	if !iter.Scan(&project.Name, &project.Description, &project.StartDate, &project.DueDate, &project.Status) {
		if err := iter.Close(); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
	}

	if err := iter.Close(); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, project)
}

// get project tasks
func GetProjectTasks(c *gin.Context) {
	projectIDStr := c.Param("projectId")
	var tasks []utils.Task
	projectId, err := gocql.ParseUUID(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	query := utils.Session.Query("SELECT task_id,task_name,description,status FROM tasks WHERE project_id=?", projectId)
	iter := query.Iter()

	var task utils.Task
	for iter.Scan(&task.ID, &task.Title, &task.Description, &task.Status) {
		tasks = append(tasks, task)
	}

	if err := iter.Close(); err != nil {
		fmt.Println("Error fetching tasks:", err)
	}

	c.JSON(http.StatusOK, tasks)
}

// Update project
func UpdateProject(c *gin.Context) {
	projectID, err := gocql.ParseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	var updatedProject utils.Project // Assuming you have a User struct in utils package

	if err := c.ShouldBindJSON(&updatedProject); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Perform the update logic, for example, using CQL UPDATE statement
	query := "UPDATE projects SET name=?, description=?, start_date=?, due_date=?, status=? WHERE id=?"
	err = utils.Session.Query(query, updatedProject.Name, updatedProject.Description, updatedProject.StartDate, updatedProject.DueDate, updatedProject.Status, projectID).Exec()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project updated successfully"})
}

// Delete project
func deleteProject(c *gin.Context) {
	projectIDStr := c.Param("id")

	// Parse the user ID
	projectId, err := gocql.ParseUUID(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	query := "DELETE FROM projects WHERE id = ?"
	err = utils.Session.Query(query, projectId).Exec()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project deleted successfully"})
}
