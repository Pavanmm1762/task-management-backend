package controllers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go/task_management/backend/utils"
	"github.com/jackc/pgx/v4"
)

// get project details
func GetProjectDetails(c *gin.Context) {
	projectId := c.Param("id")
	var project utils.Project

	// Base query for projects with pagination
	query := `SELECT id, name, description, start_date, due_date, priority, status, created_at, updated_at, owner_id 
				FROM projects
				WHERE id = $1 `

	var startDate time.Time
	var dueDate time.Time
	var createdAt time.Time
	var updatedAt time.Time

	// Execute query with context for better control over cancellations and timeouts
	err := utils.DBPool.QueryRow(context.Background(), query, projectId).Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&startDate,
		&dueDate,
		&project.Priority,
		&project.Status,
		&createdAt,
		&updatedAt,
		&project.OwnerID,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	project.StartDate = startDate.Format("02-01-2006 3:04:05 PM")
	project.DueDate = dueDate.Format("02-01-2006 3:04:05 PM")
	project.CreatedAt = createdAt.Format("02-01-2006 3:04:05 PM")
	project.UpdatedAt = updatedAt.Format("02-01-2006 3:04:05 PM")

	c.JSON(http.StatusOK, project)
}

// UpdateProject updates the project details based on the project ID
func UpdateProject(c *gin.Context) {
	projectID := c.Param("id")

	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	var updatedProject utils.Project

	if err := c.ShouldBindJSON(&updatedProject); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if updatedProject.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project name cannot be empty"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() // Ensure the context is canceled to avoid memory leaks

	// Check if the project exists before updating
	var existingProject utils.Project
	queryCheck := `SELECT id, name FROM projects WHERE id = $1`
	err := utils.DBPool.QueryRow(ctx, queryCheck, projectID).Scan(&existingProject.ID, &existingProject.Name)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Parse start_date and due_date from string to time.Time
	startDate, err := time.Parse("2006-01-02 15:04", updatedProject.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format. Expected format: YYYY-MM-DD HH:MM"})
		return
	}

	dueDate, err := time.Parse("2006-01-02 15:04", updatedProject.DueDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid due date format. Expected format: YYYY-MM-DD HH:MM"})
		return
	}
	// Prepare the PostgreSQL update query with parameterized values
	query := `UPDATE projects 
              SET name = $1, description = $2, start_date = $3, due_date = $4, status = $5, priority = $6, updated_at = NOW() 
              WHERE id = $7`

	// Execute the query with the provided data
	_, err = utils.DBPool.Exec(ctx, query, updatedProject.Name, updatedProject.Description, startDate, dueDate, updatedProject.Status, updatedProject.Priority, projectID)

	// Check if there was an error while executing the query
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return the updated project as a response
	c.JSON(http.StatusOK, gin.H{
		"message":        "Project has been successfully updated.",
		"updatedProject": updatedProject,
	})
}

// Delete project
func DeleteProject(c *gin.Context) {
	projectIDStr := c.Param("id")

	// Check if the project ID is valid
	if projectIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	// Start a transaction
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := utils.DBPool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin transaction"})
		return
	}
	defer tx.Rollback(ctx) // Ensure rollback in case of failure

	// Check if the project exists
	var exists bool
	checkQuery := "SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1)"
	err = tx.QueryRow(ctx, checkQuery, projectIDStr).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check project existence"})
		return
	}

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Perform the delete operation for the project
	deleteQuery := "DELETE FROM projects WHERE id = $1"
	_, err = tx.Exec(ctx, deleteQuery, projectIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project: " + err.Error()})
		return
	}

	// Reset project assignment for the users associated with the deleted project
	updateQuery := "UPDATE users SET project_id = NULL WHERE project_id = $1"
	_, err = tx.Exec(ctx, updateQuery, projectIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update users associated with the project: " + err.Error()})
		return
	}

	// Commit the transaction if both operations succeed
	err = tx.Commit(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Return success message
	c.JSON(http.StatusOK, gin.H{"message": "Project deleted successfully"})
}

// FetchAssociatedUsers handles fetching users associated with a project
func AssociatedUsers(c *gin.Context) {
	projectID := c.Param("id")

	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Query to get associated users via the project_assignments table
	query := `
		SELECT u.id, u.username, u.full_name, u.designation, u.department 
		FROM users u
		JOIN project_assignments pa ON u.id = pa.user_id
		WHERE pa.project_id = $1`

	rows, err := utils.DBPool.Query(ctx, query, projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch associated users"})
		return
	}
	defer rows.Close()

	var users []utils.User
	for rows.Next() {
		var user utils.User
		if err := rows.Scan(&user.ID, &user.Username, &user.FullName, &user.Designation, &user.Department); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error scanning user data"})
			return
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading rows"})
		return
	}

	c.JSON(http.StatusOK, users)
}

// FetchAvailableUsers handles fetching all users
func AvailableUsers(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	adminID, err := utils.GetUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: User ID not found"})
		return
	}

	if adminID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing admin_id"})
		return
	}

	// Query to get all active users under the specified admin
	query := `
        SELECT id, full_name, admin_id, department
        FROM users
        WHERE admin_id = $1 AND status = 'active'
    `

	rows, err := utils.DBPool.Query(ctx, query, adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch available users"})
		return
	}
	defer rows.Close()

	var users []utils.User
	for rows.Next() {
		var user utils.User
		if err := rows.Scan(&user.ID, &user.FullName, &user.AdminID, &user.Department); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error scanning user data"})
			return
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading rows"})
		return
	}

	c.JSON(http.StatusOK, users)
}

// UpdateProjectMembers handles updating the members of a project with rollback support
func UpdateProjectMembers(c *gin.Context) {
	projectID := c.Param("id")

	// Parse request body to get user IDs
	var request struct {
		UserIDs []string `json:"user_ids"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Check if project ID is valid
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start a transaction
	tx, err := utils.DBPool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin transaction"})
		return
	}

	// Ensure to roll back the transaction in case of an error
	defer func() {
		if err != nil {
			tx.Rollback(context.Background())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction rolled back due to an error"})
		}
	}()

	// Query to check if any users being removed have assigned tasks
	taskCheckQuery := `
        SELECT assigned_user_id
        FROM tasks
        WHERE project_id = $1 AND assigned_user_id = ANY($2) AND status != 'completed'
    `
	var pendingUserIDs []string
	rows, err := tx.Query(ctx, taskCheckQuery, projectID, request.UserIDs)
	if err != nil {
		return // Error will trigger rollback
	}
	defer rows.Close()

	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return // Error will trigger rollback
		}
		pendingUserIDs = append(pendingUserIDs, userID)
	}

	// If there are users with pending tasks, return an error
	if len(pendingUserIDs) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Some users have pending tasks and cannot be removed: " + strings.Join(pendingUserIDs, ", ")})
		return
	}

	// Clear existing user associations for the project in project_assignments table
	clearQuery := "DELETE FROM project_assignments WHERE project_id = $1"
	_, err = tx.Exec(ctx, clearQuery, projectID)
	if err != nil {
		return // Error will trigger rollback
	}

	// Insert new project assignments for the users
	insertQuery := "INSERT INTO project_assignments (project_id, user_id, role) VALUES ($1, $2, 'member')"
	for _, userID := range request.UserIDs {
		_, err = tx.Exec(ctx, insertQuery, projectID, userID)
		if err != nil {
			return // Error will trigger rollback
		}
	}

	// Commit the transaction if everything is successful
	if err := tx.Commit(context.Background()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Fetch updated users to return
	updatedUsers, err := fetchAssociatedUsers(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated users"})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{"message": "Project members updated successfully", "updated_users": updatedUsers})
}

// Helper function to fetch associated users for a project
func fetchAssociatedUsers(projectID string) ([]utils.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
        SELECT u.id, u.full_name 
        FROM users u
        JOIN project_assignments pa ON u.id = pa.user_id
        WHERE pa.project_id = $1`
	rows, err := utils.DBPool.Query(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []utils.User
	for rows.Next() {
		var user utils.User
		if err := rows.Scan(&user.ID, &user.FullName); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}
