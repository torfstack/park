package util

import (
	"os"
	"path/filepath"
)

func HomeDir() string {
	h, _ := os.UserHomeDir()
	return h
}

func OpenWithParents(path string, flag int, perm os.FileMode) (*os.File, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return os.OpenFile(path, flag, perm)
}
