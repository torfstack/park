package service

import (
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"

	"github.com/schollz/progressbar/v3"
	"github.com/torfstack/park/internal/logging"
	"github.com/torfstack/park/internal/util"
	"google.golang.org/api/drive/v3"
)

const (
	FolderMimeType = "application/vnd.google-apps.folder"
)

// SetupAndInitialSync initializes the drive directory and does a first synchronization
func (s *Service) SetupAndInitialSync() {
	if !s.cfg.IsSetup {
		logging.Logf("Enter Google Drive directory path (default: %s): ", filepath.Join(util.HomeDir(), "GoogleDrive"))
		var input string
		fmt.Scanln(&input)
		if input == "" {
			input = filepath.Join(util.HomeDir(), "GoogleDrive")
		}
		err := os.MkdirAll(input, 0755)
		if err != nil {
			logging.Logf("Could not create Google Drive directory '%s': %s", input, err)
			os.Exit(1)
		}
		entries, err := os.ReadDir(input)
		if err != nil {
			logging.Logf("Could not read Google Drive directory '%s': %s", input, err)
			os.Exit(1)
		}
		if len(entries) > 0 {
			logging.Log("directory is not empty. Please specify an empty directory.")
			os.Exit(1)
		}
		s.cfg.DriveDir = input
		s.cfg.IsSetup = true
		s.cfg.PersistConfig()
	}
	if !s.cfg.IsInitialized {
		s.download(s.cfg.DriveDir)
		s.cfg.IsInitialized = true
		s.cfg.PersistConfig()
	}
}

type job struct {
	file    *drive.File
	parents map[string]string
}

func (s *Service) download(intoDir string) {
	q := fmt.Sprintf("trashed = false")
	files, err := s.drv.Files.List().
		Q(q).
		Fields("files(id, name, mimeType, parents)").
		Do()
	if err != nil {
		logging.Logf("Could not list directories: %s", err)
		os.Exit(1)
	}

	parents := s.createDirs(files)

	numWorkers := 8
	jobs := make(chan job, numWorkers)
	results := make(chan error, numWorkers)

	var wg sync.WaitGroup
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for j := range jobs {
				if err := s.downloadFile(j.file, j.parents); err != nil {
					results <- err
					return // or continue based on desired behavior
				}
			}
		}()
	}

	s.downloadFiles(files, parents, jobs)
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	for err := range results {
		if err != nil {
			// handle
		}
	}
	logging.Log("Downloads finished!")

	os.Exit(0)
}

func (s *Service) startWorker(jobs <-chan job, results chan<- error) {
	for j := range jobs {
		if err := s.downloadFile(j.file, j.parents); err != nil {
			results <- err
		}
	}

	results <- nil
}

func (s *Service) createDirs(files *drive.FileList) map[string]string {
	fileMap := make(map[string]*drive.File)
	for _, f := range files.Files {
		fileMap[f.Id] = f
	}

	p := make(map[string]string)
	for _, d := range files.Files {
		if len(d.Parents) != 0 && slices.Contains(slices.Collect(maps.Keys(fileMap)), d.Parents[0]) {
			p[d.Name] = fileMap[d.Parents[0]].Name
		}
	}

	for _, d := range files.Files {
		if d.MimeType == FolderMimeType {
			fullpath := filepath.Join(s.cfg.DriveDir, fullPath(d.Name, p))
			os.MkdirAll(fullpath, 0755)
		}
	}

	return p
}

func fullPath(dirName string, parents map[string]string) string {
	for parents[dirName] != "" {
		dirName = filepath.Join(parents[dirName], dirName)
	}
	return dirName
}

func (s *Service) downloadFiles(files *drive.FileList, parents map[string]string, jobs chan<- job) {
	for _, file := range files.Files {
		if file.MimeType != FolderMimeType {
			jobs <- job{file, parents}
		}
	}
}

func (s *Service) downloadFile(f *drive.File, parents map[string]string) error {
	res, err := s.drv.Files.Get(f.Id).Download()
	if err != nil {
		logging.LogDebugf("Could not download file: %s", err)
		return err
	}
	defer res.Body.Close()

	localPath := filepath.Join(s.cfg.DriveDir, fullPath(f.Name, parents))
	logging.LogDebugf("Downloading %s to %s", f.Name, localPath)

	out, err := os.Create(localPath)

	totalSize := res.Header.Get("Content-Length")
	if totalSize != "" {
		total, _ := strconv.ParseInt(totalSize, 10, 64)
		bar := progressbar.DefaultBytes(total, f.Name)
		_, err = io.Copy(io.MultiWriter(out, bar), res.Body)
		bar.Finish()
	} else {
		_, err = io.Copy(out, res.Body)
	}
	out.Close()

	return nil
}

func (s *Service) isSetupAlready() bool {
	return s.cfg.IsSetup &&
		s.cfg.IsInitialized &&
		s.cfg.DriveDir != ""
}
