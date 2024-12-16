// // controllers/users.go
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
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

// InitTaskRoutes initializes routes for tasks
func InitUserRoutes(router *gin.RouterGroup) {
	router.GET("/users", GetUsers)
	router.GET("/users/:project_id", AvailableUsers)
	router.POST("users/create-user", CreateUser)
	router.DELETE("/users/:userid", DeleteUser)
	router.PUT("/users/:userid", UpdateUser)
}

// CreateUser handles the creation of a user
func CreateUser(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")
	var user utils.User

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Start a transaction
	tx, err := utils.DBPool.Begin(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin transaction"})
		return
	}
	defer tx.Rollback(context.Background())

	adminId, err := utils.GetUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	user.AdminID = adminId

	// Get next user ID
	newID, err := utils.GetNextUserID(tx, adminId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate new ID"})
		return
	}
	user.ID = newID

	hashedPassword, err := HashPassword(user.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	user.Password = hashedPassword
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user.Status = "active"
	user.Role = "member"
	user.CreatedAt = time.Now().Format("02-01-2006 3:04:05 PM")
	user.UpdatedAt = user.CreatedAt

	query := `INSERT INTO users (id, username, full_name, contact_no, gender, email, password, designation, department, status, role, admin_id) 
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err = utils.DBPool.Exec(ctx, query,
		user.ID, user.Username, user.FullName, user.ContactNo, user.Gender, user.Email, user.Password, user.Designation, user.Department, user.Status, user.Role, user.AdminID)

	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == "23505" {
				if strings.Contains(pgErr.Message, "users_username_key") {
					c.JSON(http.StatusConflict, gin.H{"error": "Username is already taken"})
				} else if strings.Contains(pgErr.Message, "users_email_key") {
					c.JSON(http.StatusConflict, gin.H{"error": "Email is already taken"})
				} else {
					c.JSON(http.StatusConflict, gin.H{"error": "Username or email is already taken"})
				}
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		log.Print(err)
		return
	}

	// Commit the transaction
	err = tx.Commit(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// GetUsers get the users lists
func GetUsers(c *gin.Context) {
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

	adminID, err := utils.GetUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: User ID not found"})
		return
	}

	offset := (page - 1) * limit

	// Base query for projects with pagination
	query := `SELECT id, username, full_name, contact_no, gender, email, designation, department, status, created_at, admin_id 
	          FROM users
			  WHERE admin_id = $1 `

	// Modify query if search parameter is provided
	if searchQuery != "" {
		query += `AND username ILIKE '%' || $4 || '%' `
	}

	query += `ORDER BY created_at DESC 
	          LIMIT $2 OFFSET $3`

	// Execute the query
	var rows pgx.Rows
	if searchQuery != "" {
		rows, err = utils.DBPool.Query(context.Background(), query, adminID, limit, offset, searchQuery)
	} else {
		rows, err = utils.DBPool.Query(context.Background(), query, adminID, limit, offset)
	}

	if err != nil {
		fmt.Print(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
		return
	}
	defer rows.Close()

	users := []utils.User{}

	for rows.Next() {
		var user utils.User
		var created_at time.Time
		if err := rows.Scan(&user.ID, &user.Username, &user.FullName, &user.ContactNo, &user.Gender, &user.Email, &user.Designation, &user.Department, &user.Status, &created_at, &user.AdminID); err != nil {
			log.Print(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan user"})
			return
		}
		user.CreatedAt = created_at.Format("02-01-2006 3:04:05 PM")
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred during row iteration"})
		return
	}

	// Count total number of projects, including the search filter if applied
	countQuery := `SELECT COUNT(*) FROM users WHERE admin_id = $1 `
	if searchQuery != "" {
		countQuery += `AND username ILIKE '%' || $2 || '%'`
	}

	var total int
	if searchQuery != "" {
		err = utils.DBPool.QueryRow(context.Background(), countQuery, adminID, searchQuery).Scan(&total)
	} else {
		err = utils.DBPool.QueryRow(context.Background(), countQuery, adminID).Scan(&total)
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
		"users":      users,
		"page":       page,
		"limit":      limit,
		"total":      total,
		"totalPages": totalPages,
		"hasNext":    hasNext,
		"hasPrev":    hasPrev,
	})
}

// Update User
func UpdateUser(c *gin.Context) {
	userID := c.Param("userid")

	var updateUser utils.User
	if err := c.ShouldBindJSON(&updateUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `UPDATE users SET full_name=$1, email=$2, designation=$3, department=$4, contact_no=$5, gender=$6, updated_at=NOW() WHERE id=$7`

	_, err := utils.DBPool.Exec(context.Background(), query, updateUser.FullName, updateUser.Email, updateUser.Designation, updateUser.Department, updateUser.ContactNo, updateUser.Gender, userID)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == "23505" {
				if strings.Contains(pgErr.Message, "users_username_key") {
					c.JSON(http.StatusConflict, gin.H{"error": "Username is already taken"})
				} else if strings.Contains(pgErr.Message, "users_email_key") {
					c.JSON(http.StatusConflict, gin.H{"error": "Email is already taken"})
				} else {
					c.JSON(http.StatusConflict, gin.H{"error": "Username or email is already taken"})
				}
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

// Delete user
func DeleteUser(c *gin.Context) {
	userID := c.Param("userid")

	query := "DELETE FROM users WHERE id = $1"

	_, err := utils.DBPool.Exec(context.Background(), query, userID)
	if err != nil {
		log.Print(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}
