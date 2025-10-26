package service

import (
	"context"
	"os"
	"path/filepath"

	"github.com/torfstack/park/internal/logging"
	"github.com/torfstack/park/internal/util"
)

func (s *Service) CheckForChanges(ctx context.Context) {
	pageToken := s.pageToken()
	if pageToken == "" {
		token, err := s.drv.Changes.GetStartPageToken().Do()
		if err != nil {
			panic(err)
		}
		pageToken = token.StartPageToken
		s.savePageToken(pageToken)
	}
	changes, err := s.drv.Changes.List(pageToken).
		Fields("newStartPageToken, changes(changeType, fileId, file(trashed), removed)").
		Do()
	if err != nil {
		panic(err)
	}

	for _, change := range changes.Changes {
		if change.ChangeType == "drive" {
			logging.LogDebug("Ignoring drive change")
			continue
		}
		if change.Removed || change.File.Trashed {
			logging.LogDebugf("Removing file %s", change.FileId)
			// TODO: Remove file from drive
		} else {
			logging.LogDebugf("Updating/Creating file %s", change.FileId)
			// TODO: Download file
		}
	}

	// Persist page token only if all changes were processed successfully for now
	s.savePageToken(changes.NewStartPageToken)
}

func (s *Service) pageToken() string {
	pageTokenFile := filepath.Join(util.HomeDir(), ".config", "park", "pageToken.txt")
	b, err := os.ReadFile(pageTokenFile)
	if err != nil {
		return ""
	}
	return string(b)
}

func (s *Service) savePageToken(token string) string {
	pageTokenFile := filepath.Join(util.HomeDir(), ".config", "park", "pageToken.txt")
	err := os.WriteFile(pageTokenFile, []byte(token), 0644)
	if err != nil {
		panic(err)
	}
	return token
}
