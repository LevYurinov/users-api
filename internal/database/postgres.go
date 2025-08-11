package database

import (
	"database/sql"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
	"os"
)

func ConnectDB(dbName string, log *zap.Logger) (*sql.DB, error) {

	connStr := os.Getenv("POSTGRES_DSN")
	if connStr == "" {
		connStr = fmt.Sprintf("host=localhost port=5432 user=postgres password=postgres dbname=%s sslmode=disable", dbName)

		log.Error("POSTGRES_DSN is empty", // отсутствие строки подкл. это проблема
			zap.String("db_name", dbName),
			zap.String("component", "database"),
			zap.String("operation", "Getenv"))

		return nil, fmt.Errorf("database/ConnectDB: db connection error")
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Error("open db error",
			zap.Error(err),
			zap.String("db_name", dbName),
			zap.String("component", "database"),
			zap.String("operation", "Open"))

		return nil, fmt.Errorf("database/ConnectDB: open db error: %w", err)
	}

	err = db.Ping()
	if err != nil {
		log.Error("connection db error",
			zap.Error(err),
			zap.String("db_name", dbName),
			zap.String("component", "database"),
			zap.String("operation", "Ping"))

		return nil, fmt.Errorf("database/ConnectDB: connection db error: %w", err)
	}

	log.Info("db connection set",
		zap.String("db_name", dbName),
		zap.String("component", "database"),
		zap.String("operation", "connection"))

	return db, nil
}
