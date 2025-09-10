package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"evently/internal/shared/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB holds database connections
type DB struct {
	PostgreSQL *gorm.DB
}

// InitDB initializes the database connections
func InitDB(cfg *config.Config) (*DB, error) {
	// Initialize PostgreSQL
	pg, err := initPostgreSQL(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PostgreSQL: %w", err)
	}
	err = Migrate(pg)
	if err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &DB{
		PostgreSQL: pg,
	}, nil
}

// initPostgreSQL initializes PostgreSQL connection with GORM
func initPostgreSQL(cfg *config.Config) (*gorm.DB, error) {
	// Configure GORM logger
	var gormLogger logger.Interface
	if cfg.IsDevelopment() {
		gormLogger = logger.Default.LogMode(logger.Info)
	} else {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}

	// GORM configuration
	gormConfig := &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		PrepareStmt:                              true,
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("✅ PostgreSQL connected successfully")
	return db, nil
}

// Close closes all database connections
func (db *DB) Close() error {
	// Close PostgreSQL
	if db.PostgreSQL != nil {
		if sqlDB, err := db.PostgreSQL.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				return fmt.Errorf("failed to close PostgreSQL: %w", err)
			}
		}
	}

	log.Println("✅ Database connection closed")
	return nil
}

// HealthCheck performs health checks on all database connections
func (db *DB) HealthCheck(ctx context.Context) error {
	// Check PostgreSQL
	if db.PostgreSQL != nil {
		sqlDB, err := db.PostgreSQL.DB()
		if err != nil {
			return fmt.Errorf("PostgreSQL health check failed: %w", err)
		}
		if err := sqlDB.PingContext(ctx); err != nil {
			return fmt.Errorf("PostgreSQL ping failed: %w", err)
		}
	}

	return nil
}

// BeginTx starts a new database transaction
func (db *DB) BeginTx(ctx context.Context) *gorm.DB {
	return db.PostgreSQL.WithContext(ctx).Begin()
}

// GetPostgreSQL returns the PostgreSQL GORM instance
func (db *DB) GetPostgreSQL() *gorm.DB {
	return db.PostgreSQL
}
