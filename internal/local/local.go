package local

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/torfstack/park/internal/util"
)

type ParkTable struct {
	Files []ParkFile
}

func (p *ParkTable) Persist() error {
	f, err := os.OpenFile(parkTablePath(), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	for _, file := range p.Files {
		_, err := f.WriteString(file.Serialize() + "\n")
		if err != nil {
			panic(err)
		}
	}
	return f.Sync()
}

func LoadParkTable() *ParkTable {
	_, err := os.Stat(parkTablePath())
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		panic(err)
	}
	file, err := os.Open(parkTablePath())
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(file)
	var files []ParkFile
	for scanner.Scan() {
		parkFile, err := DeserializeParkFile(scanner.Text())
		if err != nil {
			panic(err)
		}
		files = append(files, *parkFile)
	}
	return &ParkTable{files}
}

func parkTablePath() string {
	return filepath.Join(util.HomeDir(), ".config", "park", "parkTable")
}

type ParkFile struct {
	Path        string
	FileId      string
	ContentHash []byte
}

func (p *ParkFile) Serialize() string {
	contentHashEncoded := base64.StdEncoding.EncodeToString(p.ContentHash)
	return fmt.Sprintf("%s:%s:%s", p.Path, p.FileId, contentHashEncoded)
}

func DeserializeParkFile(s string) (*ParkFile, error) {
	parts := strings.Split(s, ":")
	contentHash, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}
	return &ParkFile{
		Path:        parts[0],
		FileId:      parts[1],
		ContentHash: contentHash,
	}, nil
}
