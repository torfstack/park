package remote

import (
	"context"
	"crypto"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/torfstack/park/internal/config"
	"github.com/torfstack/park/internal/local"
	"github.com/torfstack/park/internal/logging"
	"google.golang.org/api/drive/v3"
)

const (
	FolderMimeType   = "application/vnd.google-apps.folder"
	ShortcutMimeType = "application/vnd.google-apps.shortcut"
	RootFolderId     = "root"

	NumWorkers = 4
)

type job struct {
	file    *drive.File
	syncCtx *syncContext
}

func performInitialSync(ctx context.Context, drv *drive.Service, cfg config.Config) error {
	syncCtx := syncContext{
		fileMap: make(map[string]*drive.File),
		parents: make(map[string]string),
	}

	err := walkFolder(ctx, drv, RootFolderId, cfg.LocalDir, &syncCtx)
	if err != nil {
		return fmt.Errorf("error walking root folder: %w", err)
	}

	err = createDirs(cfg, &syncCtx)
	if err != nil {
		return fmt.Errorf("error creating initial directories: %w", err)
	}
	logging.Debug("Created initial directories")

	jobs := make(chan job)
	results := make(chan local.ParkFile)
	var wg sync.WaitGroup

	logging.Debug("Starting download workers")
	for i := 0; i < NumWorkers; i++ {
		wg.Go(
			func() {
				for j := range jobs {
					parkFile, errGo := downloadFile(drv, cfg, j.file, &syncCtx)
					if errGo != nil {
						logging.Debugf("Skipping: could not download file '%s': %s", j.file.Name, errGo)
						continue
					}
					results <- *parkFile
				}
			},
		)
	}
	go func() {
		enqueueDownloadJobs(jobs, &syncCtx)
		close(jobs)
		logging.Debug("Finished enqueueing download jobs")
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	parkFiles := make(map[string]local.ParkFile)
	for parkFile := range results {
		parkFiles[parkFile.FileId] = parkFile
	}
	parkTable := local.NewParkTable(cfg, parkFiles)
	err = parkTable.Persist()
	if err != nil {
		return fmt.Errorf("error persisting initial park table: %w", err)
	}

	logging.Debug("Downloads finished!")
	return nil
}

func walkFolder(ctx context.Context, drv *drive.Service, folderID, path string, syncCtx *syncContext) error {
	pageToken := ""
	for {
		req := drv.Files.List().
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
			if err = handleFile(ctx, drv, f, folderID, path, syncCtx); err != nil {
				logging.Errorf("error handling file %s: %s", f.Name, err)
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

func handleFile(
	ctx context.Context,
	drv *drive.Service,
	f *drive.File,
	folderId, path string,
	syncCtx *syncContext,
) error {
	if f.DriveId != "" {
		return nil
	}

	fullPath := filepath.Join(path, f.Name)

	switch f.MimeType {
	case FolderMimeType:
		if err := walkFolder(ctx, drv, f.Id, fullPath, syncCtx); err != nil {
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
			if err := walkFolder(ctx, drv, targetID, filepath.Join(path, f.Name), syncCtx); err != nil {
				return fmt.Errorf("error walking shortcut folder %s: %w", f.Name, err)
			}
			syncCtx.parents[targetID] = folderId
			syncCtx.fileMap[targetID] = f
		} else {
			shortcutFile, err := drv.Files.Get(targetID).Fields("id, name").Do()
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

func createDirs(cfg config.Config, syncCtx *syncContext) error {
	files := slices.Collect(maps.Values(syncCtx.fileMap))
	for _, f := range files {
		if f.MimeType == FolderMimeType {
			path := filepath.Join(cfg.LocalDir, localPath(f, syncCtx))
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

func enqueueDownloadJobs(jobs chan<- job, syncCtx *syncContext) {
	files := slices.Collect(maps.Values(syncCtx.fileMap))
	for _, file := range files {
		if file.MimeType != FolderMimeType {
			jobs <- job{file, syncCtx}
		}
	}
}

func downloadFile(drv *drive.Service, cfg config.Config, f *drive.File, syncCtx *syncContext) (*local.ParkFile, error) {
	res, err := drv.Files.Get(f.Id).Download()
	if err != nil {
		return nil, fmt.Errorf("could not download file '%s': %w", f.Name, err)
	}
	defer func(Body io.ReadCloser) {
		if err = Body.Close(); err != nil {
			logging.Debugf("Could not close body: %s", err)
		}
	}(res.Body)

	absoluteLocalPath := filepath.Join(cfg.LocalDir, localPath(f, syncCtx))
	logging.Debugf("Downloading %s to %s", f.Name, absoluteLocalPath)

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
