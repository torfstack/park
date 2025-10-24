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

	srv := service.NewService(config.LoadConfig())

	var setupCmd = &cobra.Command{
		Use:   "setup",
		Short: "Setup and perform initial sync",
		PreRun: func(cmd *cobra.Command, args []string) {
			logging.LogLevelDebug = debug
		},
		Run: func(cmd *cobra.Command, args []string) {
			srv.SetupAndInitialSync()
		},
	}

	var loginCmd = &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Google Drive",
		PreRun: func(cmd *cobra.Command, args []string) {
			logging.LogLevelDebug = debug
		},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Authentication successful.")
		},
	}

	var checkCmd = &cobra.Command{
		Use:   "check",
		Short: "Check for changes",
		PreRun: func(cmd *cobra.Command, args []string) {
			logging.LogLevelDebug = debug
		},
		Run: func(cmd *cobra.Command, args []string) {
			srv.CheckForChanges(context.Background())
		},
	}

	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List files",
		PreRun: func(cmd *cobra.Command, args []string) {
			logging.LogLevelDebug = debug
		},
		Run: func(cmd *cobra.Command, args []string) {
			srv.ListFiles()
		},
	}

	rootCmd.AddCommand(setupCmd, loginCmd, checkCmd, listCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
