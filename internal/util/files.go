package util

import (
	"os"
	"path/filepath"
)

// HomeDir returns the user's home directory.'
func HomeDir() string {
	h, _ := os.UserHomeDir()
	return h
}

// OpenWithParents opens a file at the given path with the given flag and creates all parent directories if necessary.
func OpenWithParents(path string, flag int, perm os.FileMode) (*os.File, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return os.OpenFile(path, flag, perm)
}

// WriteFile creates a file at the given path with the given data and creates all parent directories if necessary.
func WriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil
	}
	return os.WriteFile(path, data, 0644)
}

// CreateTempDir creates a temporary directory with a prefix of "park-"
func CreateTempDir() (string, error) {
	dir, err := os.MkdirTemp("", "park-")
	if err != nil {
		return "", err
	}
	return dir, nil
}
