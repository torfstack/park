package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/torfstack/park/internal/auth"
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

	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Setup config",
		PreRun: func(cmd *cobra.Command, args []string) {
			logging.SetDebug(debug)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.GetInteractive(cmd.Context())
			if err != nil {
				return fmt.Errorf("main; error while running setup cmd: %w", err)
			}
			return nil
		},
	}

	daemonCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run in daemon mode (watch for changes and sync)",
		PreRun: func(cmd *cobra.Command, args []string) {
			logging.SetDebug(debug)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd.Context())
			if err != nil {
				return fmt.Errorf("main; error while getting config: %w", err)
			}
			err = service.RunDaemon(cmd.Context(), cfg)
			if err != nil {
				return fmt.Errorf("main; error while running daemon cmd: %w", err)
			}
			return nil
		},
	}

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "InitialSync config",
		PreRun: func(cmd *cobra.Command, args []string) {
			logging.SetDebug(debug)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.GetInteractive(cmd.Context())
			if err != nil {
				return fmt.Errorf("main; error while running init cmd: %w", err)
			}
			drv, err := auth.DriveService(cmd.Context())
			if err != nil {
				return fmt.Errorf("main; error while getting drive service: %w", err)
			}
			err = service.InitialSync(cmd.Context(), cfg, drv)
			if err != nil {
				return fmt.Errorf("main; error during initial sync: %w", err)
			}
			return nil
		},
	}

	rootCmd.AddCommand(configCmd, daemonCmd, initCmd)

	if err := rootCmd.Execute(); err != nil {
		logging.Fatalf("ERROR: %s", err)
		os.Exit(1)
	}
}
