package sync

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/torfstack/park/internal/auth"
	"github.com/torfstack/park/internal/config"
	"github.com/torfstack/park/internal/util"
	"google.golang.org/api/drive/v3"
)

type Service struct {
	drv *drive.Service
	cfg config.Config
}

func NewService(cfg config.Config) *Service {
	drv, err := auth.DriveService()
	if err != nil {
		panic(err)
	}
	return &Service{drv, cfg}
}

func (s *Service) CheckForChanges(ctx context.Context) {
	files, err := s.drv.Files.List().Fields("nextPageToken, files(id, name)").Do()
	if err != nil {
		panic(err)
	}
	fmt.Println("Changes:")
	for _, file := range files.Files {
		fmt.Println(file.Name)
	}
}

func (s *Service) DownloadFile(fileId string) {
	f, err := s.drv.Files.Get(fileId).Download()
	if err != nil {
		panic(err)
	}
	localPath := fileId + ".txt"
	defer f.Body.Close()
	out, _ := os.Create(localPath)
	written, err := io.Copy(out, f.Body)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Wrote %d bytes to %s\n", written, localPath)
}

func (s *Service) UploadFile(localPath string) {
	absoluteLocalPath, err := util.CanonizePath(s.cfg.DriveDir, localPath)
	if err != nil {
		panic(err)
	}
	file, err := os.Open(absoluteLocalPath)
	if err != nil {
		panic(err)
	}
	f := &drive.File{Name: filepath.Base(localPath)}
	s.drv.Files.Create(f).Media(file).Do()
}
