package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/torfstack/park/internal/config"
	"github.com/torfstack/park/internal/logging"
	"github.com/torfstack/park/internal/service"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "park",
		Short: "Google Drive file synchronization tool",
	}

	var debug bool
	rootCmd.PersistentFlags().
		BoolVarP(&debug, "debug", "d", false, "Enable debug output")

	cfg := config.LoadConfig()
	srv := service.NewService(cfg)

	var setupCmd = &cobra.Command{
		Use:   "setup",
		Short: "Setup and perform initial sync",
		PreRun: func(cmd *cobra.Command, args []string) {
			logging.LogLevelDebug = debug
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return srv.SetupAndInitialSync()
		},
	}

	var checkCmd = &cobra.Command{
		Use:   "check",
		Short: "Check for changes",
		PreRun: func(cmd *cobra.Command, args []string) {
			logging.LogLevelDebug = debug
		},
		Run: func(cmd *cobra.Command, args []string) {
			if !cfg.IsInitialized {
				fmt.Println("Please run 'park setup' first")
				os.Exit(1)
			}
			srv.CheckForChanges(context.Background())
		},
	}

	rootCmd.AddCommand(setupCmd, checkCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
