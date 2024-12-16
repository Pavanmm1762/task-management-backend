package utils

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
)

// get the next id from the counter table

func GetNextAdminID() (string, error) {
	var lastAdminNum int
	row := DBPool.QueryRow(context.Background(), "SELECT COUNT(*) FROM admins ")
	if err := row.Scan(&lastAdminNum); err != nil {
		return "", err
	}

	// Increment to get the next project number
	nextAdminNum := lastAdminNum + 1

	// Format the new project ID
	newID := fmt.Sprintf("A%03d", nextAdminNum)

	return newID, nil
}

// Function to get the next User ID for a given admin
func GetNextUserID(tx pgx.Tx, adminId string) (string, error) {
	var nextUserNum int

	// Lock the admin's row in the id_counter table for update
	err := tx.QueryRow(context.Background(), "SELECT user_value FROM id_counter WHERE admin_id = $1 FOR UPDATE", adminId).Scan(&nextUserNum)
	if err != nil {
		return "", err
	}

	// Increment user counter
	nextUserNum++

	// Update the user counter for the admin in id_counter
	_, err = tx.Exec(context.Background(), "UPDATE id_counter SET user_value = $1 WHERE admin_id = $2", nextUserNum, adminId)
	if err != nil {
		return "", err
	}

	// Generate the new User ID in the format U-adminId-XXXX
	newUserID := fmt.Sprintf("U-%s-%04d", adminId, nextUserNum)

	return newUserID, nil
}

// Function to get the next Project ID for a given admin
func GetNextProjectID(tx pgx.Tx, adminId string) (string, error) {
	var nextProjectNum int

	// Lock the admin's row in the id_counter table for update
	err := tx.QueryRow(context.Background(), "SELECT project_value FROM id_counter WHERE admin_id = $1 FOR UPDATE", adminId).Scan(&nextProjectNum)
	if err != nil {
		return "", err
	}

	// Increment project counter
	nextProjectNum++

	// Update the project counter for the admin in id_counter
	_, err = tx.Exec(context.Background(), "UPDATE id_counter SET project_value = $1 WHERE admin_id = $2", nextProjectNum, adminId)
	if err != nil {
		return "", err
	}

	// Generate the new Project ID in the format P-adminId-XXXX
	newProjectID := fmt.Sprintf("P-%s-%04d", adminId, nextProjectNum)

	return newProjectID, nil
}

func GetNextTaskID(tx pgx.Tx, adminId string) (string, error) {
	var nextTaskNum int

	// Lock the task's row in the task_counter table for update
	err := tx.QueryRow(context.Background(), "SELECT task_value FROM id_counter WHERE admin_id = $1 FOR UPDATE", adminId).Scan(&nextTaskNum)
	if err != nil {
		return "", err
	}

	// Increment task counter
	nextTaskNum++

	// Update the task counter for the admin in task_counter
	_, err = tx.Exec(context.Background(), "UPDATE id_counter SET task_value = $1 WHERE admin_id = $2", nextTaskNum, adminId)
	if err != nil {
		return "", err
	}

	// Generate the new task ID in the format T-projectId-XXXX
	newTaskID := fmt.Sprintf("T-%s-%06d", adminId, nextTaskNum)

	return newTaskID, nil
}
