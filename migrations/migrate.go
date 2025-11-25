package migrations

import (
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var embedMigrations embed.FS

func Run(dbURL string) error {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}

	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
