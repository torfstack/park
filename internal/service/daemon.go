package service

import (
	"context"
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/torfstack/park/internal/local"
	"github.com/torfstack/park/internal/logging"
)

func (s *Service) RunDaemon(ctx context.Context) error {
	w, err := local.NewWatcher(s.cfg.DriveDir)
	if err != nil {
		return fmt.Errorf("run-daemon: could not create watcher: %w", err)
	}
	defer w.Close()

	go s.consumeWatcherEvents(w.Events)
	err = w.Run(ctx)
	if err != nil {
		return fmt.Errorf("run-daemon: error while running watcher: %w", err)
	}
	return nil
}

func (s *Service) consumeWatcherEvents(c <-chan local.WatchEvent) {
	for {
		select {
		case event := <-c:
			switch {
			case event.Op&fsnotify.Create == fsnotify.Create:
				logging.LogDebugf("Received create event: %s", event)
			case event.Op&fsnotify.Write == fsnotify.Write:
				logging.LogDebugf("Received write event: %s", event)
			case event.Op&fsnotify.Remove == fsnotify.Remove:
				logging.LogDebugf("Received remove event: %s", event)
			}
		}
	}
}
