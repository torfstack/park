package service

import (
	"bufio"
	"context"
	"crypto"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/torfstack/park/internal/local"
	"github.com/torfstack/park/internal/logging"
	"github.com/torfstack/park/internal/util"
	"google.golang.org/api/drive/v3"
)

const (
	FolderMimeType   = "application/vnd.google-apps.folder"
	ShortcutMimeType = "application/vnd.google-apps.shortcut"
	RootFolderId     = "root"

	NumWorkers = 4
)

// SetupAndInitialSync initializes the drive directory and does a first synchronization
func (s *Service) SetupAndInitialSync() error {
	if !s.cfg.IsSetup {
		logging.Logf("Enter Google Drive directory path (default: %s): ", filepath.Join(util.HomeDir(), "GoogleDrive"))
		var input string
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("could not read input: %w", err)
		}
		input = strings.TrimSpace(input)
		if input == "" {
			input = filepath.Join(util.HomeDir(), "GoogleDrive")
		}
		err = os.MkdirAll(input, 0755)
		if err != nil {
			return fmt.Errorf("could not create Google Drive directory '%s': %w", input, err)
		}
		entries, err := os.ReadDir(input)
		if err != nil {
			return fmt.Errorf("could not read Google Drive directory '%s': %w", input, err)
		}
		if len(entries) > 0 {
			return errors.New("directory is not empty. Please specify an empty directory")
		}
		s.cfg.DriveDir = input
		s.cfg.IsSetup = true
		if err = s.cfg.PersistConfig(); err != nil {
			return fmt.Errorf("could not persist config: %w", err)
		}
	}
	if !s.cfg.IsInitialized {
		err := s.performInitialSync()
		if err != nil {
			return fmt.Errorf("could not perform initial sync: %w", err)
		}
		s.cfg.IsInitialized = true
		if err = s.cfg.PersistConfig(); err != nil {
			return fmt.Errorf("could not persist config: %w", err)
		}
	}
	return nil
}

type job struct {
	file    *drive.File
	syncCtx *syncContext
}

func (s *Service) performInitialSync() error {
	syncCtx := syncContext{
		fileMap: make(map[string]*drive.File),
		parents: make(map[string]string),
	}

	err := s.walkFolder(context.Background(), RootFolderId, s.cfg.DriveDir, &syncCtx)
	if err != nil {
		return fmt.Errorf("error walking root folder: %w", err)
	}

	err = s.createDirs(&syncCtx)
	if err != nil {
		return fmt.Errorf("error creating initial directories: %w", err)
	}
	logging.LogDebug("Created initial directories")

	jobs := make(chan job)
	results := make(chan local.ParkFile)
	var wg sync.WaitGroup

	logging.LogDebug("Starting download workers")
	for i := 0; i < NumWorkers; i++ {
		wg.Go(func() {
			for j := range jobs {
				parkFile, errGo := s.downloadFile(j.file, &syncCtx)
				if errGo != nil {
					logging.LogDebugf("Skipping: could not download file '%s': %s", j.file.Name, errGo)
					continue
				}
				results <- *parkFile
			}
		})
	}
	go func() {
		s.enqueueDownloadJobs(jobs, &syncCtx)
		close(jobs)
		logging.LogDebug("Finished enqueueing download jobs")
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	parkFiles := make([]local.ParkFile, 0)
	for parkFile := range results {
		parkFiles = append(parkFiles, parkFile)
	}
	parkTable := local.ParkTable{Files: parkFiles}
	err = parkTable.Persist()
	if err != nil {
		return fmt.Errorf("error persisting initial park table: %w", err)
	}

	logging.LogDebug("Downloads finished!")
	return nil
}

func (s *Service) walkFolder(ctx context.Context, folderID, path string, syncCtx *syncContext) error {
	pageToken := ""
	for {
		req := s.drv.Files.List().
			Q(fmt.Sprintf("'%s' in parents and trashed=false", folderID)).
			Fields("nextPageToken, files(id, name, mimeType, parents, shortcutDetails)").
			PageSize(1000)

		if pageToken != "" {
			req = req.PageToken(pageToken)
		}

		r, err := req.Do()
		if err != nil {
			return fmt.Errorf("error listing files in %s: %w", path, err)
		}

		for _, f := range r.Files {
			if err = s.handleFile(ctx, f, folderID, path, syncCtx); err != nil {
				logging.Logf("error handling file %s: %w", f.Name, err)
				continue
			}
		}

		pageToken = r.NextPageToken
		if pageToken == "" {
			break
		}
	}
	return nil
}

func (s *Service) handleFile(ctx context.Context, f *drive.File, folderId, path string, syncCtx *syncContext) error {
	if f.DriveId != "" {
		return nil
	}

	fullPath := filepath.Join(path, f.Name)

	switch f.MimeType {
	case FolderMimeType:
		if err := s.walkFolder(ctx, f.Id, fullPath, syncCtx); err != nil {
			return fmt.Errorf("error walking folder %s: %w", f.Name, err)
		}
		syncCtx.parents[f.Id] = folderId
		syncCtx.fileMap[f.Id] = f
	case ShortcutMimeType:
		shortcut := f.ShortcutDetails
		if shortcut == nil {
			return fmt.Errorf("shortcut without details: %s (%s)", f.Name, f.Id)
		}

		targetID := shortcut.TargetId
		targetType := shortcut.TargetMimeType

		if targetType == FolderMimeType {
			// TODO: keep track of visited ids to not get into a shortcut loop
			if err := s.walkFolder(ctx, targetID, filepath.Join(path, f.Name), syncCtx); err != nil {
				return fmt.Errorf("error walking shortcut folder %s: %w", f.Name, err)
			}
			syncCtx.parents[targetID] = folderId
			syncCtx.fileMap[targetID] = f
		} else {
			shortcutFile, err := s.drv.Files.Get(targetID).Fields("id, name").Do()
			if err != nil {
				return fmt.Errorf("error getting shortcut target file %s: %w", f.Name, err)
			}
			syncCtx.parents[shortcutFile.Id] = folderId
			syncCtx.fileMap[shortcutFile.Id] = shortcutFile
		}
	default:
		syncCtx.parents[f.Id] = folderId
		syncCtx.fileMap[f.Id] = f
	}

	return nil
}

type syncContext struct {
	fileMap map[string]*drive.File
	parents map[string]string
}

func (s *Service) createDirs(syncCtx *syncContext) error {
	files := slices.Collect(maps.Values(syncCtx.fileMap))
	for _, f := range files {
		if f.MimeType == FolderMimeType {
			path := filepath.Join(s.cfg.DriveDir, localPath(f, syncCtx))
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
		}
	}
	return nil
}

func localPath(file *drive.File, syncCtx *syncContext) string {
	parents := syncCtx.parents
	fileMap := syncCtx.fileMap
	id := file.Id

	f := fileMap[id].Name
	// fileMap[parents[id]] != nil checks for the root directory
	for parents[id] != "" && fileMap[parents[id]] != nil {
		f = filepath.Join(fileMap[parents[id]].Name, f)
		id = parents[id]
	}
	return f
}

func (s *Service) enqueueDownloadJobs(jobs chan<- job, syncCtx *syncContext) {
	files := slices.Collect(maps.Values(syncCtx.fileMap))
	for _, file := range files {
		if file.MimeType != FolderMimeType {
			jobs <- job{file, syncCtx}
		}
	}
}

func (s *Service) downloadFile(f *drive.File, syncCtx *syncContext) (*local.ParkFile, error) {
	res, err := s.drv.Files.Get(f.Id).Download()
	if err != nil {
		return nil, fmt.Errorf("could not download file '%s': %w", f.Name, err)
	}
	defer func(Body io.ReadCloser) {
		if err = Body.Close(); err != nil {
			logging.LogDebugf("Could not close body: %s", err)
		}
	}(res.Body)

	absoluteLocalPath := filepath.Join(s.cfg.DriveDir, localPath(f, syncCtx))
	logging.LogDebugf("Downloading %s to %s", f.Name, absoluteLocalPath)

	out, err := os.Create(absoluteLocalPath)
	if err != nil {
		return nil, fmt.Errorf("could not create file '%s': %w", absoluteLocalPath, err)
	}

	sha := crypto.SHA3_256.New()
	_, err = io.Copy(io.MultiWriter(out, sha), res.Body)
	if err != nil {
		return nil, fmt.Errorf("could not write file '%s': %w", absoluteLocalPath, err)
	}
	err = out.Close()
	if err != nil {
		return nil, fmt.Errorf("could not close file '%s': %w", absoluteLocalPath, err)
	}

	return &local.ParkFile{
		Path:        absoluteLocalPath,
		FileId:      f.Id,
		ContentHash: sha.Sum(nil),
	}, nil
}

func (s *Service) isSetupAlready() bool {
	return s.cfg.IsSetup &&
		s.cfg.IsInitialized &&
		s.cfg.DriveDir != ""
}
