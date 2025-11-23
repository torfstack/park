package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/torfstack/park/internal/logging"
	"golang.org/x/oauth2"
)

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("could not open token file: %w", err)
	}
	defer func(f *os.File) {
		err = f.Close()
		if err != nil {
			logging.Infof("Could not close token file: %s", err)
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
			logging.Infof("Could not close token file after writing: %s", err)
		}
	}(f)
	if err = json.NewEncoder(f).Encode(token); err != nil {
		return fmt.Errorf("could not encode token and write to disk: %w", err)
	}
	return nil
}

func getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	codeCh, port, err := startOAuthCallbackServer(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not start OAuth callback server: %w", err)
	}

	redirectURL := fmt.Sprintf("http://localhost:%d", port)
	config.RedirectURL = redirectURL
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	logging.Infof("Trying to open your browser to visit the URL to authorize this application: %s", authURL)
	openBrowser(authURL)

	code, err := waitForAuthCode(ctx, codeCh)
	if err != nil {
		return nil, fmt.Errorf("could not complete OAuth flow: %w", err)
	}

	tok, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("could not exchange auth code: %w", err)
	}

	logging.Info("Login successful!")
	return tok, nil
}

func startOAuthCallbackServer(ctx context.Context) (<-chan string, int, error) {
	codeCh := make(chan string)

	// Use port 0 to let the OS assign a free port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return codeCh, 0, fmt.Errorf("could not create listener: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer close(codeCh)
		if errMsg := r.FormValue("error"); errMsg != "" {
			http.Error(w, errMsg, http.StatusBadRequest)
			codeCh <- errMsg
			return
		}
		code := r.FormValue("code")
		w.WriteHeader(http.StatusOK)
		_, err = fmt.Fprintln(w, "Authorization successful! You can close this window now.")
		if err != nil {
			logging.Infof("Could not write response to client: %s", err)
			return
		}
		codeCh <- code
		// Shut down server after handling
		go func() { _ = srv.Shutdown(ctx) }()
	})

	go func() {
		if err = srv.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			close(codeCh)
			log.Fatalf("Serve: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	return codeCh, port, nil
}

func waitForAuthCode(ctx context.Context, codeCh <-chan string) (string, error) {
	logging.Info("Waiting for successful login... ")
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case code := <-codeCh:
		if code == "" {
			return "", fmt.Errorf("no authorization code received")
		}
		return code, nil
	case <-time.After(10 * time.Minute):
		return "", fmt.Errorf("timed out waiting for authorization code")
	}
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
