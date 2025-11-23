package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestSaveAndRetrieveToken(t *testing.T) {
	tests := []struct {
		name string
		do   func(*testing.T)
	}{
		{
			name: "save token creates file",
			do: func(t *testing.T) {
				token := &oauth2.Token{
					AccessToken:  "valid_access_token",
					RefreshToken: "valid_refresh_token",
				}
				tempDir := t.TempDir()
				tokenFile := filepath.Join(tempDir, "token.json")
				err := saveToken(tokenFile, token)
				require.NoError(t, err)

				_, err = os.Stat(tokenFile)
				require.NoError(t, err)
			},
		},
		{
			name: "saved token can be retrieved",
			do: func(t *testing.T) {
				token := &oauth2.Token{
					AccessToken:  "valid_access_token",
					RefreshToken: "valid_refresh_token",
				}
				tempDir := t.TempDir()
				tokenFile := filepath.Join(tempDir, "token.json")
				err := saveToken(tokenFile, token)
				require.NoError(t, err)

				retrievedToken, err := tokenFromFile(tokenFile)
				require.NoError(t, err)
				require.Equal(t, token, retrievedToken)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.do(t)
		})
	}
}

func TestStartOAuthCallbackServer(t *testing.T) {
	tests := []struct {
		name string
		do   func(t *testing.T, port int, c <-chan string, cancel context.CancelFunc)
	}{
		{
			name: "successful callback",
			do: func(t *testing.T, port int, c <-chan string, _ context.CancelFunc) {
				client := &http.Client{Timeout: time.Second * 1}
				code := "auth_code"

				go func() {
					select {
					case receivedCode := <-c:
						require.Equal(t, code, receivedCode)
					case <-time.After(time.Second * 1):
						require.Fail(t, "timeout waiting for code")
					}
				}()

				req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d?code=%s", port, code), nil)
				require.NoError(t, err)

				res, err := client.Do(req)
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, res.StatusCode)

			},
		},
		{
			name: "error callback",
			do: func(t *testing.T, port int, c <-chan string, _ context.CancelFunc) {
				client := &http.Client{Timeout: time.Second * 1}
				e := "error_description"

				go func() {
					select {
					case receivedCode := <-c:
						require.Equal(t, e, receivedCode)
					case <-time.After(time.Second * 1):
						require.Fail(t, "timeout waiting for code")
					}
				}()

				req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d?error=%s", port, e), nil)
				require.NoError(t, err)

				res, err := client.Do(req)
				require.NoError(t, err)
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
			},
		},
		{
			name: "context canceled",
			do: func(t *testing.T, _ int, c <-chan string, cancel context.CancelFunc) {
				go func() {
					select {
					case receivedCode := <-c:
						require.Empty(t, receivedCode)
					case <-time.After(time.Second * 1):
						require.Fail(t, "timeout waiting for code")
					}
				}()

				cancel()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())

			codeCh, port, err := startOAuthCallbackServer(ctx)
			if err != nil {
				t.Fatalf("error starting OAuth callback server: %v", err)
			}

			if port == 0 {
				t.Fatalf("expected a valid port, got: %d", port)
			}

			tt.do(t, port, codeCh, cancel)

			go func() {
				time.Sleep(5 * time.Second)
				cancel()
			}()
		})
	}
}
