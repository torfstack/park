package auth

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/torfstack/park/internal/util"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

var (
	credentialsFilePath   = filepath.Join(util.ParkConfigDir, "credentials.json")
	tokenFilePath         = filepath.Join(util.ParkConfigDir, "token.json")
	errTokenNoLongerValid = errors.New("token on disk is no longer valid")
)

func DriveService(ctx context.Context) (*drive.Service, error) {
	b, err := os.ReadFile(credentialsFilePath)
	if err != nil {
		return nil, fmt.Errorf("could not read google credentials: %w", err)
	}

	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		return nil, fmt.Errorf("could not parse google config: %w", err)
	}

	client, err := getClient(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("could not get client for drive service: %w", err)
	}

	drv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("could not create drive service: %w", err)
	}
	return drv, nil
}

func getClient(ctx context.Context, config *oauth2.Config) (*http.Client, error) {
	tok, err := tokenFromFile(tokenFilePath)
	switch {
	case errors.Is(err, fs.ErrNotExist) || errors.Is(err, errTokenNoLongerValid):
		tok, err = getTokenFromWeb(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("could not get token from web: %w", err)
		}
		err = saveToken(tokenFilePath, tok)
		if err != nil {
			return nil, fmt.Errorf("could not save token: %w", err)
		}
	case err != nil:
		return nil, fmt.Errorf("could not read token file: %w", err)
	}
	return config.Client(ctx, tok), nil
}
