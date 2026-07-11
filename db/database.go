package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type ChatInfo struct {
	ChatID      string
	Model       string
	LastSummary string
}

type Database interface {
	UpdateChatInfo(chatId, model, summary string) error
	GetModelForChatId(chatId string) (string, error)
	GetLastSummary(chatId string) (string, error)
	Close()
}

type SqlDB struct {
	*sql.DB
}

var database Database

func InitDB(connStr string) error {
	conn, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err = conn.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)

	sqlDB := &SqlDB{DB: conn}

	if err := createTables(sqlDB); err != nil {
		conn.Close()
		return fmt.Errorf("failed to create tables: %w", err)
	}

	database = sqlDB
	return nil
}

func GetDB() Database {
	return database
}

func (d *SqlDB) UpdateChatInfo(chatId, model, summary string) error {
	if model == "" && summary == "" {
		return nil
	}

	query := `UPDATE chat_info SET model = COALESCE($1, chat_info.model), last_summary = COALESCE($2, chat_info.last_summary) WHERE chat_id = $3`
	_, err := d.Exec(query, nullIfEmpty(model), nullIfEmpty(summary), chatId)
	if err != nil {
		return fmt.Errorf("failed to update chat info for chat id: %w", err)
	}
	return nil
}

func (d *SqlDB) GetModelForChatId(chatId string) (string, error) {
	var model string
	err := d.QueryRow(`SELECT model FROM chat_info WHERE chat_id = $1`, chatId).Scan(&model)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to get model for chat id: %w", err)
	}
	return model, nil
}

func (d * SqlDB) GetLastSummary(chatId string) (string, error) {
	var summary string
	err := d.QueryRow(`SELECT last_summary from chat_info WHERE chat_id = $1`, chatId).Scan(&summary)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to get last article for chat id: %w", err)
	}
	return summary, nil
}

func createTables(d *SqlDB) error {
	chatInfoTable := `
	CREATE TABLE IF NOT EXISTS chat_info (
		chat_id TEXT PRIMARY KEY,
		model TEXT,
		last_summary TEXT
	);`

	if _, err := d.Exec(chatInfoTable); err != nil {
		return err
	}
	return nil
}

func Close() {
	if database != nil {
		database.Close()
	}
}

func (d *SqlDB) Close() {
	if d != nil && d.DB != nil {
		d.DB.Close()
	}
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
