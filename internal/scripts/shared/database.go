package shared

import (
	"errors"
	"fmt"
	"prime-erp-core/internal/db"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ExecuteSeedStatements executes SQL statements within a transaction
func ExecuteSeedStatements(databaseName string, connectionString string, statements []string) error {
	gormDB, err := GetDatabaseConnection(databaseName, connectionString)
	if err != nil {
		return err
	}
	defer db.CloseGORM(gormDB)

	return ExecuteStatementsInTransaction(gormDB, statements)
}

// ExecuteStatementsInTransaction executes SQL statements within a transaction on an existing DB connection
func ExecuteStatementsInTransaction(gormDB *gorm.DB, statements []string) error {
	if gormDB == nil {
		return errors.New("gorm DB instance is nil")
	}

	tx := gormDB.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	for _, stmt := range statements {
		if err := tx.Exec(stmt).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute statement: %w\nStatement: %s", err, stmt)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ExecuteStatementsWithLabel executes SQL statements with a label for error messages
func ExecuteStatementsWithLabel(tx *gorm.DB, statements []string, label string) error {
	for _, stmt := range statements {
		if err := tx.Exec(stmt).Error; err != nil {
			return fmt.Errorf("failed to execute %s statement: %w", label, err)
		}
	}
	return nil
}

// GetDatabaseConnection returns a GORM database connection
func GetDatabaseConnection(databaseName string, connectionString string) (*gorm.DB, error) {
	if connectionString != "" {
		// Use direct connection string
		return ConnectWithString(connectionString)
	}
	// Use environment variable approach
	return db.ConnectGORM(databaseName)
}

// ConnectWithString creates a GORM database connection using a connection string
func ConnectWithString(connectionString string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	return db, nil
}
