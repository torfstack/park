package local

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/torfstack/park/internal/logging"
)

type Watcher struct {
	watcher  *fsnotify.Watcher
	Events   chan fsnotify.Event
	RootPath string
}

func NewWatcher(rootPath string) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		watcher:  watcher,
		Events:   make(chan fsnotify.Event),
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
		return fmt.Errorf("add-dir; could not stat directory: %w", err)
	}
	if !info.IsDir() {
		return nil
	}
	if err = w.watcher.Add(path); err != nil {
		return fmt.Errorf("add-dir; could not add directory to watcher: %w", err)
	}
	logging.Debugf("Added directory to watcher: %s", path)
	return nil
}

func (w *Watcher) Close() {
	close(w.Events)
	if err := w.watcher.Close(); err != nil {
		logging.Infof("Error closing watcher: %s", err)
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

			err = w.handle(event)
			if err != nil {
				return fmt.Errorf("run; could not handle event: %w", err)
			}

			w.Events <- event

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher error channel closed")
			}
			logging.Infof("FSNotify Error: %v", err)
		}
	}
}

func (w *Watcher) handle(event fsnotify.Event) error {
	switch {
	case event.Has(fsnotify.Create):
		return w.addDir(event.Name)
	case event.Has(fsnotify.Write):
		// Nothing to do yet
	case event.Has(fsnotify.Remove):
		// Nothing to do yet, fsnotify stops watching directories when they are removed
	case event.Has(fsnotify.Rename):
		// Nothing to do yet, fsnotify stops watching directories when they are removed
		// A rename is followed by a create event
	default:
		logging.Debugf("Ignoring event: %s", event)
	}
	return nil
}
