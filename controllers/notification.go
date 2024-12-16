package controllers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go/task_management/backend/utils"
)

// CreateNotification inserts a new notification into the database
func CreateNotification(notification utils.Notification) error {
	_, err := utils.DBPool.Exec(context.Background(),
		"INSERT INTO notifications (user_id, message, type, is_read, created_at) VALUES ($1, $2, $3, $4, $5)",
		notification.UserID, notification.Message, notification.Type, notification.IsRead, notification.CreatedAt)
	return err
}

// GetNotifications retrieves notifications for a specific user
func GetNotifications(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: user ID not found"})
		return
	}

	rows, err := utils.DBPool.Query(context.Background(),
		"SELECT id, user_id, message, type, is_read, created_at FROM notifications WHERE user_id=$1 ORDER BY created_at DESC",
		userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}
	defer rows.Close()

	var notifications []utils.Notification
	for rows.Next() {
		var n utils.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Message, &n.Type, &n.IsRead, &n.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse notification"})
			return
		}
		notifications = append(notifications, n)
	}

	c.JSON(http.StatusOK, notifications)
}

// MarkAsRead marks a specific notification as read
func MarkAsRead(c *gin.Context) {
	notificationID := c.Param("id")
	if notificationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	_, err := utils.DBPool.Exec(context.Background(),
		"UPDATE notifications SET is_read=true WHERE id=$1",
		notificationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notification as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}
