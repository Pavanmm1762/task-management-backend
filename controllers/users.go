// controllers/users.go
package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go/task_management/backend/utils"
	"github.com/gocql/gocql"
)

// InitTaskRoutes initializes routes for tasks
func InitUserRoutes(router *gin.RouterGroup) {
	router.GET("/users-list", getUsers)
	router.POST("/add-user", addUser)
	router.DELETE("/user/:userid", deleteUser)
	router.PUT("/user/:userid", UpdateUser)
}

// CreateProject creates a new project
func addUser(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")
	var user utils.Users
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	adminId, err := getUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	user.UserId = gocql.TimeUUID()

	if err := utils.Session.Query("INSERT INTO users (user_id, firstname, lastname, role, email, password, admin_id) VALUES (?, ?, ?, ?, ?, ?, ?)",
		user.UserId, user.FirstName, user.LastName, user.UserRole, user.UserEmail, user.UserPassword, adminId).Exec(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// GetProjects gets all projects
func getUsers(c *gin.Context) {
	var users []utils.Users
	tokenString1 := c.GetHeader("Authorization")

	admin_id, err := getUserId(tokenString1)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	iter := utils.Session.Query("SELECT user_id, firstname, lastname,  role, email  FROM users where admin_id = ?", admin_id).Iter()
	for {
		var user utils.Users

		if !iter.Scan(&user.UserId, &user.FirstName, &user.LastName, &user.UserRole, &user.UserEmail) {
			break
		}

		users = append(users, user)
	}

	if err := iter.Close(); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

// Update User
func UpdateUser(c *gin.Context) {
	userID, err := gocql.ParseUUID(c.Param("userid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var updatedUser utils.Users // Assuming you have a User struct in utils package

	if err := c.ShouldBindJSON(&updatedUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Perform the update logic, for example, using CQL UPDATE statement
	query := "UPDATE users SET firstname=?, lastname=?, email=?, role=?, password=? WHERE user_id=?"
	err = utils.Session.Query(query, updatedUser.FirstName, updatedUser.LastName, updatedUser.UserEmail, updatedUser.UserRole, updatedUser.UserPassword, userID).Exec()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

// Delete user
func deleteUser(c *gin.Context) {
	userIDStr := c.Param("userid")

	// Parse the user ID
	userID, err := gocql.ParseUUID(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	query := "DELETE FROM users WHERE user_id = ?"
	err = utils.Session.Query(query, userID).Exec()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}
