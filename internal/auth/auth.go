package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/torfstack/park/internal/logging"
	"github.com/torfstack/park/internal/util"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func DriveService(ctx context.Context) (*drive.Service, error) {
	credPath := filepath.Join(util.HomeDir(), ".config", "park", "credentials.json")
	b, err := os.ReadFile(credPath)
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
	return drive.NewService(ctx, option.WithHTTPClient(client))
}

func getClient(ctx context.Context, config *oauth2.Config) (*http.Client, error) {
	tokFile := filepath.Join(util.HomeDir(), ".config", "park", "token.json")
	tok, err := tokenFromFile(tokFile)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		tok, err = getTokenFromWeb(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("could not get token from web: %w", err)
		}
		err = saveToken(tokFile, tok)
		if err != nil {
			return nil, fmt.Errorf("could not save token: %w", err)
		}
	case err != nil:
		return nil, fmt.Errorf("could not read token file: %w", err)
	}
	return config.Client(ctx, tok), nil
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("could not open token file: %w", err)
	}
	defer func(f *os.File) {
		err = f.Close()
		if err != nil {
			logging.Logf("Could not close token file: %s", err)
		}
	}(f)
	var tok oauth2.Token
	err = json.NewDecoder(f).Decode(&tok)
	if err != nil {
		return nil, fmt.Errorf("could not decode token file: %w", err)
	}
	return &tok, nil
}

func saveToken(path string, token *oauth2.Token) error {
	fmt.Printf("Saving token to: %s\n", path)
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer func(f *os.File) {
		err = f.Close()
		if err != nil {
			logging.Logf("Could not close token file after writing: %s", err)
		}
	}(f)
	if err = json.NewEncoder(f).Encode(token); err != nil {
		return fmt.Errorf("could not encode token and write to disk: %w", err)
	}
	return nil
}

func getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Visit this URL in your browser, then paste the code here:\n%v\n", authURL)

	var authCode string
	fmt.Print("Enter the code: ")
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("could not read auth code: %w", err)
	}

	tok, err := config.Exchange(ctx, authCode)
	if err != nil {
		return nil, fmt.Errorf("could not exchange auth code: %w", err)
	}
	return tok, nil
}
