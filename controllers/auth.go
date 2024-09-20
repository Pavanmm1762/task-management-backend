package controllers

import (
	"net/http"

	"github.com/go/task_management/backend/utils"
	"github.com/gocql/gocql"

	"github.com/gin-gonic/gin"
)

func Register(c *gin.Context) {
	var user utils.RegisterUser
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if isUserRegistered(user.Username) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User with this username already exists"})
		return
	}

	// user registration logic and storing user data in Cassandra
	user.Admin_id = gocql.TimeUUID()

	query := "INSERT INTO admin (admin_id, username, email, password) VALUES (?, ?, ?, ?)"
	err := utils.Session.Query(query, user.Admin_id, user.Username, user.Email, user.Password).Exec()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
}

func Login(c *gin.Context) {
	var user utils.LoginUser
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if the user exists
	userID, err := authenticateUser(user.Username, user.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// Generate JWT token
	token, err := utils.GenerateToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token, "userId": userID})
}

func isUserRegistered(username string) bool {
	// Perform a database query to check if a user with the given username already exists
	// Example using gocql
	var existingUser utils.RegisterUser
	if err := utils.Session.Query("SELECT admin_id FROM admin WHERE username=? ALLOW FILTERING", username).Scan(&existingUser.Admin_id); err != nil {
		// User not found, return false
		return false
	}

	// User found, return true
	return true
}

func authenticateUser(username, password string) (gocql.UUID, error) {
	// Perform a database query to authenticate the user
	// Example using gocql
	var userID gocql.UUID
	if err := utils.Session.Query("SELECT admin_id FROM admin WHERE username=? AND password=? ALLOW FILTERING", username, password).Scan(&userID); err != nil {
		// Authentication failed
		return gocql.UUID{}, err
	}

	// Authentication successful
	return userID, nil
}
