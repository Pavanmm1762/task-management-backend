// // reportDetails.go
package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go/task_management/backend/utils"
	"github.com/jackc/pgx/v4"
)

// InitTaskRoutes initializes the task-related routes
func InitReportRoutes(router *gin.RouterGroup) {
	router.GET("/reports/overall", GetOverallReport)
	router.GET("/reports/projects/summaries", GetAllProjectSummaries)
	router.GET("/reports/projects/:id", GetProjectDetailsReport)
	router.GET("/reports/users/summaries", GetAllUsers)
	router.GET("/reports/users/:userId", GetUserReport)
}

// InitTaskRoutes initializes the task-related routes
func GetOverallReport(c *gin.Context) {
	// Get the user ID from the token
	tokenString := c.GetHeader("Authorization")
	userID, err := utils.GetUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: User ID not found"})
		return
	}

	report := utils.OverallReport{}

	// Database queries

	// Fetch total number of projects for the user
	err = utils.DBPool.QueryRow(context.Background(), `
        SELECT COUNT(*) 
        FROM projects 
        WHERE owner_id = $1
    `, userID).Scan(&report.TotalProjects)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch total projects"})
		return
	}

	// Fetch total number of tasks for the user's projects
	err = utils.DBPool.QueryRow(context.Background(), `
        SELECT COUNT(*)
        FROM tasks 
        JOIN projects ON tasks.project_id = projects.id
        WHERE projects.owner_id = $1
    `, userID).Scan(&report.TotalTasks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch total tasks"})
		return
	}

	// Fetch completed tasks
	err = utils.DBPool.QueryRow(context.Background(), `
        SELECT COUNT(*)
        FROM tasks 
        JOIN projects ON tasks.project_id = projects.id
        WHERE projects.owner_id = $1 AND tasks.status = 'completed'
    `, userID).Scan(&report.CompletedTasks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch completed tasks"})
		return
	}

	// Fetch in-progress tasks
	err = utils.DBPool.QueryRow(context.Background(), `
        SELECT COUNT(*)
        FROM tasks 
        JOIN projects ON tasks.project_id = projects.id
        WHERE projects.owner_id = $1 AND tasks.status = 'in progress'
    `, userID).Scan(&report.InProgressTasks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch in-progress tasks"})
		return
	}

	// Fetch overdue tasks (tasks where due_date < NOW() and status is not completed)
	err = utils.DBPool.QueryRow(context.Background(), `
        SELECT COUNT(*)
        FROM tasks 
        JOIN projects ON tasks.project_id = projects.id
        WHERE projects.owner_id = $1 AND tasks.status != 'completed' AND tasks.due_date < NOW()
    `, userID).Scan(&report.OverdueTasks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch overdue tasks"})
		return
	}

	// Fetch total users in the organization/team
	err = utils.DBPool.QueryRow(context.Background(), `
        SELECT COUNT(*)
        FROM users 
        WHERE admin_id = $1
    `, userID).Scan(&report.TotalUsers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch total users"})
		return
	}

	// Fetch completed projects
	err = utils.DBPool.QueryRow(context.Background(), `
        SELECT COUNT(*)
        FROM projects 
        WHERE owner_id = $1 AND status = 'completed'
    `, userID).Scan(&report.CompletedProjects)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch completed projects"})
		return
	}

	// Fetch in-progress projects
	err = utils.DBPool.QueryRow(context.Background(), `
        SELECT COUNT(*)
        FROM projects 
        WHERE owner_id = $1 AND status = 'in progress'
    `, userID).Scan(&report.InProgressProjects)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch in-progress projects"})
		return
	}

	// Fetch not started projects
	err = utils.DBPool.QueryRow(context.Background(), `
        SELECT COUNT(*)
        FROM projects 
        WHERE owner_id = $1 AND status = 'not started'
    `, userID).Scan(&report.NotStartedProjects)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch not started projects"})
		return
	}

	// Fetch active users
	err = utils.DBPool.QueryRow(context.Background(), `
        SELECT COUNT(*)
        FROM users 
        WHERE admin_id = $1 AND status = 'active'
    `, userID).Scan(&report.ActiveUsers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch active users"})
		return
	}

	// Fetch inactive users
	err = utils.DBPool.QueryRow(context.Background(), `
        SELECT COUNT(*)
        FROM users 
        WHERE admin_id = $1 AND status = 'inactive'
    `, userID).Scan(&report.InactiveUsers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch inactive users"})
		return
	}

	// Return the report data
	c.JSON(http.StatusOK, report)
}

// Get the report for project summaries
func GetAllProjectSummaries(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")
	userID, err := utils.GetUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: User ID not found"})
		return
	}

	pageStr := c.Query("page")
	limitStr := c.Query("limit")
	searchQuery := c.Query("search") // Search by project name

	page, limit := 1, 5

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

	query := `
		SELECT id, name, description
		FROM projects
		WHERE owner_id = $1
	`
	if searchQuery != "" {
		query += `AND name ILIKE '%' || $4 || '%' `
	}
	query += `
		ORDER BY name
		LIMIT $2 OFFSET $3;`

	// Fetch project summaries
	var rows pgx.Rows
	if searchQuery != "" {
		rows, err = utils.DBPool.Query(context.Background(), query, userID, limit, offset, searchQuery)
	} else {
		rows, err = utils.DBPool.Query(context.Background(), query, userID, limit, offset)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch projects"})
		return
	}
	defer rows.Close()

	var projectSummaries []utils.ProjectSummary

	for rows.Next() {
		var project utils.ProjectSummary

		err := rows.Scan(&project.ID, &project.Name, &project.Description)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch project data"})
			return
		}

		// Fetch task statuses for each project
		project.CompletedTasks, err = GetTaskCountByStatus(project.ID, "completed")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch completed tasks"})
			return
		}

		project.InProgressTasks, err = GetTaskCountByStatus(project.ID, "in progress")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch in-progress tasks"})
			return
		}

		project.OverdueTasks, err = GetOverdueTaskCount(project.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch overdue tasks"})
			return
		}
		// Fetch total users and tasks for the project
		project.TotalUsers, err = GetTotalUsersForProject(project.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch total users"})
			return
		}

		project.TotalTasks, err = GetTotalTasksForProject(project.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch total tasks"})
			return
		}

		projectSummaries = append(projectSummaries, project)
	}

	// Count total number of projects, including the search filter if applied
	countQuery := `SELECT COUNT(*) FROM projects WHERE owner_id = $1 `
	if searchQuery != "" {
		countQuery += `AND name ILIKE '%' || $2 || '%'`
	}

	var total int
	if searchQuery != "" {
		err = utils.DBPool.QueryRow(context.Background(), countQuery, userID, searchQuery).Scan(&total)
	} else {
		err = utils.DBPool.QueryRow(context.Background(), countQuery, userID).Scan(&total)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count projects"})
		return
	}

	// Calculate total pages
	totalPages := (total + limit - 1) / limit
	hasNext := page < totalPages
	hasPrev := page > 1

	c.JSON(http.StatusOK, gin.H{
		"projects":   projectSummaries,
		"page":       page,
		"limit":      limit,
		"total":      total,
		"totalPages": totalPages,
		"hasNext":    hasNext,
		"hasPrev":    hasPrev,
	})
}

// Get the report for individual project
func GetProjectDetailsReport(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")
	userID, err := utils.GetUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: User ID not found"})
		return
	}
	projectID := c.Param("id")
	// Fetch project info
	project, err := GetProjectInfo(projectID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch project information"})
		return
	}

	// Fetch project tasks
	tasks, err := GetProjectTasks(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch project tasks"})
		return
	}

	// Fetch users assigned to the project
	users, err := GetProjectUsers(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch project users"})
		return
	}

	// Return detailed project information
	c.JSON(http.StatusOK, gin.H{
		"project": project,
		"tasks":   tasks,
		"users":   users,
	})
}

// GetTaskCountByStatus fetches task count by status for a specific project
func GetTaskCountByStatus(projectID string, status string) (int, error) {
	var count int
	err := utils.DBPool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM tasks
		WHERE project_id = $1 AND status = $2
	`, projectID, status).Scan(&count)
	return count, err
}

// GetOverdueTaskCount fetches the number of overdue tasks for a project
func GetOverdueTaskCount(projectID string) (int, error) {
	var count int
	err := utils.DBPool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM tasks
		WHERE project_id = $1 AND status != 'completed' AND due_date < NOW()
	`, projectID).Scan(&count)
	return count, err
}

// GetTotalUsersForProject counts the total users associated with a project
func GetTotalUsersForProject(projectID string) (int, error) {
	var count int
	err := utils.DBPool.QueryRow(context.Background(), `
		SELECT COUNT(*) 
		FROM project_assignments 
		WHERE project_id = $1
	`, projectID).Scan(&count)
	return count, err
}

// GetTotalTasksForProject counts the total tasks associated with a project
func GetTotalTasksForProject(projectID string) (int, error) {
	var count int
	err := utils.DBPool.QueryRow(context.Background(), `
		SELECT COUNT(*) 
		FROM tasks 
		WHERE project_id = $1
	`, projectID).Scan(&count)
	return count, err
}

// GetProjectInfo fetches basic information about the project
func GetProjectInfo(projectID, userId string) (utils.Project, error) {

	var project utils.Project

	var createdAt, updatedAt time.Time
	var startDate, dueDate *time.Time

	err := utils.DBPool.QueryRow(context.Background(), `
		SELECT id, name, description, start_date, due_date, status, created_at, updated_at
		FROM projects
		WHERE id = $1 AND owner_id = $2
	`, projectID, userId).Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&startDate,
		&dueDate,
		&project.Status,
		&createdAt,
		&updatedAt,
	)
	project.StartDate = startDate.Format("02-01-2006 3:04:05 PM")
	project.DueDate = dueDate.Format("02-01-2006 3:04:05 PM")
	project.CreatedAt = createdAt.Format("02-01-2006 3:04:05 PM")
	project.UpdatedAt = updatedAt.Format("02-01-2006 3:04:05 PM")

	return project, err
}

// GetProjectTasks fetches tasks assigned to a specific project
func GetProjectTasks(projectID string) ([]utils.Task, error) {

	rows, err := utils.DBPool.Query(context.Background(), `
		SELECT id, title, description, status, due_date
		FROM tasks
		WHERE project_id = $1
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []utils.Task

	var dueDate *time.Time

	for rows.Next() {
		var task utils.Task
		err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Description,
			&task.Status,
			&dueDate,
		)

		if err != nil {
			return nil, err
		}

		if dueDate != nil {
			task.DueDate = dueDate.Format("02-01-2006 3:04:05 PM")
		} else {
			task.DueDate = ""
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetProjectUsers fetches users assigned to a specific project
func GetProjectUsers(projectID string) ([]utils.User, error) { // Assuming a struct User exists in utils
	rows, err := utils.DBPool.Query(context.Background(), `
		SELECT users.id, users.username, users.email, users.role
		FROM users
		JOIN project_assignments ON project_assignments.user_id = users.id
		WHERE project_assignments.project_id = $1
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []utils.User

	for rows.Next() {
		var user utils.User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Role,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// GetAllUsers fetches users report
func GetAllUsers(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")
	userID, err := utils.GetUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: User ID not found"})
		return
	}

	pageStr := c.Query("page")
	limitStr := c.Query("limit")
	searchQuery := c.Query("search") // Search by project name

	page, limit := 1, 5

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

	query := `
        SELECT 
            id, 
			full_name,
            username,  
            role,
            (SELECT COUNT(*) FROM tasks WHERE assigned_user_id = users.id AND status = 'completed') AS completed_tasks,
            (SELECT COUNT(*) FROM tasks WHERE assigned_user_id = users.id AND status = 'in progress') AS in_progress_tasks,
            (SELECT COUNT(*) FROM tasks WHERE assigned_user_id = users.id AND due_date < NOW() AND status != 'completed') AS overdue_tasks
        FROM users
		WHERE admin_id = $1
    `
	if searchQuery != "" {
		query += `AND username ILIKE '%' || $4 || '%' `
	}
	query += `
		ORDER BY username
		LIMIT $2 OFFSET $3;`

	var rows pgx.Rows
	if searchQuery != "" {
		rows, err = utils.DBPool.Query(context.Background(), query, userID, limit, offset, searchQuery)
	} else {
		rows, err = utils.DBPool.Query(context.Background(), query, userID, limit, offset)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	defer rows.Close()

	var users []utils.UsersReport
	for rows.Next() {
		var user utils.UsersReport
		err := rows.Scan(&user.ID, &user.FullName, &user.Username, &user.Role, &user.CompletedTasks, &user.InProgressTasks, &user.OverdueTasks)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan user data"})
			return
		}

		users = append(users, user)
	}

	// Count total number of projects, including the search filter if applied
	countQuery := `SELECT COUNT(*) FROM users WHERE admin_id = $1 `
	if searchQuery != "" {
		countQuery += `AND username ILIKE '%' || $2 || '%'`
	}

	var total int
	if searchQuery != "" {
		err = utils.DBPool.QueryRow(context.Background(), countQuery, userID, searchQuery).Scan(&total)
	} else {
		err = utils.DBPool.QueryRow(context.Background(), countQuery, userID).Scan(&total)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count users"})
		return
	}

	// Calculate total pages
	totalPages := (total + limit - 1) / limit
	hasNext := page < totalPages
	hasPrev := page > 1

	c.JSON(http.StatusOK, gin.H{
		"users":      users,
		"page":       page,
		"limit":      limit,
		"total":      total,
		"totalPages": totalPages,
		"hasNext":    hasNext,
		"hasPrev":    hasPrev,
	})
}

// GetUserReport fetches user specific tasks
func GetUserReport(c *gin.Context) {
	userID := c.Param("userId")

	tokenString := c.GetHeader("Authorization")
	adminId, err := utils.GetUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: User ID not found"})
		return
	}

	query := `
         SELECT 
            username, 
            designation,
          	department,
			status,
            COALESCE((
                SELECT json_agg(task_data) FROM (
                    SELECT id, title, status, due_date FROM tasks WHERE assigned_user_id = $1
                ) AS task_data
            ), '[]'::json) AS task_completion_data
        FROM users WHERE id = $1 AND admin_id = $2;
    `

	var userReport utils.UsersReport
	var taskCompletionDataJSON []byte

	err = utils.DBPool.QueryRow(context.Background(), query, userID, adminId).Scan(
		&userReport.Username,
		&userReport.Designation,
		&userReport.Department,
		&userReport.Status,
		&taskCompletionDataJSON,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user report"})
		return
	}

	// Check if taskCompletionData is empty or null
	if len(taskCompletionDataJSON) == 0 {
		userReport.TaskCompletionData = []utils.Task{}
	} else {
		// Unmarshal the JSON data into the taskCompletionData struct
		err = json.Unmarshal(taskCompletionDataJSON, &userReport.TaskCompletionData)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing task completion data"})
			return
		}
	}

	// Return the final response
	c.JSON(http.StatusOK, userReport)
}
