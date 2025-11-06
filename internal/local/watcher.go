package local

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/torfstack/park/internal/logging"
)

type WatchEvent struct {
	Path string
	Op   fsnotify.Op
}

type Watcher struct {
	watcher  *fsnotify.Watcher
	Events   chan WatchEvent
	RootPath string
}

func NewWatcher(rootPath string) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		watcher:  watcher,
		Events:   make(chan WatchEvent),
		RootPath: rootPath,
	}

	// NOTE: fsnotify does not recursively watch subdirectories
	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if err = w.addDir(path); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return w, nil
}

func (w *Watcher) addDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("add-dir: could not stat directory: %w", err)
	}
	if !info.IsDir() {
		return nil
	}

	if err = w.watcher.Add(path); err != nil {
		return fmt.Errorf("add-dir: could not add directory to watcher: %w", err)
	}
	logging.LogDebugf("Added directory to watcher: %s", path)
	return nil
}

func (w *Watcher) Close() {
	close(w.Events)
	if err := w.watcher.Close(); err != nil {
		logging.Logf("Error closing watcher: %s", err)
	}
}

func (w *Watcher) Run(_ context.Context) error {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}

			relativePath, err := filepath.Rel(w.RootPath, event.Name)
			if err != nil || relativePath == ".." || event.Name == w.RootPath {
				continue
			}

			if event.Op&fsnotify.Create == fsnotify.Create {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					err = w.addDir(event.Name)
					if err != nil {
						return fmt.Errorf("add-dir: could not add directory to watcher: %w", err)
					}
				}
			}

			w.Events <- WatchEvent{Path: event.Name, Op: event.Op}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher error channel closed")
			}
			logging.Logf("FSNotify Error: %v", err)
		}
	}
}
