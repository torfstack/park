package main

import (
	"context"
	"fmt"
	"os"

	"github.com/torfstack/park/internal/config"
	"github.com/torfstack/park/internal/sync"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: park [login|list|download <fileId>]")
		return
	}

	cfg := config.Config{DriveDir: "/home/david/TestGoogleDrive"}
	srv := sync.NewService(cfg)
	cmd := os.Args[1]

	switch cmd {
	case "login":
		fmt.Println("✔️  Authentication successful.")
	case "list":
		srv.CheckForChanges(context.Background())
	case "download":
		if len(os.Args) < 4 {
			fmt.Println("Usage: park download <fileId>")
			return
		}
		srv.DownloadFile(os.Args[2])
	default:
		fmt.Println("Unknown command:", cmd)
	}
}
