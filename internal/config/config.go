package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/torfstack/park/internal/logging"
	"github.com/torfstack/park/internal/util"
)

var (
	configFilePath      = filepath.Join(util.ParkConfigDir, "config.toml")
	defaultDriveDir     = filepath.Join(util.HomeDir(), "park-drive")
	defaultSyncInterval = 60 * time.Second
)

type Config struct {
	LocalDir     string        `toml:"local_dir"`
	SyncInterval time.Duration `toml:"sync_interval"`
}

func Get() (Config, error) {
	return get(false)
}

func GetInteractive() (Config, error) {
	return get(true)
}

func get(interactive bool) (Config, error) {
	c := Config{}
	f, err := os.Open(configFilePath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return initConfig(interactive)
	case err != nil:
		return c, fmt.Errorf("could not open config file for reading '%s': %s", configFilePath, err)
	}

	_, err = toml.NewDecoder(f).Decode(&c)
	if err != nil {
		return c, fmt.Errorf("could not decode config file '%s': %s", configFilePath, err)
	}
	return c, nil
}

func initConfig(interactive bool) (Config, error) {
	c := initialConfig()
	if interactive {
		err := guidedInitialization(&c)
		if err != nil {
			return c, fmt.Errorf("could not initialize config interactively: %w", err)
		}
	}
	return c, c.persist()
}

func (c *Config) persist() error {
	f, err := util.OpenWithParents(configFilePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("could not open config file for writing '%s': %w", configFilePath, err)
	}

	logging.Debugf("Persisting config file to '%s'", configFilePath)
	err = toml.NewEncoder(f).Encode(c)
	if err != nil {
		return fmt.Errorf("could not persist config to file '%s': %w", configFilePath, err)
	}

	return nil
}

func initialConfig() Config {
	return Config{SyncInterval: defaultSyncInterval, LocalDir: defaultDriveDir}
}
