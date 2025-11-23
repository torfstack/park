package remote

import "github.com/torfstack/park/internal/config"

type FileTable struct {
	files map[FileId]FileTableEntry
	cfg   config.Config
}

type FileId string

type FileTableEntry struct {
	FileId      string
	ContentHash []byte
	Path        string
}
