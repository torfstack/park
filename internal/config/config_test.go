package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name    string
		want    func(*testing.T)
		wantErr bool
	}{
		{
			name: "config file initially does not exist",
			want: func(t *testing.T) {
				_, err := os.Open(configFilePath)
				require.ErrorIs(t, err, os.ErrNotExist)
			},
		},
		{
			name: "non-interactive; creates config file",
			want: func(t *testing.T) {
				_, err := Get(false)
				require.NoError(t, err)

				_, err = os.Stat(configFilePath)
				require.NoError(t, err)
			},
		},
		{
			name: "non-interactive; config does not exist",
			want: func(t *testing.T) {
				cfg, err := Get(false)
				require.NoError(t, err)
				require.Equal(t, defaultDriveDir, cfg.LocalDir)
			},
		},
		{
			name: "non-interactive; config exists",
			want: func(t *testing.T) {
				path := t.TempDir()
				require.NoError(t, (&Config{LocalDir: path}).persist())

				cfg, err := Get(false)
				require.NoError(t, err)
				require.Equal(t, path, cfg.LocalDir)
			},
		},
		{
			name: "interactive; creates config file",
			want: func(t *testing.T) {
				inputFile = fileWithTextContent(t, "some/path")
				_, err := Get(false)
				require.NoError(t, err)

				_, err = os.Stat(configFilePath)
				require.NoError(t, err)
			},
		},
		{
			name: "interactive; config does not exist",
			want: func(t *testing.T) {
				inputFile = fileWithTextContent(t, "some/path")
				cfg, err := Get(true)
				require.NoError(t, err)
				require.Equal(t, "some/path", cfg.LocalDir)
			},
		},
		{
			name: "interactive; config does exist",
			want: func(t *testing.T) {
				path := t.TempDir()
				require.NoError(t, (&Config{LocalDir: path}).persist())

				cfg, err := Get(true)
				require.NoError(t, err)
				require.Equal(t, path, cfg.LocalDir)
			},
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				tempDirSetup(t)
				tt.want(t)
			},
		)
	}
}

func tempDirSetup(t *testing.T) {
	tempDir := t.TempDir()
	configFilePath = filepath.Join(tempDir, "config.yaml")
}

func fileWithTextContent(t *testing.T, text string) *os.File {
	tempDir := t.TempDir()
	f, err := os.Create(filepath.Join(tempDir, "file.txt"))
	require.NoError(t, err)
	_, err = f.WriteString(text)
	require.NoError(t, err)

	ff, _ := os.Open(f.Name())
	return ff
}
