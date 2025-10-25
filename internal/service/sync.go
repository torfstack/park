package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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
	}
	changes, err := s.drv.Changes.List(pageToken).Fields("nextPageToken", "changes(fileId)").Do()
	if err != nil {
		panic(err)
	}
	fmt.Println("Changes:")
	for _, change := range changes.Changes {
		fmt.Printf("%s\n", change.FileId)
	}
	s.savePageToken(changes.NewStartPageToken)
}

func (s *Service) ListFiles() {
	q := "trashed = false"
	files, err := s.drv.Files.List().Q(q).Fields("files(id, name)").Do()
	if err != nil {
		panic(err)
	}
	for _, file := range files.Files {
		fmt.Printf("%s:%s\n", file.Id, file.Name)
	}
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
