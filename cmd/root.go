package cmd

import (
	"fmt"
	"os"

	"cli/pkg/api"
	"cli/pkg/config"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kestrel",
	Short: "Kestrel CLI — manage workflows from the terminal",
	Long: `Kestrel CLI lets you create, manage, and monitor Kestrel workflows
directly from the command line. Authenticate with your Kestrel credentials
and perform all workflow operations available in the UI.`,
	SilenceUsage: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(workflowsCmd)
	rootCmd.AddCommand(approvalsCmd)
	rootCmd.AddCommand(requestsCmd)
}

func mustClient() *api.Client {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	client, err := api.NewFromConfig(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return client
}
