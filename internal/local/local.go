package local

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/torfstack/park/internal/config"
	"github.com/torfstack/park/internal/logging"
	"github.com/torfstack/park/internal/util"
)

type ParkTable struct {
	config config.Config
	files  map[string]ParkFile
}

func NewParkTable(cfg config.Config, files map[string]ParkFile) *ParkTable {
	return &ParkTable{cfg, files}
}

func (p *ParkTable) Persist() error {
	f, err := os.OpenFile(parkTablePath(), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer func(f *os.File) {
		err = f.Close()
		if err != nil {
			logging.Logf("Could not close park table file: %s", err)
		}
	}(f)
	for _, file := range p.files {
		_, err = f.WriteString(file.Serialize() + "\n")
		if err != nil {
			return fmt.Errorf("could not write park table file to file: %w", err)
		}
	}
	return f.Sync()
}

func (p *ParkTable) Exists(id string) bool {
	_, ok := p.files[id]
	return ok
}

func (p *ParkTable) Remove(id string) error {
	if file, ok := p.files[id]; ok {
		err := file.remove()
		if err != nil {
			return fmt.Errorf("could not remove file: %w", err)
		}
		delete(p.files, id)
	}
	return nil
}

func (p *ParkTable) Update(id string, content io.ReadCloser) error {
	defer func(content io.ReadCloser) {
		err := content.Close()
		if err != nil {
			logging.Logf("could not close content after update: %s", err)
		}
	}(content)
	if file, ok := p.files[id]; ok {
		err := file.update(content)
		if err != nil {
			return fmt.Errorf("could not remove file: %w", err)
		}
	} else {
		return fmt.Errorf("could not find file with id '%s', can not update", id)
	}
	return nil
}

func (p *ParkTable) Create(id, name string, content io.ReadCloser) error {
	defer func(content io.ReadCloser) {
		err := content.Close()
		if err != nil {
			logging.Logf("could not close content after update: %s", err)
		}
	}(content)
	var f *ParkFile
	f, err := create(content, filepath.Join(p.config.LocalDir, name), id)
	if err != nil {
		return fmt.Errorf("could not create file: %w", err)
	}
	p.files[id] = *f
	return nil
}

func LoadParkTable(cfg config.Config) (*ParkTable, error) {
	file, err := os.Open(parkTablePath())
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("could not stat park table file: %w", err)
	}
	scanner := bufio.NewScanner(file)
	files := make(map[string]ParkFile)
	for scanner.Scan() {
		var parkFile *ParkFile
		parkFile, err = DeserializeParkFile(scanner.Text())
		if err != nil {
			return nil, fmt.Errorf("could not deserialize park table file: %w", err)
		}
		files[parkFile.FileId] = *parkFile
	}
	return &ParkTable{cfg, files}, nil
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
		return nil, fmt.Errorf("could not decode content hash: %w", err)
	}
	return &ParkFile{
		Path:        parts[0],
		FileId:      parts[1],
		ContentHash: contentHash,
	}, nil
}

func (p *ParkFile) remove() error {
	err := os.Remove(p.Path)
	if err != nil {
		return fmt.Errorf("could not remove at path '%s': %w", p.Path, err)
	}
	return nil
}

func create(content io.ReadCloser, path, id string) (*ParkFile, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("could not open file for writing: %w", err)
	}
	defer func(f *os.File) {
		err = f.Close()
		if err != nil {
			logging.Logf("Could not close file: %s", err)
		}
	}(file)
	_, err = io.Copy(file, content)
	if err != nil {
		return nil, fmt.Errorf("could not copy content to file: %w", err)
	}
	return &ParkFile{
		Path:        path,
		FileId:      id,
		ContentHash: nil,
	}, nil
}

func (p *ParkFile) update(content io.ReadCloser) error {
	file, err := os.OpenFile(p.Path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("could not open file for writing: %w", err)
	}
	defer func(f *os.File) {
		err = f.Close()
		if err != nil {
			logging.Logf("Could not close file: %s", err)
		}
	}(file)
	_, err = io.Copy(file, content)
	if err != nil {
		return fmt.Errorf("could not copy content to file: %w", err)
	}
	return nil
}
