package util

import (
	"os"
)

func HomeDir() string {
	h, _ := os.UserHomeDir()
	return h
}
