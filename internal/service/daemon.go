package service

import (
	"context"
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/torfstack/park/internal/config"
	"github.com/torfstack/park/internal/local"
	"github.com/torfstack/park/internal/logging"
)

func RunDaemon(ctx context.Context, cfg config.Config) error {
	w, err := local.NewWatcher(cfg.LocalDir)
	if err != nil {
		return fmt.Errorf("run-daemon: could not create watcher: %w", err)
	}
	defer w.Close()

	go consumeWatcherEvents(w.Events)
	err = w.Run(ctx)
	if err != nil {
		return fmt.Errorf("run-daemon: error while running watcher: %w", err)
	}
	return nil
}

func consumeWatcherEvents(c <-chan fsnotify.Event) {
	for event := range c {
		switch {
		case event.Has(fsnotify.Create):
			logging.Debugf("Received create event: %s", event)
		case event.Has(fsnotify.Write):
			logging.Debugf("Received write event: %s", event)
		case event.Has(fsnotify.Remove):
			logging.Debugf("Received remove event: %s", event)
		}
	}
}
