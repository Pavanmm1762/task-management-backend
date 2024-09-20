// Dashboard.go
package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/go/task_management/backend/utils"
	"github.com/gocql/gocql"
)

// InitTaskRoutes initializes the task-related routes
func InitDashboardRoutes(router *gin.RouterGroup) {
	router.GET("/dashboard", getDashboard)
}

// getTasks fetches report details from the database and returns them as a JSON response
func getDashboard(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")
	var dashboard utils.Dashboard
	userID, err := getUserId(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	iter := utils.Session.Query("SELECT count(id) FROM projects where owner_id = ? ",
		userID).Iter()

	var totalProjects, completedProjects int

	if !iter.Scan(&totalProjects) {
		if err := iter.Close(); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

	}
	dashboard.TotalProjects = totalProjects
	fmt.Printf("Total Projects: %v\n", totalProjects)
	iter.Close()

	iter1 := utils.Session.Query("SELECT count(status) FROM projects where owner_id = ? and status = 'Completed' ALLOW FILTERING ",
		userID).Iter()

	if !iter1.Scan(&completedProjects) {
		if err := iter.Close(); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

	}
	dashboard.CompletedProjects = completedProjects
	fmt.Printf("Completed Projects: %v\n", completedProjects)
	iter1.Close()

	// Fetch tasks for each project
	userIter := utils.Session.Query(`
			SELECT count(user_id)
			FROM users
			WHERE admin_id = ?
			`, userID).Iter()
	var totalUsers int
	if !userIter.Scan(&totalUsers) {
		if err := userIter.Close(); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
	}
	dashboard.TotalUsers = totalUsers
	fmt.Printf("Total users: %v\n", totalUsers)
	userIter.Close()
	// Print the response before sending it
	fmt.Println("Response:", dashboard)
	c.JSON(200, dashboard)
}

var jwtSecrt = []byte("secure_secret_key")

func getUserId(tokenString string) (gocql.UUID, error) {

	tokenString = strings.Replace(tokenString, "Bearer ", "", 1)

	token, err := jwt.ParseWithClaims(tokenString, &utils.Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecrt, nil
	})

	// Check for errors
	if err != nil {
		return gocql.UUID{}, err
	}

	// Check if the token is valid
	if claims, ok := token.Claims.(*utils.Claims); ok && token.Valid {
		return claims.UserID, nil
	}

	return gocql.UUID{}, errors.New("invalid token")
}
