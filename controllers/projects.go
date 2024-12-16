// controllers.go
package controllers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go/task_management/backend/utils"
	"github.com/jackc/pgx/v4"
	"github.com/lib/pq"
)

// InitTaskRoutes initializes routes for projects
func InitProjectRoutes(router *gin.RouterGroup) {
	router.GET("/projects", GetProjects)
	router.POST("/projects", CreateProject)
	router.GET("/project/:id", GetProjectDetails)
	//router.POST("/project/add-task/:id", AddTaskToProject)
	router.GET("/project/:id/members", AssociatedUsers)
	router.DELETE("/project/:id", DeleteProject)
	router.PUT("/project/:id", UpdateProject)
	router.PUT("/project/:id/members", UpdateProjectMembers)
	//router.PUT("/projects/:projectId/tasks/:taskId", UpdateTask)
	//router.DELETE("/projects/:projectId/tasks/:taskId", deleteTask)
}

// CreateProject creates a new project
func CreateProject(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")

	var project utils.Project
	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data: " + err.Error()})
		return
	}

	// Start a transaction
	tx, err := utils.DBPool.Begin(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin transaction"})
		return
	}
	// Defer rollback, but ensure to commit on success
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(context.Background()) // Rollback on panic
			panic(p)
		} else if err != nil {
			tx.Rollback(context.Background()) // Rollback on error
		}
	}()

	// Extract user ID from token
	userID, err := utils.GetUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: User ID not found"})
		return
	}

	// Generate new project ID
	newID, err := utils.GetNextProjectID(tx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating new project ID: " + err.Error()})
		return
	}
	project.ID = newID

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Parse start_date and due_date
	startDate, err := time.Parse("2006-01-02 15:04", project.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format. Expected format: YYYY-MM-DD HH:MM"})
		return
	}

	dueDate, err := time.Parse("2006-01-02 15:04", project.DueDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid due date format. Expected format: YYYY-MM-DD HH:MM"})
		return
	}

	// Set timestamps
	createdAt := time.Now().UTC()
	updatedAt := createdAt

	// Insert project into database
	query := `
        INSERT INTO projects 
        (id, name, description, start_date, due_date, priority, status, created_at, updated_at, owner_id) 
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `
	if _, err := tx.Exec(ctx, query,
		project.ID,
		project.Name,
		project.Description,
		startDate,
		dueDate,
		project.Priority,
		project.Status,
		createdAt,
		updatedAt,
		userID,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project: " + err.Error()})
		return
	}

	// Initialize task counter for the new project
	query = `INSERT INTO task_counter (project_id, task_value) VALUES ($1, 0)`
	if _, err := tx.Exec(ctx, query, project.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task counter: " + err.Error()})
		return
	}

	// Commit transaction
	if err = tx.Commit(context.Background()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Return success response
	c.JSON(http.StatusCreated, gin.H{
		"message": "Project created successfully",
		"project": project,
	})
}

// GetProjects get the project lists
func GetProjects(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")

	pageStr := c.Query("page")
	limitStr := c.Query("limit")
	searchQuery := c.Query("search") // Search by project name

	page, limit := 1, 5
	var err error

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

	userID, err := utils.GetUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: User ID not found"})
		return
	}

	offset := (page - 1) * limit

	// Base query for projects with pagination
	query := `SELECT 
                p.id, 
                p.name, 
                p.description, 
                p.start_date, 
                p.due_date, 
                p.priority, 
                p.status, 
                p.created_at, 
                p.updated_at, 
                p.owner_id, 
                COALESCE(ARRAY_AGG(u.id) FILTER (WHERE u.id IS NOT NULL), '{}') AS user_ids, 
                COALESCE(ARRAY_AGG(u.full_name) FILTER (WHERE u.full_name IS NOT NULL), '{}') AS user_names
              FROM 
                projects p
              LEFT JOIN 
                project_assignments pa ON p.id = pa.project_id
              LEFT JOIN 
                users u ON pa.user_id = u.id
              WHERE 
                p.owner_id = $1 
              `

	// Modify query if search parameter is provided
	if searchQuery != "" {
		query += `AND p.name ILIKE '%' || $4 || '%' `
	}

	query += `GROUP BY 
                p.id, p.name, p.description, p.start_date, p.due_date, p.priority, 
                p.status, p.created_at, p.updated_at, p.owner_id 
              ORDER BY p.created_at DESC 
              LIMIT $2 OFFSET $3;`

	// Execute the query
	var rows pgx.Rows
	if searchQuery != "" {
		rows, err = utils.DBPool.Query(context.Background(), query, userID, limit, offset, searchQuery)
	} else {
		rows, err = utils.DBPool.Query(context.Background(), query, userID, limit, offset)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve projects"})
		return
	}
	defer rows.Close()

	projects := []utils.Project{}

	for rows.Next() {
		var project utils.Project
		var startDate time.Time
		var dueDate time.Time
		var createdAt time.Time
		var updatedAt time.Time
		project.UserIds = []string{}
		project.UserNames = []string{}
		if err := rows.Scan(&project.ID, &project.Name, &project.Description, &startDate, &dueDate, &project.Priority, &project.Status, &createdAt, &updatedAt, &project.OwnerID, pq.Array(&project.UserIds), pq.Array(&project.UserNames)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan project"})
			return
		}
		project.StartDate = startDate.Format("02-01-2006 3:04:05 PM")
		project.DueDate = dueDate.Format("02-01-2006 3:04:05 PM")
		project.CreatedAt = createdAt.Format("02-01-2006 3:04:05 PM")
		project.UpdatedAt = updatedAt.Format("02-01-2006 3:04:05 PM")
		projects = append(projects, project)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred during row iteration"})
		return
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
		"projects":   projects,
		"page":       page,
		"limit":      limit,
		"total":      total,
		"totalPages": totalPages,
		"hasNext":    hasNext,
		"hasPrev":    hasPrev,
	})
}

func GetMemberProjects(c *gin.Context) {
	pageStr := c.Query("page")
	limitStr := c.Query("limit")
	searchQuery := c.Query("search") // Search by project name

	page, limit := 1, 5
	var err error

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

	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: user ID not found"})
		return
	}

	offset := (page - 1) * limit

	// Base query for projects with pagination
	query := `
		SELECT 
			p.id, 
			p.name, 
			p.description, 
			p.start_date, 
			p.due_date, 
			p.priority, 
			p.status, 
			p.created_at, 
			p.updated_at, 
			p.owner_id, 
			COALESCE(ARRAY_AGG(u.id) FILTER (WHERE u.id IS NOT NULL), '{}') AS user_ids, 
			COALESCE(ARRAY_AGG(u.full_name) FILTER (WHERE u.full_name IS NOT NULL), '{}') AS user_names
		FROM 
			projects p
		LEFT JOIN 
			project_assignments pa ON p.id = pa.project_id
		LEFT JOIN 
			users u ON pa.user_id = u.id
		WHERE 
			pa.user_id = $1
    `

	// Modify query if search parameter is provided
	if searchQuery != "" {
		query += `AND p.name ILIKE '%' || $4 || '%' `
	}

	query += `GROUP BY 
                p.id, p.name, p.description, p.start_date, p.due_date, p.priority, 
                p.status, p.created_at, p.updated_at, p.owner_id 
              ORDER BY p.updated_at DESC 
              LIMIT $2 OFFSET $3;`

	// Execute the query
	var rows pgx.Rows
	if searchQuery != "" {
		rows, err = utils.DBPool.Query(context.Background(), query, userID, limit, offset, searchQuery)
	} else {
		rows, err = utils.DBPool.Query(context.Background(), query, userID, limit, offset)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve projects"})
		return
	}
	defer rows.Close()

	projects := []utils.Project{}

	for rows.Next() {
		var project utils.Project
		var startDate time.Time
		var dueDate time.Time
		var createdAt time.Time
		var updatedAt time.Time
		project.UserIds = []string{}
		project.UserNames = []string{}
		if err := rows.Scan(&project.ID, &project.Name, &project.Description, &startDate, &dueDate, &project.Priority, &project.Status, &createdAt, &updatedAt, &project.OwnerID, pq.Array(&project.UserIds), pq.Array(&project.UserNames)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan project"})
			return
		}
		project.StartDate = startDate.Format("02-01-2006 3:04:05 PM")
		project.DueDate = dueDate.Format("02-01-2006 3:04:05 PM")
		project.CreatedAt = createdAt.Format("02-01-2006 3:04:05 PM")
		project.UpdatedAt = updatedAt.Format("02-01-2006 3:04:05 PM")
		projects = append(projects, project)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred during row iteration"})
		return
	}

	// Count total number of projects, including the search filter if applied
	countQuery := `SELECT COUNT(*)
					FROM projects p
					JOIN project_assignments pa ON p.id = pa.project_id
					WHERE pa.user_id = $1
				`
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
		"projects":   projects,
		"page":       page,
		"limit":      limit,
		"total":      total,
		"totalPages": totalPages,
		"hasNext":    hasNext,
		"hasPrev":    hasPrev,
	})
}
