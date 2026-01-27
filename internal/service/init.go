package service

import (
	"context"
	"fmt"

	"github.com/torfstack/park/internal/config"
	"github.com/torfstack/park/internal/db"
	"github.com/torfstack/park/internal/db/sqlc"
	"google.golang.org/api/drive/v3"
)

func InitialSync(ctx context.Context, cfg config.Config, drv *drive.Service) error {
	pageToken, err := initialPageToken(drv)
	if err != nil {
		return fmt.Errorf("could not get initial page token: %w", err)
	}

	err = performInitialSync(ctx, drv, cfg)
	if err != nil {
		return fmt.Errorf("could not perform initial sync: %w", err)
	}

	err = persistPageToken(ctx, pageToken)
	if err != nil {
		return fmt.Errorf("could not persist page token: %w", err)
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

func persistPageToken(ctx context.Context, pageToken string) error {
	d, err := db.New(ctx)
	if err != nil {
		return fmt.Errorf("could not create database: %w", err)
	}
	defer d.Close()
	err = d.Queries().UpsertPageToken(ctx, sqlc.UpsertPageTokenParams{
		ID:               1,
		CurrentPageToken: pageToken,
	})
	if err != nil {
		return fmt.Errorf("could not run upsert page token query: %w", err)
	}
	return nil
}
