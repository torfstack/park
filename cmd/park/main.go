package main

import (
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

	var configCmd = &cobra.Command{
		Use:   "config",
		Short: "Setup config",
		PreRun: func(cmd *cobra.Command, args []string) {
			logging.LogLevelDebug = debug
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.Get(true)
			if err != nil {
				return fmt.Errorf("main; error while running setup cmd: %w", err)
			}
			return nil
		},
	}

	var daemonCmd = &cobra.Command{
		Use:   "daemon",
		Short: "Run in daemon mode (watch for changes and sync)",
		PreRun: func(cmd *cobra.Command, args []string) {
			logging.LogLevelDebug = debug
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(false)
			err = service.RunDaemon(cmd.Context(), cfg)
			if err != nil {
				return fmt.Errorf("main; error while running daemon cmd: %w", err)
			}
			return nil
		},
	}

	rootCmd.AddCommand(configCmd, daemonCmd)

	if err := rootCmd.Execute(); err != nil {
		logging.LogError("ERROR: %s", err)
		os.Exit(1)
	}
}
