package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/torfstack/park/internal/local"
	"github.com/torfstack/park/internal/logging"
	"github.com/torfstack/park/internal/util"
)

func (s *Service) CheckForChanges(_ context.Context) error {
	pageToken := s.getPageTokenFromFile()
	var err error
	if pageToken == "" {
		pageToken, err = s.getInitialPageToken()
		if err != nil {
			return fmt.Errorf("could not get initial page token from drive api: %w", err)
		}
	}
	changes, err := s.drv.Changes.List(pageToken).
		Fields("newStartPageToken, changes(changeType, fileId, file(trashed, name, mimeType), removed)").
		Do()
	if err != nil {
		return fmt.Errorf("could not get changes from drive api: %w", err)
	}

	if len(changes.Changes) == 0 {
		logging.Log("No changes found")
		return nil
	}

	parkTable, err := local.LoadParkTable(s.cfg)
	if err != nil {
		return fmt.Errorf("could not load park table: %w", err)
	}

	for _, change := range changes.Changes {
		if change.ChangeType == "drive" {
			logging.LogDebug("Ignoring change with changeType 'drive'")
			continue
		}
		// TODO: handle folder mime type
		if change.File.MimeType == FolderMimeType {
			logging.LogDebugf("Ignoring folder %s", change.File.Name)
			continue
		}
		if change.Removed || change.File.Trashed {
			logging.LogDebugf("Removing file %s with name %s", change.FileId, change.File.Name)
			err = parkTable.Remove(change.FileId)
		} else {
			var res *http.Response
			res, err = s.drv.Files.Get(change.FileId).Download()
			if err != nil {
				return fmt.Errorf("could not download file with name '%s': %w", change.File.Name, err)
			}
			if parkTable.Exists(change.FileId) {
				logging.LogDebugf("Updating file %s with name %s", change.FileId, change.File.Name)
				err = parkTable.Update(change.FileId, res.Body)
			} else {
				logging.LogDebugf("Creating file %s with name %s", change.FileId, change.File.Name)
				err = parkTable.Create(change.FileId, change.File.Name, res.Body)
			}
		}
		if err != nil {
			return fmt.Errorf("could not do remove/update/create operation: %w", err)
		}
	}

	// Persist page token only if all changes were processed successfully for now
	err = s.persistPageToken(changes.NewStartPageToken)
	if err != nil {
		return fmt.Errorf("could not persist page token: %w", err)
	}
	return nil
}

func (s *Service) getPageTokenFromFile() string {
	pageTokenFile := filepath.Join(util.HomeDir(), ".config", "park", "pageToken.txt")
	b, err := os.ReadFile(pageTokenFile)
	if err != nil {
		return ""
	}
	return string(b)
}

func (s *Service) persistPageToken(token string) error {
	pageTokenFile := filepath.Join(util.HomeDir(), ".config", "park", "pageToken.txt")
	err := os.WriteFile(pageTokenFile, []byte(token), 0644)
	if err != nil {
		return fmt.Errorf("could not write page token to file %s: %w", pageTokenFile, err)
	}
	return nil
}

func (s *Service) getInitialPageToken() (string, error) {
	token, err := s.drv.Changes.GetStartPageToken().Do()
	if err != nil {
		return "", fmt.Errorf("could not get start page token: %w", err)
	}
	pageToken := token.StartPageToken
	err = s.persistPageToken(pageToken)
	if err != nil {
		return "", fmt.Errorf("could not persist page token: %w", err)
	}
	return pageToken, nil
}
