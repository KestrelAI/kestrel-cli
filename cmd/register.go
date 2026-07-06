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

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Create a new Kestrel account",
	Long: `Create a new Kestrel account with a work email and password.

If email verification is enabled on the server, a 6-character code is sent
to your email. Enter it when prompted (or later via ` + "`kestrel verify-email <code>`" + `)
to finish setup — you'll be logged in automatically.`,
	RunE: runRegister,
}

var (
	registerServer   string
	registerEmail    string
	registerPassword string
)

var verifyEmailCmd = &cobra.Command{
	Use:   "verify-email <code>",
	Short: "Verify your email with the 6-character code from your inbox",
	Args:  cobra.ExactArgs(1),
	RunE:  runVerifyEmail,
}

var verifyEmailServer string

func init() {
	registerCmd.Flags().StringVarP(&registerServer, "server", "s", "", "Kestrel server URL (default https://platform.usekestrel.ai)")
	registerCmd.Flags().StringVarP(&registerEmail, "email", "e", "", "Work email address")
	registerCmd.Flags().StringVarP(&registerPassword, "password", "p", "", "Password (omit for interactive prompt)")
	rootCmd.AddCommand(registerCmd)

	verifyEmailCmd.Flags().StringVarP(&verifyEmailServer, "server", "s", "", "Kestrel server URL (default https://platform.usekestrel.ai)")
	rootCmd.AddCommand(verifyEmailCmd)
}

// resolveServer picks the server URL from a flag, saved config, or the
// platform default, in that order.
func resolveServer(flagValue string) string {
	server := flagValue
	if server == "" {
		cfg, _ := config.Load()
		if cfg != nil && cfg.ServerURL != "" {
			server = cfg.ServerURL
		} else {
			server = "https://platform.usekestrel.ai"
		}
	}
	return strings.TrimRight(server, "/")
}

func runRegister(cmd *cobra.Command, _ []string) error {
	reader := bufio.NewReader(os.Stdin)
	server := resolveServer(registerServer)

	if registerEmail == "" {
		fmt.Print("Work email: ")
		line, _ := reader.ReadString('\n')
		registerEmail = strings.TrimSpace(line)
	}
	if registerEmail == "" {
		return fmt.Errorf("email is required")
	}

	if registerPassword == "" {
		fmt.Print("Password (8-30 characters): ")
		pw, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("read password: %w", err)
		}
		fmt.Print("Confirm password: ")
		pw2, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("read password: %w", err)
		}
		if string(pw) != string(pw2) {
			return fmt.Errorf("passwords do not match")
		}
		registerPassword = string(pw)
	}
	if len(registerPassword) < 8 || len(registerPassword) > 30 {
		return fmt.Errorf("password must be 8-30 characters long")
	}

	client := api.NewUnauthenticated(server)
	resp, err := client.Register(registerEmail, registerPassword)
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	render.Success("Account created")
	fmt.Printf("  %s\n", resp.Message)

	// If the server auto-verified the account (email delivery disabled),
	// log in right away with the same credentials.
	if !strings.Contains(strings.ToLower(resp.Message), "verification code") {
		loginResp, err := client.Login(registerEmail, registerPassword)
		if err != nil {
			fmt.Println("\nRun `kestrel login` to authenticate.")
			return nil
		}
		if loginResp.Requires2FA {
			fmt.Println("\nThis account requires 2FA — log in via the web UI.")
			return nil
		}
		cfg := &config.Config{
			ServerURL:    server,
			SessionToken: loginResp.SessionToken,
			UserID:       loginResp.UserID,
			Email:        registerEmail,
		}
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Printf("\n%s Logged in as %s\n", render.Green("✓"), render.Bold(registerEmail))
		fmt.Printf("  Config saved to %s\n", config.Path())
		printPostRegisterHints()
		return nil
	}

	// Email verification required. Prompt for the code when interactive,
	// otherwise point at the standalone verify-email command.
	if !term.IsTerminal(int(syscall.Stdin)) {
		fmt.Printf("\nCheck %s for a 6-character code, then run:\n", registerEmail)
		fmt.Printf("  kestrel verify-email <code> --server %s\n", server)
		return nil
	}

	fmt.Println()
	for attempts := 0; attempts < 3; attempts++ {
		fmt.Printf("Verification code (check %s, or press Enter to verify later): ", registerEmail)
		line, _ := reader.ReadString('\n')
		code := strings.TrimSpace(line)
		if code == "" {
			fmt.Println("\nYour account stays unverified until you enter the code. Verify with:")
			fmt.Printf("  kestrel verify-email <code> --server %s\n", server)
			return nil
		}
		if err := completeVerification(server, code, registerEmail); err != nil {
			render.Warn(err.Error())
			continue
		}
		return nil
	}
	fmt.Println("\nToo many invalid codes. Verify later with:")
	fmt.Printf("  kestrel verify-email <code> --server %s\n", server)
	return nil
}

func runVerifyEmail(cmd *cobra.Command, args []string) error {
	server := resolveServer(verifyEmailServer)
	return completeVerification(server, args[0], "")
}

// completeVerification exchanges the emailed code for a verified account
// and, when the server returns a session token, saves the login session.
// email may be empty when the code is entered via the standalone command.
func completeVerification(server, code, email string) error {
	client := api.NewUnauthenticated(server)
	resp, err := client.VerifyEmail(code)
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	render.Success("Email verified")
	if resp.TenantName != "" {
		fmt.Printf("  Organization: %s\n", resp.TenantName)
	}

	if resp.SessionToken == "" {
		// 2FA-protected accounts (or session issue) must use the normal login flow.
		fmt.Println("\nRun `kestrel login` to authenticate.")
		return nil
	}

	cfg, _ := config.Load()
	if cfg == nil {
		cfg = &config.Config{}
	}
	cfg.ServerURL = server
	cfg.SessionToken = resp.SessionToken
	cfg.UserID = resp.UserID
	if email != "" {
		cfg.Email = email
	}
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("\n%s Logged in\n", render.Green("✓"))
	fmt.Printf("  Config saved to %s\n", config.Path())
	printPostRegisterHints()
	return nil
}

func printPostRegisterHints() {
	fmt.Println("\nNext steps:")
	fmt.Println("  kestrel integrations list        # see available integrations")
	fmt.Println("  kestrel integrations connect ... # connect your tools")
	fmt.Println("  kestrel workflows list           # view your workflows")
}
