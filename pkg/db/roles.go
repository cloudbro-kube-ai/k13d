package db

import (
	"fmt"
	"time"
)

// CustomRoleRow represents a row from the custom_roles table
type CustomRoleRow struct {
	Name       string
	Definition string
}

// InitCustomRolesTable creates the custom_roles table
func InitCustomRolesTable() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	var query string
	switch currentDBType {
	case DBTypePostgres:
		query = `
		CREATE TABLE IF NOT EXISTS custom_roles (
			name VARCHAR(255) PRIMARY KEY,
			definition TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
	case DBTypeMariaDB, DBTypeMySQL:
		query = `
		CREATE TABLE IF NOT EXISTS custom_roles (
			name VARCHAR(255) PRIMARY KEY,
			definition TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`
	default: // SQLite
		query = `
		CREATE TABLE IF NOT EXISTS custom_roles (
			name TEXT PRIMARY KEY,
			definition TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`
	}

	_, err := DB.Exec(query)
	return err
}

// SaveCustomRole inserts or updates a custom role definition
func SaveCustomRole(name, definitionJSON string) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	now := time.Now()
	var query string
	switch currentDBType {
	case DBTypePostgres:
		query = `INSERT INTO custom_roles (name, definition, created_at, updated_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (name) DO UPDATE SET definition = $2, updated_at = $4`
	case DBTypeMariaDB, DBTypeMySQL:
		query = `INSERT INTO custom_roles (name, definition, created_at, updated_at)
			VALUES (?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE definition = VALUES(definition), updated_at = VALUES(updated_at)`
	default: // SQLite
		query = `INSERT INTO custom_roles (name, definition, created_at, updated_at)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(name) DO UPDATE SET definition = excluded.definition, updated_at = excluded.updated_at`
	}

	_, err := DB.Exec(query, name, definitionJSON, now, now)
	return err
}

// GetCustomRole returns the JSON definition of a custom role
func GetCustomRole(name string) (string, error) {
	if DB == nil {
		return "", fmt.Errorf("database not initialized")
	}

	var query string
	switch currentDBType {
	case DBTypePostgres:
		query = `SELECT definition FROM custom_roles WHERE name = $1`
	default:
		query = `SELECT definition FROM custom_roles WHERE name = ?`
	}

	var definition string
	err := DB.QueryRow(query, name).Scan(&definition)
	if err != nil {
		return "", err
	}
	return definition, nil
}

// ListCustomRoles returns all custom roles
func ListCustomRoles() ([]CustomRoleRow, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	rows, err := DB.Query("SELECT name, definition FROM custom_roles ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []CustomRoleRow
	for rows.Next() {
		var r CustomRoleRow
		if err := rows.Scan(&r.Name, &r.Definition); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// DeleteCustomRole removes a custom role from the database
func DeleteCustomRole(name string) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	var query string
	switch currentDBType {
	case DBTypePostgres:
		query = `DELETE FROM custom_roles WHERE name = $1`
	default:
		query = `DELETE FROM custom_roles WHERE name = ?`
	}

	result, err := DB.Exec(query, name)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("custom role not found: %s", name)
	}
	return nil
}
