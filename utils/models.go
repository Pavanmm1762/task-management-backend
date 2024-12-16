package utils

import (
	"time"

	"github.com/gocql/gocql"
)

// LoginUser represents the input for the login request
type LoginUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterUser represents the input for registering a new user
type RegisterUser struct {
	AdminID   string    `json:"admin_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated-at"`
}

// Project represents a project with details
type Project struct {
	ID          string   `json:"id" db:"id"`
	Name        string   `json:"name" db:"name"`
	Description string   `json:"description" db:"description"`
	StartDate   string   `json:"start_date" db:"start_date"`
	DueDate     string   `json:"due_date" db:"due_date"`
	CreatedAt   string   `json:"created_at" db:"created_at"`
	UpdatedAt   string   `json:"updated_at" db:"updated_at"`
	Priority    string   `json:"priority" db:"priority"`
	Status      string   `json:"status" db:"status"`
	UserIds     []string `json:"user_ids"`
	UserNames   []string `json:"user_names"`
	OwnerID     string   `json:"owner_id" db:"owner_id"`
}

// Task represents a task within a project
type Task struct {
	ID               string  `json:"id" db:"id"`
	ProjectName      string  `json:"project_name"`
	Title            string  `json:"title" db:"title"`
	Description      string  `json:"description" db:"description"`
	Priority         string  `json:"priority" db:"priority"` // e.g.,"low", "medium", "high"
	Status           string  `json:"status" db:"status"`     // e.g.,"in progress", "pending", "completed"
	StartDate        string  `json:"start_date" db:"start_date"`
	DueDate          string  `json:"due_date" db:"due_date"`
	CreatedAt        string  `json:"created_at" db:"created_at"`
	UpdatedAt        string  `json:"updated_at" db:"updated_at"`
	ProjectID        string  `json:"project_id" db:"project_id"` // Reference to the project
	AssignedUserID   *string `json:"assigned_user_id" db:"assigned_user_id"`
	AssignedUserName *string `json:"assigned_user_name"`
	Designation      *string `json:"user_designation"`
	Department       *string `json:"user_department"`
}

// User represents the structure of a user in the database
type User struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	FullName    string  `json:"full_name"`
	ContactNo   *string `json:"contact_no"`
	Gender      *string `json:"gender"`
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	Designation string  `json:"designation"`
	Department  string  `json:"department"`
	Role        string  `json:"role"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	AdminID     string  `json:"admin_id"`
}

// ProjectAssignment struct
type ProjectAssignment struct {
	ProjectID  string    `json:"project_id"` // Reference to the project
	UserID     string    `json:"user_id"`    // Reference to the user
	AssignedAt time.Time `json:"assigned_at"`
	Role       string    `json:"role"` // e.g., "member", "lead"
}

// Report represents project statistics
type ProjectReport struct {
	ProjectID       string    `json:"project_id"`
	ProjectName     string    `json:"project_name"`
	Owner           string    `json:"owner"`
	TeamMembers     []string  `json:"team_members"`
	StartDate       time.Time `json:"start_date"`
	EndDate         time.Time `json:"end_date"`
	Priority        string    `json:"priority"`
	Status          string    `json:"status"`
	TotalTasks      int       `json:"total_tasks"`
	CompletedTasks  int       `json:"completed_tasks"`
	InProgressTasks int       `json:"in_progress_tasks"`
	OverdueTasks    int       `json:"overdue_tasks"`
	ProgressPercent int       `json:"progress_percentage"`
}

// Dashboard represents statistics for the dashboard
type Statistics struct {
	TotalProjects     int `json:"total_projects"`
	CompletedProjects int `json:"completed_projects"`
	TotalUsers        int `json:"total_users"`
}

// Report represents overall statistics
type OverallReport struct {
	TotalProjects      int `json:"total_projects"`
	TotalTasks         int `json:"total_tasks"`
	CompletedTasks     int `json:"completed_tasks"`
	InProgressTasks    int `json:"in_progress_tasks"`
	OverdueTasks       int `json:"overdue_tasks"`
	TotalUsers         int `json:"total_users"`
	CompletedProjects  int `json:"completed_projects"`
	InProgressProjects int `json:"in_progress_projects"`
	NotStartedProjects int `json:"not_started_projects"`
	ActiveUsers        int `json:"active_users"`
	InactiveUsers      int `json:"inactive_users"`
}

// Report represents project summaries
type ProjectSummary struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	CompletedTasks  int    `json:"completed_tasks"`
	TotalTasks      int    `json:"total_tasks"`
	InProgressTasks int    `json:"in_progress_tasks"`
	OverdueTasks    int    `json:"overdue_tasks"`
	TotalUsers      int    `json:"total_users"`
}

type UsersReport struct {
	ID                 string `json:"id"`
	Username           string `json:"username"`
	FullName           string `json:"full_name"`
	Designation        string `json:"designation"`
	Department         string `json:"department"`
	Role               string `json:"role"`
	Status             string `json:"status"`
	CompletedTasks     int    `json:"completed_tasks"`
	InProgressTasks    int    `json:"in_progress_tasks"`
	OverdueTasks       int    `json:"overdue_tasks"`
	TaskCompletionData []Task `json:"task_completion_data"`
}

// Updates represents a recent updates
type Updates struct {
	ID            int       `db:"id"`
	UpdatedBy     string    `json:"updated_by"`
	UpdatedByName string    `json:"updated_by_name"`
	Description   string    `json:"description"`
	UpdateTime    time.Time `json:"update_time"`
	ProjectID     *string   `json:"project_id"`
	ProjectName   *string   `json:"project_name"`
	TaskID        *string   `json:"task_id"`
	TaskName      *string   `json:"task_name"`
	Role          string    `json:"role"`
	ChangeType    string    `json:"change_type"`
	BeforeValue   *string   `json:"before_value"`
	AfterValue    *string   `json:"after_value"`
}

// Comment represents a comment on a task
type Comment struct {
	ID     gocql.UUID `json:"id"`
	Text   string     `json:"text"`
	UserID string     `json:"user_id"` // Format: u-0001
	TaskID string     `json:"task_id"` // Format: t-000001
}

// Message represents a chat message
type Message struct {
	Sender      string `json:"sender"`
	Text        string `json:"text"`
	RecipientID string `json:"recipient_id"` // Format: u-0001
}

// Notification represents a notification for a user
type Notification struct {
	ID        int       `json:"id"`
	UserID    string    `json:"user_id"`
	Message   string    `json:"message"`
	Type      string    `json:"type"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

// package utils

// import (
// 	"github.com/gocql/gocql"
// )

// type LoginUser struct {
// 	Username string `json:"username"`
// 	Password string `json:"password"`
// }

// type RegisterUser struct {
// 	Admin_id gocql.UUID `json:"admin_id"`
// 	Username string     `json:"username"`
// 	Email    string     `json:"email"`
// 	Password string     `json:"password"`
// }

// type Project struct {
// 	ID          gocql.UUID `json:"id"`
// 	Name        string     `json:"name"`
// 	Description string     `json:"description"`
// 	StartDate   string     `json:"start_date"`
// 	DueDate     string     `json:"due_date"`
// 	Status      string     `json:"status"`
// 	OwnerID     gocql.UUID `json:"owner_id"`
// }

// type Task struct {
// 	ID          gocql.UUID `json:"task_id"`
// 	Title       string     `json:"task_name"`
// 	ProjectName string     `json:"project_name"`
// 	Description string     `json:"description"`
// 	Progress    int        `json:"progress"`
// 	Status      string     `json:"status"`
// 	ProjectID   gocql.UUID `json:"project_id"`
// 	// Add more fields as needed
// }

// type Users struct {
// 	UserId       gocql.UUID `json:"user_id"`
// 	FirstName    string     `json:"firstname"`
// 	LastName     string     `json:"lastname"`
// 	UserRole     string     `json:"role"`
// 	UserEmail    string     `json:"email"`
// 	UserPassword string     `json:"password"`
// }

// type Reports struct {
// 	ProjectName    string  `json:"project_name"`
// 	TotalTasks     int     `json:"total_tasks"`
// 	CompletedTasks int     `json:"completed_tasks"`
// 	Progress       float64 `json:"progress"`
// 	Status         string  `json:"status"`
// 	Usertask       string  `json:"user_task"`
// }

// type Dashboard struct {
// 	TotalProjects     int `json:"total_projects"`
// 	CompletedProjects int `json:"completed_projects"`
// 	TotalUsers        int `json:"total_users"`
// }

// type Comment struct {
// 	ID     gocql.UUID `json:"id"`
// 	Text   string     `json:"text"`
// 	UserID gocql.UUID `json:"user_id"`
// 	TaskID gocql.UUID `json:"task_id"`
// 	// Add more fields as needed
// }

// type Message struct {
// 	Sender      string `json:"sender"`
// 	Text        string `json:"text"`
// 	RecipientId string `json:"recipient_id"`
// 	// Add more fields as needed
// }

// type Notification struct {
// 	ID      gocql.UUID `json:"id"`
// 	Message string     `json:"message"`
// 	UserID  gocql.UUID `json:"userId"`
// 	// Add more fields as needed
// }
