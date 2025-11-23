package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"path/filepath"

	"github.com/pressly/goose/v3"
	"github.com/torfstack/park/internal/logging"
	"github.com/torfstack/park/internal/util"
	_ "modernc.org/sqlite"

	"github.com/torfstack/park/internal/config"
)

//go:generate sqlc generate -f sql/sqlc.yaml

var (
	dbName = "park.sqlite"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

type Database struct {
	cfg config.Config
	db  *sql.DB
}

func NewDatabase(ctx context.Context, cfg config.Config) (*Database, error) {
	fp := filepath.Join(util.ParkConfigDir, dbName)
	sqlDb, err := sql.Open("sqlite", fp)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}
	d := &Database{cfg, sqlDb}
	err = d.runMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not run migrations: %w", err)
	}
	return d, nil
}

func (d *Database) runMigrations(ctx context.Context) error {
	err := goose.SetDialect("sqlite")
	if err != nil {
		return fmt.Errorf("could not set dialect 'sqlite': %w", err)
	}
	goose.SetLogger(logging.ParkLoggerGoose{})
	goose.SetBaseFS(embedMigrations)

	if err = goose.UpContext(ctx, d.db, "migrations"); err != nil {
		return fmt.Errorf("could not run migrations: %w", err)
	}
	return nil
}

func (d *Database) Close() error {
	if d.db == nil {
		return nil
	}
	return d.db.Close()
}
