package config

import (
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

func (c *Config) PersistConfig() {
	err := os.MkdirAll(configDirPath(), 0755)
	if err != nil {
		logging.Logf("Could not create config directory '%s': %s", configDirPath(), err)
		os.Exit(1)
	}

	f, err := os.OpenFile(configPath(), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		logging.Logf("Could not open config file for writing '%s': %s", configPath(), err)
		os.Exit(1)
	}

	logging.LogDebugf("Persisting config file to '%s'", configPath())
	err = toml.NewEncoder(f).Encode(c)
	if err != nil {
		logging.Logf("Could not persist config to file '%s': %s", configPath(), err)
		os.Exit(1)
	}
}

func LoadConfig() Config {
	c := Config{}
	_, err := os.Stat(configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return c
		}
		logging.Logf("Could not stat config file '%s': %s", configPath(), err)
		os.Exit(1)
	}

	f, err := os.Open(configPath())
	if err != nil {
		logging.Logf("Could not open config file for reading '%s': %s", configPath(), err)
		os.Exit(1)
	}

	_, err = toml.NewDecoder(f).Decode(&c)
	if err != nil {
		logging.Logf("Could not decode config file '%s': %s", configPath(), err)
		os.Exit(1)
	}

	return c
}

func configPath() string {
	return filepath.Join(configDirPath(), "config.toml")
}

func configDirPath() string {
	return util.HomeDir() + "/.config/park"
}
