package config

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/torfstack/park/internal/db"
	"github.com/torfstack/park/internal/db/sqlc"
	"github.com/torfstack/park/internal/util"
)

var (
	defaultDriveDir     = filepath.Join(util.HomeDir(), "park-drive")
	defaultSyncInterval = 60 * time.Second
)

type Config struct {
	LocalDir     string        `toml:"local_dir"`
	SyncInterval time.Duration `toml:"sync_interval"`
}

func Get(ctx context.Context) (Config, error) {
	return get(ctx, false)
}

func GetInteractive(ctx context.Context) (Config, error) {
	return get(ctx, true)
}

func get(ctx context.Context, interactive bool) (Config, error) {
	d, err := db.New(ctx)
	if err != nil {
		return Config{}, fmt.Errorf("could not create database: %w", err)
	}
	defer d.Close()

	c, err := d.Queries().GetConfig(ctx)
	if err != nil {
		return Config{}, fmt.Errorf("could not get config from database: %w", err)
	}

	config := Config{}
	config.LocalDir = c.RootDir
	config.SyncInterval = time.Duration(c.SyncInterval) * time.Second
	if config.isNotInitialized() {
		config, err = initConfig(ctx, interactive)
		if err != nil {
			return Config{}, fmt.Errorf("could not initialize config: %w", err)
		}
	}

	return config, nil
}

func initConfig(ctx context.Context, interactive bool) (Config, error) {
	c := initialConfig()
	if interactive {
		err := guidedInitialization(&c)
		if err != nil {
			return c, fmt.Errorf("could not initialize config interactively: %w", err)
		}
	}
	return c, c.persist(ctx)
}

func (c *Config) persist(ctx context.Context) error {
	d, err := db.New(ctx)
	if err != nil {
		return fmt.Errorf("could not create database: %w", err)
	}
	defer d.Close()

	err = d.Queries().UpsertConfig(ctx, sqlc.UpsertConfigParams{
		RootDir:      c.LocalDir,
		SyncInterval: int64(c.SyncInterval.Seconds()),
	})
	if err != nil {
		return fmt.Errorf("could not persist config: %w", err)
	}
	return nil
}

func initialConfig() Config {
	return Config{SyncInterval: defaultSyncInterval, LocalDir: defaultDriveDir}
}

func (c *Config) isNotInitialized() bool {
	return c.LocalDir == ""
}
