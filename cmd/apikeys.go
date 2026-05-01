package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"cli/pkg/api"
	"cli/pkg/config"
	"cli/pkg/render"

	"github.com/spf13/cobra"
)

// kestrel auth — configure API key authentication
var authCmd = &cobra.Command{
	Use:   "auth [api-key]",
	Short: "Authenticate with an API key",
	Long:  `Configure the CLI to use an API key for authentication. API keys can be created in the Kestrel platform under Workflows > API Keys.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAuth,
}

var authServer string

func init() {
	authCmd.Flags().StringVarP(&authServer, "server", "s", "", "Kestrel server URL")
	rootCmd.AddCommand(authCmd)
}

func runAuth(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	if authServer == "" {
		cfg, _ := config.Load()
		if cfg != nil && cfg.ServerURL != "" {
			authServer = cfg.ServerURL
		} else {
			authServer = "https://platform.usekestrel.ai"
		}
	}
	authServer = strings.TrimRight(authServer, "/")

	var apiKey string
	if len(args) > 0 {
		apiKey = args[0]
	} else {
		fmt.Print("API Key (kestrel_sk_...): ")
		line, _ := reader.ReadString('\n')
		apiKey = strings.TrimSpace(line)
	}

	if !strings.HasPrefix(apiKey, "kestrel_sk_") {
		return fmt.Errorf("invalid API key format — keys start with kestrel_sk_")
	}

	// Validate the key by making a test request
	cfg := &config.Config{ServerURL: authServer, APIKey: apiKey}
	client, _ := api.NewFromConfig(cfg)
	_, err := client.ListWorkflows("")
	if err != nil {
		return fmt.Errorf("API key validation failed: %w", err)
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	render.Success("Authenticated with API key")
	fmt.Printf("  Server: %s\n", authServer)
	fmt.Printf("  Key:    %s...\n", apiKey[:20])
	return nil
}

// kestrel apikeys — manage API keys
var apikeysCmd = &cobra.Command{
	Use:     "apikeys",
	Aliases: []string{"keys"},
	Short:   "Manage API keys",
}

var apikeysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all API keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()
		keys, err := client.ListAPIKeys()
		if err != nil {
			return err
		}
		if len(keys) == 0 {
			fmt.Println("No API keys.")
			return nil
		}
		render.PrintHeader("NAME", "KEY", "SCOPES", "STATUS", "LAST USED", "CREATED")
		for _, k := range keys {
			status := "Active"
			if k.RevokedAt != nil {
				status = "Revoked"
			} else if k.ExpiresAt != nil {
				status = "Active"
			}
			scopes := "Full Access"
			if len(k.Scopes) > 0 && k.Scopes[0] != "*" {
				scopes = fmt.Sprintf("%d scopes", len(k.Scopes))
			}
			lastUsed := "—"
			if k.LastUsedAt != nil {
				lastUsed = render.FormatTime(*k.LastUsedAt)
			}
			render.PrintRow(k.Name, k.KeyPrefix+"...", scopes, status, lastUsed, render.FormatTime(k.CreatedAt))
		}
		return nil
	},
}

var (
	createKeyScopes    string
	createKeyExpiresIn string
)

var apikeysCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()
		scopes := []string{"*"}
		if createKeyScopes != "" {
			scopes = strings.Split(createKeyScopes, ",")
		}
		key, err := client.CreateAPIKey(api.CreateAPIKeyRequest{
			Name:      args[0],
			Scopes:    scopes,
			ExpiresIn: createKeyExpiresIn,
		})
		if err != nil {
			return err
		}
		render.Success("API key created")
		fmt.Println()
		fmt.Printf("  %s\n", key.RawKey)
		fmt.Println()
		render.Warn("Copy this key now — it won't be shown again.")
		return nil
	},
}

var apikeysRevokeCmd = &cobra.Command{
	Use:   "revoke <id>",
	Short: "Revoke an API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()
		if err := client.RevokeAPIKey(args[0]); err != nil {
			return err
		}
		render.Success("API key revoked")
		return nil
	},
}

var apikeysDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an API key permanently",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()
		if err := client.DeleteAPIKey(args[0]); err != nil {
			return err
		}
		render.Success("API key deleted")
		return nil
	},
}

func init() {
	apikeysCreateCmd.Flags().StringVar(&createKeyScopes, "scopes", "", "Comma-separated scopes (default: * for full access)")
	apikeysCreateCmd.Flags().StringVar(&createKeyExpiresIn, "expires-in", "never", "Expiration: 30d, 90d, 365d, or never")

	apikeysCmd.AddCommand(apikeysListCmd)
	apikeysCmd.AddCommand(apikeysCreateCmd)
	apikeysCmd.AddCommand(apikeysRevokeCmd)
	apikeysCmd.AddCommand(apikeysDeleteCmd)
	rootCmd.AddCommand(apikeysCmd)
}
