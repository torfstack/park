package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/torfstack/park/internal/logging"
	"github.com/torfstack/park/internal/util"
)

type Config struct {
	IsSetup       bool   `toml:"is_setup"`
	IsInitialized bool   `toml:"is_initialized"`
	DriveDir      string `toml:"drive_dir"`
}

func (c *Config) PersistConfig() error {
	err := os.MkdirAll(configDirPath(), 0755)
	if err != nil {
		return fmt.Errorf("could not create config directory '%s': %s", configDirPath(), err)
	}

	f, err := os.OpenFile(configPath(), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("could not open config file for writing '%s': %s", configPath(), err)
	}

	logging.LogDebugf("Persisting config file to '%s'", configPath())
	err = toml.NewEncoder(f).Encode(c)
	if err != nil {
		return fmt.Errorf("could not persist config to file '%s': %s", configPath(), err)
	}
	return nil
}

func LoadConfig() (Config, error) {
	c := Config{}
	f, err := os.Open(configPath())
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return c, nil
	case err != nil:
		return c, fmt.Errorf("could not open config file for reading '%s': %s", configPath(), err)
	}

	_, err = toml.NewDecoder(f).Decode(&c)
	if err != nil {
		return c, fmt.Errorf("could not decode config file '%s': %s", configPath(), err)
	}
	return c, nil
}

func configPath() string {
	return filepath.Join(configDirPath(), "config.toml")
}

func configDirPath() string {
	return util.HomeDir() + "/.config/park"
}
