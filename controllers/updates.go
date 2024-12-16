// updates.go
package controllers

import (
	"context"
	"fmt"

	"github.com/go/task_management/backend/utils"
)

func LogDetailedUpdate(
	updatedBy, description, projectID, taskID, role, changeType, beforeValue, afterValue string,
) error {
	query := `
        INSERT INTO recent_updates (updated_by, update_description, update_time, project_id, task_id, role, change_type, before_value, after_value)
        VALUES ($1, $2, NOW(), $3, $4, $5, $6, $7, $8);
    `
	_, err := utils.DBPool.Exec(context.Background(), query, updatedBy, description, projectID, taskID, role, changeType, beforeValue, afterValue)
	return err
}

func GetMemberRecentUpdates(memberID string) ([]utils.Updates, error) {
	query := `
	SELECT 
		ru.id, 
		ru.updated_by, 
		COALESCE(a.username, m.full_name) AS updated_by_name, 
		ru.update_description, 
		ru.update_time, 
		ru.project_id, 
		p.name AS project_name, 
		ru.task_id, 
		t.title AS task_name, 
		ru.role, 
		ru.change_type, 
		ru.before_value, 
		ru.after_value
	FROM recent_updates ru
	LEFT JOIN admins a ON ru.updated_by = a.id
	LEFT JOIN users m ON ru.updated_by = m.id
	LEFT JOIN project_assignments pa ON ru.project_id = pa.project_id
	LEFT JOIN projects p ON ru.project_id = p.id
	LEFT JOIN tasks t ON ru.task_id = t.id
	WHERE pa.user_id = $1
	   OR (ru.task_id IS NOT NULL AND t.assigned_user_id = $1)
	   GROUP BY ru.id, ru.updated_by, a.username, m.full_name, ru.update_description, 
         ru.update_time, ru.project_id, p.name, ru.task_id, t.title, 
         ru.role, ru.change_type, ru.before_value, ru.after_value
	ORDER BY ru.update_time DESC
	LIMIT 5;
`

	var updates []utils.Updates
	rows, err := utils.DBPool.Query(context.Background(), query, memberID)
	if err != nil {
		return nil, fmt.Errorf("querying recent updates failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var update utils.Updates
		err := rows.Scan(
			&update.ID,
			&update.UpdatedBy,
			&update.UpdatedByName,
			&update.Description,
			&update.UpdateTime,
			&update.ProjectID,
			&update.ProjectName,
			&update.TaskID,
			&update.TaskName,
			&update.Role,
			&update.ChangeType,
			&update.BeforeValue,
			&update.AfterValue,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning row failed: %w", err)
		}
		updates = append(updates, update)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("rows iteration error: %w", rows.Err())
	}

	return updates, nil
}
