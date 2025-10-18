package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/torfstack/park/internal/util"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func GetClient(config *oauth2.Config) *http.Client {
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

func DriveService() (*drive.Service, error) {
	credPath := filepath.Join(util.HomeDir(), ".config", "park", "credentials.json")
	b, err := os.ReadFile(credPath)
	if err != nil {
		return nil, err
	}

	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		return nil, err
	}

	client := GetClient(config)
	return drive.NewService(context.Background(), option.WithHTTPClient(client))
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var tok oauth2.Token
	err = json.NewDecoder(f).Decode(&tok)
	return &tok, err
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path) // TODO: persist to sqlite
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline) // TODO: randomize state-token?
	fmt.Printf("Visit this URL in your browser, then paste the code here:\n%v\n", authURL)

	var authCode string
	fmt.Print("Enter the code: ")
	if _, err := fmt.Scan(&authCode); err != nil {
		panic(err)
	}

	tok, err := config.Exchange(context.TODO(), authCode) // TODO: context
	if err != nil {
		panic(err)
	}
	return tok
}
