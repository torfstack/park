package util

import (
	"os"
	"path/filepath"
)

const (
	KB int64 = 1024
	MB       = 1024 * KB
	GB       = 1024 * MB
)

func CanonizePath(basePath, relPath string) (string, error) {
	joined := filepath.Join(basePath, relPath)
	cleaned := filepath.Clean(joined)
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}

func HomeDir() string {
	h, _ := os.UserHomeDir()
	return h
}
