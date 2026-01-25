//go:build integration

// Integration tests for database backends
// Run with: go test -tags=integration ./tests/integration/...
//
// Prerequisites:
// - docker compose -f docker-compose.test.yaml up -d postgres mariadb
// - Wait for databases to be healthy

package integration

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func TestPostgreSQL_Connection(t *testing.T) {
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		user = "k13d"
	}
	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		password = "testpassword"
	}
	dbname := os.Getenv("POSTGRES_DB")
	if dbname == "" {
		dbname = "k13d_test"
	}

	connStr := "host=" + host + " port=" + port + " user=" + user + " password=" + password + " dbname=" + dbname + " sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Skipping: cannot create connection: %v", err)
	}
	defer db.Close()

	// Set connection pool settings
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Minute * 5)

	// Test connection
	err = db.Ping()
	if err != nil {
		t.Skipf("Skipping: PostgreSQL not available: %v", err)
	}

	t.Log("PostgreSQL connection successful")

	// Test basic operations
	testDatabaseOperations(t, db, "postgres")
}

func TestMariaDB_Connection(t *testing.T) {
	host := os.Getenv("MARIADB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("MARIADB_PORT")
	if port == "" {
		port = "3306"
	}
	user := os.Getenv("MARIADB_USER")
	if user == "" {
		user = "k13d"
	}
	password := os.Getenv("MARIADB_PASSWORD")
	if password == "" {
		password = "testpassword"
	}
	dbname := os.Getenv("MARIADB_DB")
	if dbname == "" {
		dbname = "k13d_test"
	}

	connStr := user + ":" + password + "@tcp(" + host + ":" + port + ")/" + dbname + "?parseTime=true"

	db, err := sql.Open("mysql", connStr)
	if err != nil {
		t.Skipf("Skipping: cannot create connection: %v", err)
	}
	defer db.Close()

	// Set connection pool settings
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Minute * 5)

	// Test connection
	err = db.Ping()
	if err != nil {
		t.Skipf("Skipping: MariaDB not available: %v", err)
	}

	t.Log("MariaDB connection successful")

	// Test basic operations
	testDatabaseOperations(t, db, "mysql")
}

func testDatabaseOperations(t *testing.T, db *sql.DB, dialect string) {
	// Create test table
	var createTable string
	switch dialect {
	case "postgres":
		createTable = `CREATE TABLE IF NOT EXISTS test_audit (
			id SERIAL PRIMARY KEY,
			user_name VARCHAR(255),
			action VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`
	case "mysql":
		createTable = `CREATE TABLE IF NOT EXISTS test_audit (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_name VARCHAR(255),
			action VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`
	}

	_, err := db.Exec(createTable)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	t.Log("Table created successfully")

	// Insert test data
	_, err = db.Exec("INSERT INTO test_audit (user_name, action) VALUES (?, ?)", "testuser", "test_action")
	if dialect == "postgres" {
		_, err = db.Exec("INSERT INTO test_audit (user_name, action) VALUES ($1, $2)", "testuser", "test_action")
	}
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}
	t.Log("Data inserted successfully")

	// Query test data
	var userName, action string
	row := db.QueryRow("SELECT user_name, action FROM test_audit WHERE user_name = 'testuser' LIMIT 1")
	err = row.Scan(&userName, &action)
	if err != nil {
		t.Fatalf("Failed to query data: %v", err)
	}

	if userName != "testuser" || action != "test_action" {
		t.Errorf("Unexpected data: user_name=%s, action=%s", userName, action)
	}
	t.Log("Data queried successfully")

	// Clean up
	_, err = db.Exec("DROP TABLE IF EXISTS test_audit")
	if err != nil {
		t.Logf("Warning: failed to drop table: %v", err)
	}
	t.Log("Table dropped successfully")
}

func TestRedis_Connection(t *testing.T) {
	// Simple Redis connectivity test
	// In a real implementation, you would use github.com/redis/go-redis
	t.Skip("Redis test requires go-redis client - implement when caching feature is added")
}
