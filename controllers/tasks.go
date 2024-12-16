// // tasks.go
package controllers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go/task_management/backend/utils"
	"github.com/go/task_management/backend/webSocket"
	"github.com/jackc/pgx/v4"
)

// InitTaskRoutes initializes routes for tasks
func InitTaskRoutes(router *gin.RouterGroup) {
	router.GET("/tasks", GetTasks)
	//router.GET("/projects/:id/tasks", GetProjectTasks)
	router.POST("/projects/:id/tasks", AddTaskToProject)
	router.GET("/projects/:projectId/tasks/:taskId", GetTaskDetails)
	router.PUT("/projects/:projectId/tasks/:taskId", UpdateTask)
	router.DELETE("/projects/:projectId/tasks/:taskId", deleteTask)
}

func InitMemberTaskRoutes(router *gin.RouterGroup) {
	router.GET("/tasks", GetMemberTasks)
	router.PUT("/:projectId/tasks/:taskId", UpdateMemberTask)
}

func GetTasks(c *gin.Context) {
	// Get query params
	userID := c.Query("user_id")
	projectID := c.Query("project_id")
	searchQuery := c.Query("search")
	status := c.Query("status")
	priority := c.Query("priority")
	startDate := c.Query("start_date")
	dueDate := c.Query("due_date")
	pageStr := c.Query("page")
	limitStr := c.Query("limit")

	// Get owner/admin ID from the token (assuming GetUserIdFromToken extracts owner_id from JWT)
	tokenString := c.GetHeader("Authorization")
	ownerID, err := utils.GetUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Owner ID not found"})
		return
	}

	// Default pagination values
	page, limit := 1, 10
	if pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
			return
		}
	}

	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit number"})
			return
		}
	}
	offset := (page - 1) * limit

	// SQL Query Construction with Prepared Statements
	var tasks []utils.Task
	var params []interface{}
	var queryFilters []string

	// Base Query: Ensure tasks are linked to projects owned by the current admin (owner_id)
	query := `
        SELECT t.id, t.title, t.description, t.priority, t.status, t.start_date, t.due_date, 
               t.created_at, t.updated_at, t.project_id, 
               u.id AS assigned_user_id, u.full_name AS assigned_user_name, u.designation, u.department, p.name AS project_name
        FROM tasks t
        LEFT JOIN users u ON t.assigned_user_id = u.id
        LEFT JOIN projects p ON t.project_id = p.id
        WHERE p.owner_id = $1
    `
	params = append(params, ownerID)

	// Dynamically building filters
	paramIndex := 2 // Start from 2 because owner_id is $1

	if projectID != "" {
		queryFilters = append(queryFilters, fmt.Sprintf("t.project_id = $%d", paramIndex))
		params = append(params, projectID)
		paramIndex++
	}

	if userID == "null" {
		queryFilters = append(queryFilters, "t.assigned_user_id IS NULL")
	} else if userID != "" {
		queryFilters = append(queryFilters, fmt.Sprintf("u.id = $%d", paramIndex))
		params = append(params, userID)
		paramIndex++
	}

	if status != "" {
		queryFilters = append(queryFilters, fmt.Sprintf("t.status = $%d", paramIndex))
		params = append(params, status)
		paramIndex++
	}

	if priority != "" {
		queryFilters = append(queryFilters, fmt.Sprintf("t.priority = $%d", paramIndex))
		params = append(params, priority)
		paramIndex++
	}

	if searchQuery != "" {
		queryFilters = append(queryFilters, fmt.Sprintf("t.title ILIKE $%d", paramIndex))
		params = append(params, "%"+searchQuery+"%")
		paramIndex++
	}

	// Handle start_date and due_date
	if startDate != "" {
		queryFilters = append(queryFilters, fmt.Sprintf("t.start_date >= $%d", paramIndex))
		params = append(params, startDate)
		paramIndex++
	}

	if dueDate != "" {
		queryFilters = append(queryFilters, fmt.Sprintf("t.due_date <= $%d", paramIndex))
		params = append(params, dueDate)
		paramIndex++
	}

	// Combine the filters into the query
	if len(queryFilters) > 0 {
		query += " AND " + strings.Join(queryFilters, " AND ")
	}

	// Add pagination and ordering
	query += fmt.Sprintf(" ORDER BY t.created_at DESC LIMIT $%d OFFSET $%d", paramIndex, paramIndex+1)
	params = append(params, limit, offset)

	// Execute the query
	rows, err := utils.DBPool.Query(context.Background(), query, params...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
		return
	}
	defer rows.Close()

	// Fetch and process tasks
	for rows.Next() {
		var task utils.Task
		var createdAt, updatedAt time.Time
		var startDate, dueDate *time.Time

		// Scan the row data
		if err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Priority, &task.Status,
			&startDate, &dueDate, &createdAt, &updatedAt, &task.ProjectID, &task.AssignedUserID,
			&task.AssignedUserName, &task.Designation, &task.Department, &task.ProjectName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan task"})
			return
		}

		// Format CreatedAt and UpdatedAt
		task.CreatedAt = createdAt.Format("02-01-2006 3:04:05 PM")
		task.UpdatedAt = updatedAt.Format("02-01-2006 3:04:05 PM")

		// Format StartDate and DueDate
		if startDate != nil {
			task.StartDate = startDate.Format("02-01-2006 3:04:05 PM")
		} else {
			task.StartDate = ""
		}

		if dueDate != nil {
			task.DueDate = dueDate.Format("02-01-2006 3:04:05 PM")
		} else {
			task.DueDate = ""
		}

		tasks = append(tasks, task)
	}

	// Get total count for pagination, using the same filters but without pagination limits
	countQuery := `
        SELECT COUNT(*) 
        FROM tasks t
        LEFT JOIN users u ON t.assigned_user_id = u.id
        LEFT JOIN projects p ON t.project_id = p.id
        WHERE p.owner_id = $1
    `
	if len(queryFilters) > 0 {
		countQuery += " AND " + strings.Join(queryFilters, " AND ")
	}

	var total int
	err = utils.DBPool.QueryRow(context.Background(), countQuery, params[:len(params)-2]...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count tasks"})
		return
	}

	// Calculate pagination details
	totalPages := (total + limit - 1) / limit
	hasNext := page < totalPages
	hasPrev := page > 1

	// Return the tasks and pagination details
	c.JSON(http.StatusOK, gin.H{
		"tasks":      tasks,
		"page":       page,
		"limit":      limit,
		"total":      total,
		"totalPages": totalPages,
		"hasNext":    hasNext,
		"hasPrev":    hasPrev,
	})
}

// Function to get member tasks with offset and limit for infinite scroll
func GetMemberTasks(c *gin.Context) {
	// Get query params
	offset := c.DefaultQuery("offset", "0") // Default to 0 if not provided
	limit := c.DefaultQuery("limit", "10")  // Default to 10 tasks per page

	// Convert to integers
	offsetInt, err := strconv.Atoi(offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offset"})
		return
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit"})
		return
	}

	// Get owner/admin ID from the token (assuming GetUserIdFromToken extracts owner_id from JWT)
	tokenString := c.GetHeader("Authorization")
	userID, err := utils.GetUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Owner ID not found"})
		return
	}

	// SQL Query Construction with Prepared Statements
	var tasks []utils.Task
	query := `
        SELECT t.id, t.title, t.description, t.priority, t.status, t.start_date, t.due_date, 
               t.created_at, t.updated_at, t.project_id, 
               u.id AS assigned_user_id, u.full_name AS assigned_user_name, u.designation, u.department, p.name AS project_name
        FROM tasks t
        LEFT JOIN users u ON t.assigned_user_id = u.id
        LEFT JOIN projects p ON t.project_id = p.id
        WHERE t.assigned_user_id = $1
        ORDER BY t.created_at DESC
        LIMIT $2 OFFSET $3
    `
	// Execute the query with offset and limit
	rows, err := utils.DBPool.Query(context.Background(), query, userID, limitInt, offsetInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
		return
	}
	defer rows.Close()

	// Process tasks
	for rows.Next() {
		var task utils.Task
		var createdAt, updatedAt time.Time
		var startDate, dueDate *time.Time

		if err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Priority, &task.Status,
			&startDate, &dueDate, &createdAt, &updatedAt, &task.ProjectID, &task.AssignedUserID,
			&task.AssignedUserName, &task.Designation, &task.Department, &task.ProjectName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan task"})
			return
		}

		// Format dates
		task.CreatedAt = createdAt.Format("02-01-2006 3:04:05 PM")
		task.UpdatedAt = updatedAt.Format("02-01-2006 3:04:05 PM")
		if startDate != nil {
			task.StartDate = startDate.Format("02-01-2006 3:04:05 PM")
		} else {
			task.StartDate = ""
		}
		if dueDate != nil {
			task.DueDate = dueDate.Format("02-01-2006 3:04:05 PM")
		} else {
			task.DueDate = ""
		}

		tasks = append(tasks, task)
	}

	// Return tasks
	c.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
	})
}

// get task details
func GetTaskDetails(c *gin.Context) {
	projectID := c.Param("projectId")
	taskID := c.Param("taskId")

	var err error
	var task utils.Task

	// Query to fetch task details by task ID and project ID
	query := `
        SELECT t.id, t.title, t.description, t.priority, t.status, t.start_date, t.due_date, 
               t.created_at, t.updated_at, t.project_id, 
               u.id AS assigned_user_id, u.full_name AS assigned_user_name, u.designation, u.department,
               p.name AS project_name
        FROM tasks t
        LEFT JOIN users u ON t.assigned_user_id = u.id
        LEFT JOIN projects p ON t.project_id = p.id
        WHERE t.id = $1 AND t.project_id = $2
    `
	var createdAt, updatedAt time.Time
	var startDate, dueDate *time.Time

	err = utils.DBPool.QueryRow(context.Background(), query, taskID, projectID).Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&task.Priority,
		&task.Status,
		&startDate,
		&dueDate,
		&createdAt,
		&updatedAt,
		&task.ProjectID,
		&task.AssignedUserID,
		&task.AssignedUserName,
		&task.Designation,
		&task.Department,
		&task.ProjectName,
	)

	if err != nil {
		// Handle case where the task is not found
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
		// Handle other errors
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve task details"})
		log.Print(err)
		return
	}

	task.CreatedAt = createdAt.Format("02-01-2006 3:04:05 PM")
	task.UpdatedAt = updatedAt.Format("02-01-2006 3:04:05 PM")

	if startDate != nil {
		task.StartDate = startDate.Format("02-01-2006 3:04:05 PM")
	} else {
		task.StartDate = ""
	}

	if dueDate != nil {
		task.DueDate = dueDate.Format("02-01-2006 3:04:05 PM")
	} else {
		task.DueDate = ""
	}

	// Respond with the task details
	c.JSON(http.StatusOK, task)
}

// Add a task to a project
func AddTaskToProject(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")

	projectID := c.Param("id")
	var task utils.Task

	// Parse JSON body for the task
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task data"})
		return
	}

	// Extract user ID from token
	userID, err := utils.GetUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: User ID not found"})
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start a transaction
	tx, err := utils.DBPool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin transaction"})
		return
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx) // Rollback on panic
			panic(p)
		} else if err != nil {
			tx.Rollback(ctx) // Rollback on error
		}
	}()

	// Generate the next task ID
	newID, err := utils.GetNextTaskID(tx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate task ID"})
		return
	}
	task.ID = newID

	var startDateParam, dueDateParam interface{}

	if task.StartDate != "" {
		startDate, err := time.Parse("2006-01-02 15:04", task.StartDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format. Expected format: YYYY-MM-DD HH:MM"})
			return
		}
		startDateParam = startDate // Assign parsed start date
	} else {
		startDateParam = nil // Pass null if empty
	}

	if task.DueDate != "" {
		dueDate, err := time.Parse("2006-01-02 15:04", task.DueDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid due date format. Expected format: YYYY-MM-DD HH:MM"})
			return
		}
		dueDateParam = dueDate // Assign parsed due date
	} else {
		dueDateParam = nil // Pass null if empty
	}

	if task.AssignedUserID == nil || *task.AssignedUserID == "" {
		task.AssignedUserID = nil
	}

	// Insert the task into PostgreSQL
	query := `
		INSERT INTO tasks (id, project_id, title, description, priority, status, start_date, due_date, assigned_user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
	`
	_, err = tx.Exec(ctx, query, task.ID, projectID, task.Title, task.Description, task.Priority, task.Status, startDateParam, dueDateParam, task.AssignedUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add task"})
		return
	}

	// Log the action in recent_updates table
	//description := fmt.Sprintf("Added a new task '%s' for member '%s'", request.TaskName, request.AssignedUserID)
	var assignedUserID string
	if task.AssignedUserID != nil {
		assignedUserID = *task.AssignedUserID
	} else {
		assignedUserID = "unassigned" // or any default value you'd like to set
	}
	err = LogDetailedUpdate(
		userID,
		"Added a new task",
		projectID,
		task.ID,
		"admin",
		"created",
		"",
		fmt.Sprintf("Task Name: %s, Assigned To: %s, Due Date: %s, Status: %s",
			task.Title,
			assignedUserID,
			dueDateParam,
			task.Status),
	)
	if err != nil {
		log.Println("Failed to log update:", err)
	}

	// Commit transaction
	if err = tx.Commit(context.Background()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Task created successfully",
		"task":    task,
	})
}

// Update task details
func UpdateTask(c *gin.Context) {
	projectID := c.Param("projectId")
	taskID := c.Param("taskId")
	var updatedTask utils.Task
	var currentTask utils.Task

	// Fetch the current task data
	var startDate, dueDate *time.Time
	queryGet := `SELECT title, description, priority, status, start_date, due_date, assigned_user_id FROM tasks WHERE project_id=$1 AND id=$2`
	err := utils.DBPool.QueryRow(context.Background(), queryGet, projectID, taskID).Scan(
		&currentTask.Title,
		&currentTask.Description,
		&currentTask.Priority,
		&currentTask.Status,
		&startDate,
		&dueDate,
		&currentTask.AssignedUserID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch current task data"})
		return
	}
	var current_start_date string
	if startDate != nil {
		current_start_date = startDate.Format("2006-01-02 15:04")
	} else {
		current_start_date = "N/A"
	}

	// Check if dueDate is nil
	var current_due_date string
	if dueDate != nil {
		current_due_date = dueDate.Format("2006-01-02 15:04")
	} else {
		current_due_date = "N/A"
	}

	// Parse the request body
	if err := c.ShouldBindJSON(&updatedTask); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task data"})
		return
	}

	updatedBy := c.GetString("userID")
	role := c.GetString("role")

	var startDateParam, dueDateParam interface{}

	if updatedTask.StartDate != "" {
		startDate, err := time.Parse("2006-01-02 15:04", updatedTask.StartDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format. Expected format: YYYY-MM-DD HH:MM"})
			return
		}
		startDateParam = startDate // Assign parsed start date
	} else {
		startDateParam = nil // Pass null if empty
	}

	if updatedTask.DueDate != "" {
		dueDate, err := time.Parse("2006-01-02 15:04", updatedTask.DueDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid due date format. Expected format: YYYY-MM-DD HH:MM"})
			return
		}
		dueDateParam = dueDate // Assign parsed due date
	} else {
		dueDateParam = nil // Pass null if empty
	}
	if updatedTask.AssignedUserID == nil || *updatedTask.AssignedUserID == "" {
		updatedTask.AssignedUserID = nil
	}

	// Perform the update in PostgreSQL
	query := `
		UPDATE tasks 
		SET title=$1, description=$2, priority=$3, status=$4, start_date=$5, due_date=$6, assigned_user_id=$7, updated_at=NOW()
		WHERE project_id=$8 AND id=$9
	`
	_, err = utils.DBPool.Exec(context.Background(), query, updatedTask.Title, updatedTask.Description, updatedTask.Priority, updatedTask.Status, startDateParam, dueDateParam, updatedTask.AssignedUserID, projectID, taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		return
	}

	// Notify the assigned user if their assignment has changed
	if updatedTask.AssignedUserID != nil && (currentTask.AssignedUserID == nil || *updatedTask.AssignedUserID != *currentTask.AssignedUserID) {
		message := fmt.Sprintf("You have been assigned to the task '%s' in project '%s'.", updatedTask.Title, projectID)
		notification := utils.Notification{
			UserID:    *updatedTask.AssignedUserID,
			Message:   message,
			Type:      "task_assignment",
			IsRead:    false,
			CreatedAt: time.Now(),
		}

		if err := CreateNotification(notification); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create notification"})
			return
		}

		// Send WebSocket notification
		if err := webSocket.WSServer.SendMessage(*updatedTask.AssignedUserID, message); err != nil {
			log.Printf("WebSocket notification failed: %v", err)
		}

	}

	// Log changes entry
	logChanges := func(updatedBy, projectID, taskID, role string, changes []string) {
		afterValue := fmt.Sprintf("Updated: %s", strings.Join(changes, ", "))
		err := LogDetailedUpdate(updatedBy, "Task updated", projectID, taskID, role, "updated", "", afterValue)
		if err != nil {
			fmt.Printf("Failed to log changes: %v\n", err)
		}
	}

	// Collect changes into a slice
	var changes []string

	// Compare and collect changes
	if updatedTask.Title != currentTask.Title {
		changes = append(changes, fmt.Sprintf("title from '%s' to '%s'", currentTask.Title, updatedTask.Title))
	}
	if updatedTask.Description != currentTask.Description {
		changes = append(changes, fmt.Sprintf("description from '%s' to '%s'", currentTask.Description, updatedTask.Description))
	}
	if updatedTask.Priority != currentTask.Priority {
		changes = append(changes, fmt.Sprintf("priority from '%s' to '%s'", currentTask.Priority, updatedTask.Priority))
	}
	if updatedTask.Status != currentTask.Status {
		changes = append(changes, fmt.Sprintf("status from '%s' to '%s'", currentTask.Status, updatedTask.Status))
	}
	if startDateParam != nil && updatedTask.StartDate != current_start_date {
		changes = append(changes, fmt.Sprintf("start_date from '%s' to '%s'", current_start_date, updatedTask.StartDate))
	}
	if dueDateParam != nil && updatedTask.DueDate != current_due_date {
		changes = append(changes, fmt.Sprintf("due_date from '%s' to '%s'", current_due_date, updatedTask.DueDate))
	}
	if updatedTask.AssignedUserID != nil || currentTask.AssignedUserID != nil {
		// If one is nil and the other isn't, log the change
		if updatedTask.AssignedUserID == nil && currentTask.AssignedUserID != nil {
			changes = append(changes, fmt.Sprintf("assigned_user_id from '%s' to 'unassigned'", getValueOrDefault(currentTask.AssignedUserID)))
		} else if updatedTask.AssignedUserID != nil && currentTask.AssignedUserID == nil {
			changes = append(changes, fmt.Sprintf("assigned_user_id from 'unassigned' to '%s'", getValueOrDefault(updatedTask.AssignedUserID)))
		} else if *updatedTask.AssignedUserID != *currentTask.AssignedUserID {
			changes = append(changes, fmt.Sprintf("assigned_user_id from '%s' to '%s'", getValueOrDefault(currentTask.AssignedUserID), getValueOrDefault(updatedTask.AssignedUserID)))
		}
	}

	// If there are any changes, log them in one go
	if len(changes) > 0 {
		logChanges(updatedBy, projectID, taskID, role, changes)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Task updated successfully.",
		"updatedProject": updatedTask,
	})
}

func getValueOrDefault(value *string) string {
	if value == nil {
		return "null"
	}
	return *value
}

// Update task details
func UpdateMemberTask(c *gin.Context) {
	projectID := c.Param("projectId")
	taskID := c.Param("taskId")
	var input struct {
		Status string `json:"status" binding:"required"`
	}

	// Parse the request body to extract only the status field
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or missing status field"})
		return
	}

	tokenString := c.GetHeader("Authorization")
	memberID, err := utils.GetUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Owner ID not found"})
		return
	}

	// Verify if the task is assigned to the authenticated member
	checkQuery := `
		SELECT id FROM tasks
		WHERE id=$1 AND project_id=$2 AND assigned_user_id=$3
	`
	row := utils.DBPool.QueryRow(context.Background(), checkQuery, taskID, projectID, memberID)

	var existingTaskID string
	if err := row.Scan(&existingTaskID); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to update this task"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify task ownership"})
		}
		return
	}

	// Perform the update in PostgreSQL
	query := `
		UPDATE tasks 
		SET status=$1, updated_at=NOW()
		WHERE project_id=$2 AND id=$3
	`
	_, err = utils.DBPool.Exec(context.Background(), query, input.Status, projectID, taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Task updated successfully.",
		"updatedProject": input,
	})
}

// Delete task by task ID
func deleteTask(c *gin.Context) {
	projectID := c.Param("projectId")
	taskID := c.Param("taskId")

	// Check if projectID or taskID are missing
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Task ID is required"})
		return
	}

	// SQL query to delete task
	query := `
        DELETE FROM tasks WHERE id=$1 AND project_id=$2
    `
	// Execute the query and check how many rows were affected
	result, err := utils.DBPool.Exec(context.Background(), query, taskID, projectID)
	if err != nil {
		// Handle internal server errors
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task due to server error"})
		return
	}

	rowsAffected := result.RowsAffected()

	if rowsAffected == 0 {
		// Task not found or already deleted
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found or it may have already been deleted"})
		return
	}

	// Task deleted successfully
	c.JSON(http.StatusOK, gin.H{"message": "Task deleted successfully"})
}

func UpdateOverdueTasks() error {
	// Get the current date and time
	now := time.Now()

	// Update tasks with a due date that has passed and is not completed
	query := `
        UPDATE tasks
        SET status = 'overdue'
        WHERE due_date < $1 AND status != 'completed';
    `

	_, err := utils.DBPool.Exec(context.Background(), query, now)
	return err
}
