package util

import "path/filepath"

var (
	ParkConfigDir = filepath.Join(HomeDir(), ".config", "park")
)
