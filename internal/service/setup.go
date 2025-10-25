package service

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/schollz/progressbar/v3"
	"github.com/torfstack/park/internal/logging"
	"github.com/torfstack/park/internal/util"
	"google.golang.org/api/drive/v3"
)

const (
	FolderMimeType      = "application/vnd.google-apps.folder"
	SpreadSheetMimeType = "application/vnd.google-apps.spreadsheet"
)

var NotDownloadableMimeTypes = []string{
	FolderMimeType,
	SpreadSheetMimeType,
}

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
		s.download()
		s.cfg.IsInitialized = true
		s.cfg.PersistConfig()
	}
}

type job struct {
	file    *drive.File
	syncCtx *syncContext
}

func (s *Service) download() {
	q := fmt.Sprintf("trashed = false")
	var files []*drive.File
	nextPageToken := ""
	initial := true
	for nextPageToken != "" || initial {
		logging.LogDebugf("Fetching files with pageToken '%s'", nextPageToken)
		initial = false
		fileList, err := s.drv.Files.List().
			Q(q).
			PageToken(nextPageToken).
			PageSize(300).
			Corpora("user").
			Fields("nextPageToken, files(id, name, mimeType, parents, capabilities, ownedByMe)").
			Do()
		if err != nil {
			logging.Logf("Could not list directories: %s", err)
			os.Exit(1)
		}
		files = append(files, fileList.Files...)
		nextPageToken = fileList.NextPageToken
	}

	syncCtx := createSyncContext(files)
	logging.LogDebug("Created sync context")

	s.createDirs(files, syncCtx)
	logging.LogDebug("Created initial directories")

	numWorkers := 8
	jobs := make(chan job, numWorkers)
	results := make(chan error, numWorkers)

	logging.LogDebug("Starting download workers")
	var wg sync.WaitGroup
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for j := range jobs {
				if err := s.downloadFile(j.file, syncCtx); err != nil {
					results <- err
				}
			}
		}()
	}

	s.downloadFiles(files, jobs, syncCtx)
	close(jobs)

	wg.Wait()
	close(results)

	logging.Log("Downloads finished!")

	os.Exit(0)
}

func createSyncContext(files []*drive.File) *syncContext {
	fileMap := make(map[string]*drive.File)
	parents := make(map[string]string)
	for _, file := range files {
		fileMap[file.Id] = file
		if len(file.Parents) > 0 {
			parents[file.Id] = file.Parents[0]
		}
	}
	return &syncContext{fileMap, parents}
}

type syncContext struct {
	fileMap map[string]*drive.File
	parents map[string]string
}

func (s *Service) createDirs(files []*drive.File, syncCtx *syncContext) {
	for _, dir := range files {
		if dir.MimeType == FolderMimeType && dir.OwnedByMe {
			fullpath := filepath.Join(s.cfg.DriveDir, fullPath(dir.Id, syncCtx))
			os.MkdirAll(fullpath, 0755)
		}
	}
}

func fullPath(id string, syncCtx *syncContext) string {
	parents := syncCtx.parents
	fileMap := syncCtx.fileMap

	f := fileMap[id].Name
	// fileMap[parents[id]] != nil checks for the root directory
	for parents[id] != "" && fileMap[parents[id]] != nil {
		f = filepath.Join(fileMap[parents[id]].Name, f)
		id = parents[id]
	}
	return f
}

func (s *Service) downloadFiles(files []*drive.File, jobs chan<- job, syncCtx *syncContext) {
	for _, file := range files {
		if file.MimeType != FolderMimeType && file.OwnedByMe && file.Capabilities.CanDownload {
			jobs <- job{file, syncCtx}
		}
	}
}

func (s *Service) downloadFile(f *drive.File, syncCtx *syncContext) error {
	res, err := s.drv.Files.Get(f.Id).Download()
	if err != nil {
		logging.LogDebugf("Could not download file '%s': %s", f.Name, err)
		return err
	}
	defer res.Body.Close()

	localPath := filepath.Join(s.cfg.DriveDir, fullPath(f.Id, syncCtx))
	logging.LogDebugf("Downloading %s to %s", f.Name, localPath)

	out, err := os.Create(localPath)

	totalSize := res.Header.Get("Content-Length")
	if total, err := strconv.ParseInt(totalSize, 10, 64); err == nil && total > 10*util.MB {
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
