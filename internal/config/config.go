package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/torfstack/park/internal/logging"
	"github.com/torfstack/park/internal/util"
)

var (
	configFilePath  = filepath.Join(util.ParkConfigDir, "config.toml")
	defaultDriveDir = filepath.Join(util.HomeDir(), "drive")
	inputFile       = os.Stdin
)

type Config struct {
	LocalDir          string `toml:"local_dir"`
	RemoteInitialized bool   `toml:"remote_initialized"`
}

func Get(interactive bool) (Config, error) {
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
	c := Config{LocalDir: defaultDriveDir}
	if interactive {
		reader := bufio.NewReader(inputFile)
		fmt.Printf("Enter local directory path for drive to sync to [default: %s]: ", c.LocalDir)
		input, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return c, fmt.Errorf("could not read user input: %w", err)
		}

		input = strings.TrimSpace(input)
		if input != "" {
			c.LocalDir = input
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
