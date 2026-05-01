package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"cli/pkg/api"
	"cli/pkg/config"
	"cli/pkg/render"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with a Kestrel server",
	RunE:  runLogin,
}

var (
	loginServer   string
	loginEmail    string
	loginPassword string
)

func init() {
	loginCmd.Flags().StringVarP(&loginServer, "server", "s", "", "Kestrel server URL (e.g. https://app.kestrel.com)")
	loginCmd.Flags().StringVarP(&loginEmail, "email", "e", "", "Email address")
	loginCmd.Flags().StringVarP(&loginPassword, "password", "p", "", "Password (omit for interactive prompt)")
}

func runLogin(cmd *cobra.Command, _ []string) error {
	reader := bufio.NewReader(os.Stdin)

	if loginServer == "" {
		cfg, _ := config.Load()
		if cfg != nil && cfg.ServerURL != "" {
			loginServer = cfg.ServerURL
		} else {
			fmt.Print("Kestrel server URL: ")
			line, _ := reader.ReadString('\n')
			loginServer = strings.TrimSpace(line)
		}
	}
	loginServer = strings.TrimRight(loginServer, "/")

	if loginEmail == "" {
		fmt.Print("Email: ")
		line, _ := reader.ReadString('\n')
		loginEmail = strings.TrimSpace(line)
	}

	if loginPassword == "" {
		fmt.Print("Password: ")
		pw, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("read password: %w", err)
		}
		loginPassword = string(pw)
	}

	client := api.NewUnauthenticated(loginServer)
	resp, err := client.Login(loginEmail, loginPassword)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	if resp.Requires2FA {
		fmt.Print("2FA code: ")
		code, _ := reader.ReadString('\n')
		code = strings.TrimSpace(code)
		_ = code
		return fmt.Errorf("2FA verification via CLI is not yet supported — please log in via the web UI and use the session token")
	}

	cfg := &config.Config{
		ServerURL:    loginServer,
		SessionToken: resp.SessionToken,
		UserID:       resp.UserID,
		Email:        loginEmail,
	}
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("%s Logged in as %s\n", render.Green("✓"), render.Bold(loginEmail))
	fmt.Printf("  Config saved to %s\n", config.Path())
	return nil
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and clear stored credentials",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.Load()
		if err != nil || !cfg.IsLoggedIn() {
			fmt.Println("Not logged in.")
			return nil
		}
		if err := cfg.Clear(); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("clear config: %w", err)
		}
		fmt.Printf("%s Logged out.\n", render.Green("✓"))
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if !cfg.IsLoggedIn() {
			fmt.Println("Not logged in. Run `kestrel login` to authenticate.")
			return nil
		}

		client, err := api.NewFromConfig(cfg)
		if err != nil {
			return err
		}

		err = client.ValidateSession()
		if err != nil {
			fmt.Printf("  Server:  %s\n", cfg.ServerURL)
			fmt.Printf("  Email:   %s\n", cfg.Email)
			fmt.Printf("  Session: %s\n", render.Red("expired / invalid"))
			fmt.Println("\nRun `kestrel login` to re-authenticate.")
			return nil
		}

		fmt.Printf("  Server:  %s\n", cfg.ServerURL)
		fmt.Printf("  Email:   %s\n", render.Bold(cfg.Email))
		fmt.Printf("  Session: %s\n", render.Green("active"))
		return nil
	},
}
