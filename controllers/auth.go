package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/go/task_management/backend/utils"
	"golang.org/x/crypto/bcrypt"

	"github.com/gin-gonic/gin"
)

// RegisterAdmin registers a new admin
func RegisterAdmin(c *gin.Context) {
	var admin utils.RegisterUser
	if err := c.ShouldBindJSON(&admin); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Check if the username already exists
	var existingAdminID string
	queryCheck := `SELECT id FROM admins WHERE username=$1`
	err := utils.DBPool.QueryRow(context.Background(), queryCheck, admin.Username).Scan(&existingAdminID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}

	// Validate the password (add your own validation logic if needed)
	if len(admin.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 8 characters"})
		return
	}

	// Hash the password before storing it
	hashedPassword, err := HashPassword(admin.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Generate a new ID for the admin
	newID, err := utils.GetNextAdminID() // Implement this function to generate IDs
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate new ID"})
		return
	}
	admin.AdminID = newID
	admin.Password = hashedPassword
	admin.CreatedAt = time.Now().UTC() // Store in UTC for consistency
	admin.UpdatedAt = admin.CreatedAt

	// Insert the admin into the database
	query := `INSERT INTO admins (id, username, password, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`
	_, err = utils.DBPool.Exec(context.Background(), query, admin.AdminID, admin.Username, admin.Password, admin.CreatedAt, admin.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create admin"})
		return
	}

	// Initialize the admin counters for the projects and users
	query = `INSERT INTO id_counter (admin_id, project_value, user_value, task_value) VALUES ($1, 0, 0, 0)`
	_, err = utils.DBPool.Exec(context.Background(), query, admin.AdminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create counter"})
		return
	}

	// Return a success response
	c.JSON(http.StatusCreated, gin.H{
		"id":        admin.AdminID,
		"username":  admin.Username,
		"createdAt": admin.CreatedAt,
	})
}

// LoginAdmin authenticates an admin and returns a token if successful.
func LoginAdmin(c *gin.Context) {
	var admin utils.RegisterUser
	if err := c.ShouldBindJSON(&admin); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Query the database for the existing admin
	var storedAdmin utils.RegisterUser
	query := `SELECT id, password FROM admins WHERE username=$1`
	err := utils.DBPool.QueryRow(context.Background(), query, admin.Username).Scan(&storedAdmin.AdminID, &storedAdmin.Password)

	if err != nil {
		// Admin not found
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Username not registered!"})
		return
	}

	// Compare the hashed password with the stored hash
	if !CheckPasswordHash(admin.Password, storedAdmin.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect password!"})
		return
	}

	// Generate a token (implement your token generation logic)
	token, err := utils.GenerateToken(storedAdmin.AdminID, "admin") // Replace with your token generation method
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token, "adminID": storedAdmin.AdminID})
}

// LoginMember authenticates a member and returns a token if successful.
func LoginMember(c *gin.Context) {
	var member utils.RegisterUser
	if err := c.ShouldBindJSON(&member); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Query the database for the existing member
	var storedMember utils.RegisterUser
	query := `SELECT id, password FROM users WHERE username=$1`
	err := utils.DBPool.QueryRow(context.Background(), query, member.Username).Scan(&storedMember.AdminID, &storedMember.Password)

	if err != nil {
		// Member not found
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Username not registered!"})
		return
	}

	// Compare the hashed password with the stored hash
	if !CheckPasswordHash(member.Password, storedMember.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect password!"})
		return
	}

	// Generate a token for the member (implement your token generation logic)
	token, err := utils.GenerateToken(storedMember.AdminID, "member") // Replace with your token generation method
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token, "memberID": storedMember.AdminID})
}

// HashPassword hashes the password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// CheckPasswordHash checks the password against the hashed password
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
