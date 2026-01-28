package service

import (
	"context"
	"fmt"
	"os"

	"github.com/torfstack/park/internal/config"
	"github.com/torfstack/park/internal/db"
	"github.com/torfstack/park/internal/db/sqlc"
	"github.com/torfstack/park/internal/logging"
	"github.com/torfstack/park/internal/util"
	"google.golang.org/api/drive/v3"
)

func InitialSync(ctx context.Context, cfg config.Config, drv *drive.Service) error {
	d, err := db.New(ctx)
	if err != nil {
		return fmt.Errorf("could not create database: %w", err)
	}
	defer d.Close()

	isInitialized, err := d.Queries().IsInitialized(ctx)
	if err != nil {
		return fmt.Errorf("could not check if initialized: %w", err)
	}
	if isInitialized {
		logging.Info("Already initialized!")
		return nil
	}

	pageToken, err := initialPageToken(drv)
	if err != nil {
		return fmt.Errorf("could not get initial page token: %w", err)
	}

	tempDir, err := util.CreateTempDir()
	if err != nil {
		return fmt.Errorf("could not create temp dir: %w", err)
	}
	defer os.Remove(tempDir)

	err = d.WithTransaction(ctx, func(q *sqlc.Queries) error {
		err = performInitialSync(ctx, q, drv, tempDir)
		if err != nil {
			return fmt.Errorf("could not perform initial sync: %w", err)
		}

		err = os.Rename(tempDir, cfg.LocalDir)
		if err != nil {
			return fmt.Errorf("could not rename temp dir: %w", err)
		}

		err = persistPageToken(ctx, q, pageToken)
		if err != nil {
			return fmt.Errorf("could not persist page token: %w", err)
		}

		err = q.SetInitialized(ctx)
		if err != nil {
			return fmt.Errorf("could not set initialized flag: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

func initialPageToken(drv *drive.Service) (string, error) {
	initialToken, err := drv.Changes.GetStartPageToken().Do()
	if err != nil {
		return "", fmt.Errorf("initialPageToken; could not get initial page token: %w", err)
	}
	if initialToken.StartPageToken == "" {
		return "", fmt.Errorf("initialPageToken; empty page token")
	}
	return initialToken.StartPageToken, nil
}

func persistPageToken(ctx context.Context, q *sqlc.Queries, pageToken string) error {
	err := q.UpdatePageToken(ctx, pageToken)
	if err != nil {
		return fmt.Errorf("could not run upsert page token query: %w", err)
	}
	return nil
}
