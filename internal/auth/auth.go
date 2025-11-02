package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/torfstack/park/internal/logging"
	"github.com/torfstack/park/internal/util"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

var (
	errTokenNoLongerValid = errors.New("token on disk is no longer valid")
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

	drv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("could not create drive service: %w", err)
	}
	return drv, nil
}

func getClient(ctx context.Context, config *oauth2.Config) (*http.Client, error) {
	tokFile := filepath.Join(util.HomeDir(), ".config", "park", "token.json")
	tok, err := tokenFromFile(tokFile)
	switch {
	case errors.Is(err, fs.ErrNotExist) || errors.Is(err, errTokenNoLongerValid):
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
	if !tok.Valid() {
		return nil, errTokenNoLongerValid
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
	localPort := 8080
	redirectURL := fmt.Sprintf("http://localhost:%d", localPort)
	config.RedirectURL = redirectURL

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	logging.Logf("Trying to open your browser to visit the URL to authorize this application: %s", authURL)
	openBrowser(authURL)

	codeCh := make(chan string)
	srv := &http.Server{Addr: fmt.Sprintf(":%d", localPort)}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if errMsg := r.FormValue("error"); errMsg != "" {
			http.Error(w, errMsg, http.StatusBadRequest)
			codeCh <- ""
			return
		}
		code := r.FormValue("code")
		_, err := fmt.Fprintln(w, "Authorization successful! You can close this tab.")
		if err != nil {
			logging.Logf("Could not write response to client: %s", err)
			return
		}

		codeCh <- code

		// Shut down server after handling
		go func() { _ = srv.Shutdown(ctx) }()
	})

	go func() {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	logging.Log("Waiting for successful login... ")
	code := <-codeCh
	if code == "" {
		log.Fatal("No code received")
	}

	tok, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("could not exchange auth code: %w", err)
	}
	logging.Log("Login successful!")
	return tok, nil
}

func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		fmt.Printf("Please open the following URL manually: %s\n", url)
	}

	if err != nil {
		fmt.Printf("Failed to open browser automatically: %v\n", err)
		fmt.Printf("Visit this URL manually: %s\n", url)
	}
}
