package service

import (
	"bufio"
	"context"
	"crypto"
	"fmt"
	"io"
	"log"
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
	FolderMimeType      = "application/vnd.google-apps.folder"
	SpreadSheetMimeType = "application/vnd.google-apps.spreadsheet"
	ShortcutMimeType    = "application/vnd.google-apps.shortcut"
	RootFolderId        = "root"
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
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if err != nil {
			logging.Logf("Could not read input: %s", err)
			os.Exit(1)
		}
		if input == "" {
			input = filepath.Join(util.HomeDir(), "GoogleDrive")
		}
		err = os.MkdirAll(input, 0755)
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
	syncCtx := syncContext{
		fileMap: make(map[string]*drive.File),
		parents: make(map[string]string),
	}
	err := s.walkFolder(context.Background(), RootFolderId, s.cfg.DriveDir, &syncCtx)
	if err != nil {
		panic(err)
	}

	s.createDirs(&syncCtx)
	logging.LogDebug("Created initial directories")

	numWorkers := 4
	jobs := make(chan job, numWorkers)
	results := make(chan error, numWorkers)

	logging.LogDebug("Starting download workers")
	var wg sync.WaitGroup
	wg.Add(numWorkers)
	result := newSyncResult()
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for j := range jobs {
				if err := s.downloadFile(j.file, &syncCtx, result); err != nil {
					results <- err
				}
			}
		}()
	}

	s.downloadFiles(jobs, &syncCtx)
	close(jobs)

	wg.Wait()
	close(results)

	parkFiles := result.parkFiles.Items()
	parkTable := local.ParkTable{Files: parkFiles}
	parkTable.Persist()

	logging.LogDebug("Downloads finished!")
}

func (s *Service) walkFolder(ctx context.Context, folderID, path string, syncCtx *syncContext) error {
	logging.LogDebugf("Walking folder %s", path)
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
			// Skip any file that belongs to a shared drive
			if f.DriveId != "" {
				continue
			}

			fullPath := filepath.Join(path, f.Name)

			switch f.MimeType {
			case FolderMimeType:
				if err := s.walkFolder(ctx, f.Id, fullPath, syncCtx); err != nil {
					return err
				}
				syncCtx.parents[f.Id] = folderID
				syncCtx.fileMap[f.Id] = f
			case ShortcutMimeType:
				shortcut := f.ShortcutDetails
				if shortcut == nil {
					log.Printf("Shortcut without details: %s (%s)", f.Name, f.Id)
					continue
				}

				targetID := shortcut.TargetId
				targetType := shortcut.TargetMimeType

				if targetType == FolderMimeType {
					// TODO: keep track of visited ids to not get into a shortcut loop
					if err := s.walkFolder(ctx, targetID, filepath.Join(path, f.Name), syncCtx); err != nil {
						return err
					}
					syncCtx.parents[targetID] = folderID
					syncCtx.fileMap[targetID] = f
				} else {
					shortcutFile, err := s.drv.Files.Get(targetID).Fields("id, name").Do()
					if err != nil {
						return fmt.Errorf("error getting shortcut target file %s: %w", f.Name, err)
					}
					syncCtx.parents[shortcutFile.Id] = folderID
					syncCtx.fileMap[shortcutFile.Id] = shortcutFile
				}
			default:
				syncCtx.parents[f.Id] = folderID
				syncCtx.fileMap[f.Id] = f
			}
		}

		pageToken = r.NextPageToken
		if pageToken == "" {
			break
		}
	}
	return nil
}

type syncContext struct {
	fileMap map[string]*drive.File
	parents map[string]string
}

type syncResult struct {
	parkFiles util.SyncSlice[local.ParkFile]
}

func newSyncResult() *syncResult {
	return &syncResult{*util.NewSyncSlice[local.ParkFile]()}
}

func (s *Service) createDirs(syncCtx *syncContext) {
	files := slices.Collect(maps.Values(syncCtx.fileMap))
	for _, dir := range files {
		if dir.MimeType == FolderMimeType {
			fullpath := filepath.Join(s.cfg.DriveDir, fullPath(dir.Id, syncCtx))
			if err := os.MkdirAll(fullpath, 0755); err != nil {
				panic(err)
			}
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

func (s *Service) downloadFiles(jobs chan<- job, syncCtx *syncContext) {
	files := slices.Collect(maps.Values(syncCtx.fileMap))
	for _, file := range files {
		if file.MimeType != FolderMimeType {
			jobs <- job{file, syncCtx}
		}
	}
}

func (s *Service) downloadFile(f *drive.File, syncCtx *syncContext, syncResult *syncResult) error {
	res, err := s.drv.Files.Get(f.Id).Download()
	if err != nil {
		logging.LogDebugf("Could not download file '%s': %s", f.Name, err)
		return err
	}
	defer res.Body.Close()

	localPath := filepath.Join(s.cfg.DriveDir, fullPath(f.Id, syncCtx))
	logging.LogDebugf("Downloading %s to %s", f.Name, localPath)

	out, err := os.Create(localPath)
	if err != nil {
		logging.LogDebugf("Could not create file '%s': %s", localPath, err)
		return err
	}

	sha := crypto.SHA3_256.New()
	_, err = io.Copy(io.MultiWriter(out, sha), res.Body)
	if err != nil {
		logging.LogDebugf("Could not write file '%s': %s", localPath, err)
		return err
	}

	syncResult.parkFiles.Add(local.ParkFile{
		Path:        localPath,
		FileId:      f.Id,
		ContentHash: sha.Sum(nil),
	})

	return out.Close()
}

func (s *Service) isSetupAlready() bool {
	return s.cfg.IsSetup &&
		s.cfg.IsInitialized &&
		s.cfg.DriveDir != ""
}
