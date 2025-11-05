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

	cfg, err := config.LoadConfig()
	if err != nil {
		logging.Logf("Could not load config: %s", err)
		os.Exit(1)
	}
	srv, err := service.NewService(context.Background(), cfg)
	if err != nil {
		logging.Logf("Could not create service: %s", err)
		os.Exit(1)
	}

	var setupCmd = &cobra.Command{
		Use:   "setup",
		Short: "Setup and perform initial sync",
		PreRun: func(cmd *cobra.Command, args []string) {
			logging.LogLevelDebug = debug
		},
		Run: func(cmd *cobra.Command, args []string) {
			err = srv.SetupAndInitialSync(cmd.Context())
			if err != nil {
				logging.LogError("main: error while running setup cmd", err)
				os.Exit(1)
			}
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
				logging.Log("Please run 'park setup' first")
				os.Exit(1)
			}
			err = srv.CheckForChanges(cmd.Context())
			if err != nil {
				logging.LogError("main: error while running check cmd", err)
				os.Exit(1)
			}
		},
	}

	var daemonCmd = &cobra.Command{
		Use:   "daemon",
		Short: "Run in daemon mode (watch for changes and sync)",
		PreRun: func(cmd *cobra.Command, args []string) {
			logging.LogLevelDebug = debug
		},
		Run: func(cmd *cobra.Command, args []string) {
			if !cfg.IsInitialized {
				logging.Log("Please run 'park setup' first")
				os.Exit(1)
			}
			err = srv.RunDaemon(cmd.Context())
			if err != nil {
				logging.LogError("main: error while running daemon cmd", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.AddCommand(setupCmd, checkCmd, daemonCmd)

	if err = rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
