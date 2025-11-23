package remote

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/torfstack/park/internal/config"
	"github.com/torfstack/park/internal/logging"
	"github.com/torfstack/park/internal/util"
	"google.golang.org/api/drive/v3"
)

var (
	pageTokenFile = filepath.Join(util.ParkConfigDir, "page_token")
)

func Initialize(ctx context.Context, cfg config.Config, drv *drive.Service) error {
	if cfg.RemoteInitialized {
		logging.Debug("Remote already initialized!")
		return nil
	}

	pageToken, err := initialPageToken(drv)
	if err != nil {
		return fmt.Errorf("could not get initial page token: %w", err)
	}

	err = performInitialSync(ctx, drv, cfg)
	if err != nil {
		return fmt.Errorf("could not perform initial sync: %w", err)
	}

	err = persistPageToken(pageToken)
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

func getPageToken() (string, error) {
	pageToken, err := os.ReadFile(pageTokenFile)
	if err != nil {
		return "", fmt.Errorf("could not read page token from file %s: %w", pageTokenFile, err)
	}
	if len(pageToken) == 0 {
		return "", fmt.Errorf("empty page token")
	}
	return strings.TrimSuffix(string(pageToken), "\n"), nil
}

func persistPageToken(pageToken string) error {
	err := util.WriteFile(pageTokenFile, []byte(pageToken))
	if err != nil {
		return fmt.Errorf("could not write page token to file %s: %w", pageTokenFile, err)
	}
	return nil
}
