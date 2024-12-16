// Dashboard.go
package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go/task_management/backend/utils"
)

// InitTaskRoutes initializes the task-related routes
func InitDashboardRoutes(apiGroup *gin.RouterGroup, role string) {
	if role == "admin" {
		apiGroup.GET("/dashboard", AdminDashboard)
	} else if role == "member" {
		apiGroup.GET("/dashboard", MemberDashboard)
		apiGroup.GET("/dashboard/tasks/weekly-summary", GetWeeklyTaskSummary)
	}
}

// Admin
// AdminDashboard fetches report details from the database and returns them as a JSON response
func AdminDashboard(c *gin.Context) {

	tokenString := c.GetHeader("Authorization")
	userID, err := utils.GetUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	var stats utils.Statistics

	// Query to retrieve statistics for a specific user
	query := `
        SELECT 
            (SELECT COUNT(*) FROM projects WHERE owner_id = $1) AS total_projects,
            (SELECT COUNT(*) FROM projects WHERE status = 'completed' AND owner_id = $1) AS completed_projects,
            (SELECT COUNT(*) FROM users WHERE admin_id = $1) AS total_users
    `

	// Execute the query
	err = utils.DBPool.QueryRow(context.Background(), query, userID).Scan(
		&stats.TotalProjects,
		&stats.CompletedProjects,
		&stats.TotalUsers,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve statistics"})
		return
	}

	// Return the statistics as JSON
	c.JSON(http.StatusOK, stats)
}

// Member
// MemberDashboard fetches dashboard details from the database and returns them as a JSON response
func MemberDashboard(c *gin.Context) {
	// Extract member ID from token
	tokenString := c.GetHeader("Authorization")
	memberID, err := utils.GetUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Member ID not found in context"})
		return
	}

	var stats struct {
		TotalTasks       int             `json:"total_tasks"`
		InProgressTasks  int             `json:"in_progress_tasks"`
		OverdueTasks     int             `json:"overdue_tasks"`
		CompletedTasks   int             `json:"completed_tasks"`
		RecentTasks      []utils.Task    `json:"recent_tasks"`
		RecentUpdates    []utils.Updates `json:"recent_updates"`
		ImportantAspects []string        `json:"important_aspects"`
	}

	// Query statistics
	statsQuery := `
		SELECT 
			(SELECT COUNT(*) FROM tasks WHERE assigned_user_id = $1) AS total_tasks,
			(SELECT COUNT(*) FROM tasks WHERE assigned_user_id = $1 AND status IN ('in progress', 'review') AND (due_date IS NULL OR due_date > CURRENT_TIMESTAMP)) AS in_progress_tasks,
			(SELECT COUNT(*) FROM tasks WHERE assigned_user_id = $1 AND status = 'completed') AS completed_tasks,
			(SELECT COUNT(*) FROM tasks WHERE assigned_user_id = $1 AND status != 'completed' AND due_date IS NOT NULL AND due_date < CURRENT_TIMESTAMP) AS overdue_tasks
	`

	err = utils.DBPool.QueryRow(context.Background(), statsQuery, memberID).Scan(
		&stats.TotalTasks,
		&stats.InProgressTasks,
		&stats.CompletedTasks,
		&stats.OverdueTasks,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve member statistics"})
		return
	}

	// Query recent tasks
	tasksQuery := `
		SELECT id, title, status, priority, start_date, due_date 
		FROM tasks 
		WHERE assigned_user_id = $1 
		ORDER BY updated_at DESC LIMIT 5
	`
	rows, err := utils.DBPool.Query(context.Background(), tasksQuery, memberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch recent tasks"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var task utils.Task
		var startDate, dueDate *time.Time

		err := rows.Scan(&task.ID, &task.Title, &task.Status, &task.Priority, &startDate, &dueDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan recent task"})
			return
		}
		if startDate != nil {
			task.StartDate = startDate.Format("2006-01-02 15:04:05")
		}
		if dueDate != nil {
			task.DueDate = dueDate.Format("2006-01-02 15:04:05")
		}
		stats.RecentTasks = append(stats.RecentTasks, task)
	}

	// Call the GetMemberRecentUpdates function to fetch recent updates
	updates, err := GetMemberRecentUpdates(memberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch recent updates"})
		return
	}
	stats.RecentUpdates = updates

	// Add other important aspects (customize as needed)
	stats.ImportantAspects = []string{
		"Focus on overdue tasks.",
		"Collaborate with team members.",
		"Review tasks marked for review.",
	}

	// Return the statistics and recent data
	c.JSON(http.StatusOK, stats)
}

// member weekly task summary
func GetWeeklyTaskSummary(c *gin.Context) {

	tokenString := c.GetHeader("Authorization")
	memberID, err := utils.GetUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Member ID not found in context"})
		return
	}

	type TaskSummary struct {
		Day        string `json:"day"`
		Completed  int    `json:"completed"`
		InProgress int    `json:"in_progress"`
		Review     int    `json:"review"`
		Overdue    int    `json:"overdue"`
	}

	query := `
			SELECT
				TO_CHAR(dates.date, 'Day') AS day,
				COALESCE(SUM(CASE WHEN t.status = 'completed' THEN 1 ELSE 0 END), 0) AS completed,
				COALESCE(SUM(CASE WHEN t.status = 'in progress' THEN 1 ELSE 0 END), 0) AS in_progress,
				COALESCE(SUM(CASE WHEN t.status = 'review' THEN 1 ELSE 0 END), 0) AS review,
				COALESCE(SUM(CASE WHEN t.due_date < CURRENT_DATE AND t.status NOT IN ('completed', 'review') THEN 1 ELSE 0 END), 0) AS overdue
			FROM 
				GENERATE_SERIES(CURRENT_DATE - INTERVAL '6 days', CURRENT_DATE, '1 day') AS dates(date)
			LEFT JOIN tasks t ON t.assigned_user_id = $1 AND DATE(t.updated_at) = dates.date
			GROUP BY dates.date
			ORDER BY dates.date;
		`

	rows, err := utils.DBPool.Query(c, query, memberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch task summary"})
		return
	}
	defer rows.Close()

	var summaries []TaskSummary
	for rows.Next() {
		var summary TaskSummary
		if err := rows.Scan(&summary.Day, &summary.Completed, &summary.InProgress, &summary.Review, &summary.Overdue); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse task summary"})
			return
		}
		//summary.Day = summary.Day[:len(summary.Day)-1] // Trim extra spaces from day names
		summaries = append(summaries, summary)
	}

	c.JSON(http.StatusOK, gin.H{"data": summaries})
}
