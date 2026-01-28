package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/torfstack/park/internal/db"
	"github.com/torfstack/park/internal/util"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

var (
	credentialsFilePath = filepath.Join(util.ParkConfigDir, "credentials.json")
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
	d, err := db.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not connect to google drive: %w", err)
	}
	tokenString, err := d.Queries().GetAuthToken(ctx)
	var tok *oauth2.Token
	switch {
	case tokenString == "":
		tok, err = getTokenFromWeb(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("could not get token from web: %w", err)
		}
		tokenString, err = serializeToken(tok)
		if err != nil {
			return nil, fmt.Errorf("could not serialize token: %w", err)
		}
		err = d.Queries().UpdateAuthToken(ctx, tokenString)
		if err != nil {
			return nil, fmt.Errorf("could not save token: %w", err)
		}
	case err != nil:
		return nil, fmt.Errorf("could not read token file: %w", err)
	}

	tok, err = parseToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("could not parse token: %w", err)
	}
	return config.Client(ctx, tok), nil
}

func parseToken(tokenString string) (*oauth2.Token, error) {
	var tok oauth2.Token
	err := json.NewDecoder(strings.NewReader(tokenString)).Decode(&tok)
	if err != nil {
		return nil, fmt.Errorf("could not decode token file: %w", err)
	}
	return &tok, nil
}

func serializeToken(token *oauth2.Token) (string, error) {
	t, err := json.Marshal(token)
	if err != nil {
		return "", fmt.Errorf("could not serialize token: %w", err)
	}
	return string(t), nil
}
