package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

var (
	level = new(slog.LevelVar)

	logger = slog.New(
		slog.NewTextHandler(
			os.Stdout, &slog.HandlerOptions{
				Level: level,
			},
		),
	)
)

func SetDebug(enable bool) {
	if enable {
		level.Set(slog.LevelDebug)
	} else {
		level.Set(slog.LevelInfo)
	}
}

func Info(msg string) {
	logger.Info(msg)
}

func Infof(msg string, args ...any) {
	logger.Info(fmt.Sprintf(msg, args...))
}

func Error(msg string, err error) {
	logger.Error(msg, "error", err)
}

func Fatalf(format string, v ...any) {
	logger.Error(fmt.Sprintf(format, v...))
	os.Exit(1)
}

func Debug(msg string) {
	logger.Debug(msg)
}

func Debugf(msg string, args ...any) {
	if !logger.Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	logger.Debug(fmt.Sprintf(msg, args...))
}
