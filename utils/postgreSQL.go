package utils

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

// DBPool is the connection pool for PostgreSQL
var DBPool *pgxpool.Pool

// Connect initializes the PostgreSQL connection pool
func ConnectToDB() {

	// dbHost := os.Getenv("DB_HOST")
	// dbPort := os.Getenv("DB_PORT")
	// dbUser := os.Getenv("DB_USER")
	// dbPassword := os.Getenv("DB_PASSWORD")
	// dbName := os.Getenv("DB_NAME")

	// // Create the connection string
	// connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
	// 	dbUser, dbPassword, dbHost, dbPort, dbName)
	connString := os.Getenv("DATABASE_URL")

	// Setup connection pool configuration
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		fmt.Errorf("unable to parse PostgreSQL config: %v", err)
		return
	}

	// Set pool configuration options
	config.MaxConns = 10                 // Maximum number of connections
	config.MaxConnIdleTime = time.Minute // Max idle time for a connection

	// Attempt to connect to the database
	for retries := 3; retries > 0; retries-- {
		DBPool, err = pgxpool.ConnectConfig(context.Background(), config)
		if err == nil {
			// Ping to verify the connection is alive
			if err = Ping(); err == nil {
				fmt.Printf("Connected to the database successfully")
				return
			}
		}
		log.Printf("Failed to connect to the database. Retrying in 2 seconds... (%d retries left)\n", retries-1)
		time.Sleep(2 * time.Second)
	}
	log.Fatalf("Unable to connect to database after multiple attempts: %v", err)

}

// Close closes the database connection pool
func Close() {
	fmt.Printf("Database connection pool closed successfully")
	DBPool.Close()
}

// Ping checks the connection to the database
func Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return DBPool.Ping(ctx)
}

// CreateTable creates a new table in the database
func CreateTables() {
	// Define your SQL command to create the table
	queries := []string{
		`CREATE TABLE IF NOT EXISTS admins (
			id VARCHAR(20) PRIMARY KEY,
			username VARCHAR(50) UNIQUE NOT NULL,
			password VARCHAR(100) NOT NULL,  
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(20) PRIMARY KEY,
			username VARCHAR(50) NOT NULL UNIQUE,
			full_name VARCHAR(20) NOT NULL,
			contact_no VARCHAR(20),
			email VARCHAR(100) NOT NULL UNIQUE,
			password VARCHAR(255) NOT NULL,
			designation VARCHAR(50) NOT NULL,
			department VARCHAR(50) NOT NULL,
			gender VARCHAR(50) CHECK (gender IN ('male', 'female', 'NA')),
			role VARCHAR(20) NOT NULL CHECK (role IN ('admin', 'member')),
			status VARCHAR(20) NOT NULL CHECK (status IN ('active', 'inactive')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			admin_id VARCHAR(20) REFERENCES admins(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS projects (
			id VARCHAR(20) PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			description TEXT,
			start_date TIMESTAMP,
			due_date TIMESTAMP,
			priority VARCHAR(20),
			status VARCHAR(20),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			owner_id VARCHAR(20) REFERENCES admins(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS project_assignments (
			project_id VARCHAR(20) REFERENCES projects(id) ON DELETE CASCADE,
			user_id VARCHAR(20) REFERENCES users(id) ON DELETE CASCADE,
			assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			role VARCHAR(20) NOT NULL CHECK (role IN ('member', 'lead')),
			PRIMARY KEY (project_id, user_id)
		);`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id VARCHAR(20) PRIMARY KEY,
			title VARCHAR(100) NOT NULL,
			description TEXT,
			priority VARCHAR(20) NOT NULL CHECK (priority IN ('low', 'medium', 'high')),
			start_date TIMESTAMP,
			due_date TIMESTAMP,
			status VARCHAR(20) NOT NULL CHECK (status IN ( 'pending', 'in progress', 'overdue', 'review', 'completed')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			project_id VARCHAR(20) REFERENCES projects(id) ON DELETE CASCADE,
			assigned_user_id VARCHAR(20) REFERENCES users(id) ON DELETE SET NULL
		);`,
		`CREATE TABLE IF NOT EXISTS id_counter (
            admin_id TEXT PRIMARY KEY REFERENCES admins(id) ON DELETE CASCADE,
            project_value INT NOT NULL,
			user_value INT NOT NULL,
			task_value INT NOT NULL  
        );`,
		`CREATE TABLE IF NOT EXISTS task_counter (
            project_id VARCHAR(20) PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
			task_value INT NOT NULL  
        );`,
		`CREATE TABLE IF NOT EXISTS recent_updates (
			id SERIAL PRIMARY KEY,
			updated_by VARCHAR(255) NOT NULL,        
			update_description TEXT NOT NULL,        
			update_time TIMESTAMP DEFAULT NOW(),     
			project_id VARCHAR(255),               
			task_id VARCHAR(255),                   
			role VARCHAR(50) NOT NULL,               
			change_type VARCHAR(50) NOT NULL,        
			before_value TEXT, 
			after_value TEXT                        
		);`,
		`CREATE TABLE IF NOT EXISTS notifications (
			id SERIAL PRIMARY KEY,
			user_id VARCHAR(20) NOT NULL,
			message TEXT NOT NULL,
			type VARCHAR(50) NOT NULL, 
			is_read BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() // Ensure the context is canceled to avoid memory leaks

	// Execute each query
	for _, query := range queries {
		_, err := DBPool.Exec(ctx, query)
		if err != nil {
			log.Print("error: Failed to create tables ", "details %v", err.Error())
			return
		}
	}

	fmt.Printf("message : Tables created successfully")

}
