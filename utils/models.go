package utils

import (
	"github.com/gocql/gocql"
)

type LoginUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegisterUser struct {
	Admin_id gocql.UUID `json:"admin_id"`
	Username string     `json:"username"`
	Email    string     `json:"email"`
	Password string     `json:"password"`
}

type Project struct {
	ID          gocql.UUID `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	StartDate   string     `json:"start_date"`
	DueDate     string     `json:"due_date"`
	Status      string     `json:"status"`
	OwnerID     gocql.UUID `json:"owner_id"`
}

type Task struct {
	ID          gocql.UUID `json:"task_id"`
	Title       string     `json:"task_name"`
	ProjectName string     `json:"project_name"`
	Description string     `json:"description"`
	Progress    int        `json:"progress"`
	Status      string     `json:"status"`
	ProjectID   gocql.UUID `json:"project_id"`
	// Add more fields as needed
}

type Users struct {
	UserId       gocql.UUID `json:"user_id"`
	FirstName    string     `json:"firstname"`
	LastName     string     `json:"lastname"`
	UserRole     string     `json:"role"`
	UserEmail    string     `json:"email"`
	UserPassword string     `json:"password"`
}

type Reports struct {
	ProjectName    string  `json:"project_name"`
	TotalTasks     int     `json:"total_tasks"`
	CompletedTasks int     `json:"completed_tasks"`
	Progress       float64 `json:"progress"`
	Status         string  `json:"status"`
	Usertask       string  `json:"user_task"`
}

type Dashboard struct {
	TotalProjects     int `json:"total_projects"`
	CompletedProjects int `json:"completed_projects"`
	TotalUsers        int `json:"total_users"`
}

type Comment struct {
	ID     gocql.UUID `json:"id"`
	Text   string     `json:"text"`
	UserID gocql.UUID `json:"user_id"`
	TaskID gocql.UUID `json:"task_id"`
	// Add more fields as needed
}

type Message struct {
	Sender      string `json:"sender"`
	Text        string `json:"text"`
	RecipientId string `json:"recipient_id"`
	// Add more fields as needed
}

type Notification struct {
	ID      gocql.UUID `json:"id"`
	Message string     `json:"message"`
	UserID  gocql.UUID `json:"userId"`
	// Add more fields as needed
}
