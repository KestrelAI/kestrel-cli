package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"syscall"

	"cli/pkg/api"
	"cli/pkg/integrations"
	"cli/pkg/render"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// kestrel integrations — manage platform integrations
var integrationsCmd = &cobra.Command{
	Use:     "integrations",
	Aliases: []string{"integration", "int"},
	Short:   "Manage Kestrel integrations",
	Long: `List, connect, test, and disconnect Kestrel integrations
(cloud providers, CI/CD, databases, alerting, knowledge sources, and more).

Run 'kestrel integrations list' to see every integration and its status,
and 'kestrel integrations connect <name> --help' for the exact credentials
each integration needs.`,
}

var integrationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all integrations and their connection status",
	RunE:  runIntegrationsList,
}

var integrationsTestCmd = &cobra.Command{
	Use:   "test <name>",
	Short: "Test an integration's connection",
	Args:  cobra.ExactArgs(1),
	RunE:  runIntegrationsTest,
}

var integrationsDisconnectCmd = &cobra.Command{
	Use:   "disconnect <name>",
	Short: "Disconnect an integration",
	Args:  cobra.ExactArgs(1),
	RunE:  runIntegrationsDisconnect,
}

var integrationsConnectCmd = &cobra.Command{
	Use:   "connect <name> [flags]",
	Short: "Connect an integration",
	Long: `Connect an integration by name. Each integration accepts different flags
depending on its authentication type — run
'kestrel integrations connect <name> --help' to see them.

Examples:
  kestrel integrations connect cloudflare --api-token <token> --account-id <id>
  kestrel integrations connect pagerduty --api-token <token> --webhook-secret <secret>
  kestrel integrations connect confluence --base-url https://acme.atlassian.net --email me@acme.com --api-token <token>
  kestrel integrations connect kubernetes --cluster-name prod
  kestrel integrations connect github   # prints the GitHub App install URL`,
}

func init() {
	integrationsCmd.AddCommand(integrationsListCmd)
	integrationsCmd.AddCommand(integrationsConnectCmd)
	integrationsCmd.AddCommand(integrationsTestCmd)
	integrationsCmd.AddCommand(integrationsDisconnectCmd)
	rootCmd.AddCommand(integrationsCmd)

	// One connect subcommand per integration, with flags generated from the registry.
	for i := range integrations.Registry {
		spec := &integrations.Registry[i]
		integrationsConnectCmd.AddCommand(buildConnectCommand(spec))
	}
}

// buildConnectCommand creates the `connect <name>` subcommand for one integration.
func buildConnectCommand(spec *integrations.Spec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   spec.Key,
		Short: fmt.Sprintf("Connect %s — %s", spec.Name, spec.Description),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := mustClient()
			switch spec.Kind {
			case integrations.KindToken:
				return connectToken(cmd, client, spec)
			case integrations.KindKnowledge:
				return connectKnowledge(cmd, client, spec)
			case integrations.KindOAuth:
				return connectOAuth(client, spec)
			case integrations.KindCluster:
				return connectCluster(cmd, client)
			case integrations.KindCloud:
				return connectCloud(cmd, client, spec)
			}
			return fmt.Errorf("unsupported integration kind: %s", spec.Kind)
		},
	}

	for _, f := range spec.Fields {
		usage := f.Usage
		if f.Required {
			usage += " (required)"
		}
		cmd.Flags().String(f.Flag, "", usage)
		if f.File {
			cmd.Flags().String(f.Flag+"-file", "", fmt.Sprintf("Read %s from a file", strings.ReplaceAll(f.Flag, "-", " ")))
		}
	}

	switch spec.Kind {
	case integrations.KindCluster:
		cmd.Flags().String("cluster-name", "", "Cluster name (required)")
		cmd.Flags().String("description", "", "Cluster description")
		cmd.Flags().Bool("safe-apply", false, "Allow the operator to auto-apply YAML changes")
		cmd.Flags().String("values-file", "", "Path to write the generated Helm values file (default kestrel-ai-operator-values-<name>.yaml)")
	case integrations.KindCloud:
		if spec.Key == "aws" {
			cmd.Flags().String("region", "us-east-1", "AWS region")
			cmd.Flags().String("role-arn", "", "IAM role ARN (step 2, after creating the role)")
			cmd.Flags().String("external-id", "", "External ID from the bootstrap step (step 2)")
		} else if spec.Key == "oci" {
			cmd.Flags().String("tenancy-ocid", "", "OCI tenancy OCID (required)")
			cmd.Flags().String("user-ocid", "", "OCI user OCID (required)")
			cmd.Flags().String("fingerprint", "", "API key fingerprint (required)")
			cmd.Flags().String("private-key-file", "", "Path to PEM private key file (required)")
			cmd.Flags().String("region", "", "OCI region (required)")
			cmd.Flags().String("connection-name", "", "Display name for this connection")
		}
	}

	return cmd
}

func runIntegrationsList(cmd *cobra.Command, args []string) error {
	client := mustClient()
	statuses, err := client.GetIntegrationsStatus()
	if err != nil {
		return err
	}

	connected := map[string]bool{}
	for _, s := range statuses {
		connected[s.ID] = s.Connected
	}

	// Group registry entries by kind for a readable listing.
	specs := make([]integrations.Spec, len(integrations.Registry))
	copy(specs, integrations.Registry)
	sort.SliceStable(specs, func(i, j int) bool { return specs[i].Key < specs[j].Key })

	rows := make([][]string, 0, len(specs))
	for _, s := range specs {
		status := render.Gray("not connected")
		if connected[s.Key] {
			status = render.Green("connected")
		}
		rows = append(rows, []string{s.Key, s.Name, string(s.Kind), status})
	}
	fmt.Print(render.Table([]string{"NAME", "INTEGRATION", "TYPE", "STATUS"}, rows))
	fmt.Println()
	fmt.Println(render.Gray("Connect with: kestrel integrations connect <name> --help"))
	return nil
}

func runIntegrationsTest(cmd *cobra.Command, args []string) error {
	key := strings.ToLower(args[0])
	spec := integrations.Get(key)
	if spec == nil {
		return unknownIntegrationErr(key)
	}
	client := mustClient()

	switch spec.Kind {
	case integrations.KindToken:
		if spec.TestPath == "" {
			return fmt.Errorf("%s does not support connection tests", spec.Name)
		}
		out, err := client.TestIntegration(spec.TestPath)
		if err != nil {
			return fmt.Errorf("%s test failed: %w", spec.Name, err)
		}
		render.Success(fmt.Sprintf("%s connection OK", spec.Name))
		if msg, ok := out["message"].(string); ok && msg != "" {
			fmt.Printf("  %s\n", msg)
		}
		return nil
	case integrations.KindKnowledge:
		src, err := findKnowledgeSource(client, spec.SourceType)
		if err != nil {
			return err
		}
		out, err := client.TestKnowledgeSource(src.ID)
		if err != nil {
			return fmt.Errorf("%s test failed: %w", spec.Name, err)
		}
		render.Success(fmt.Sprintf("%s connection OK", spec.Name))
		if msg, ok := out["message"].(string); ok && msg != "" {
			fmt.Printf("  %s\n", msg)
		}
		return nil
	default:
		return fmt.Errorf("%s does not support `test` from the CLI — check status with `kestrel integrations list`", spec.Name)
	}
}

func runIntegrationsDisconnect(cmd *cobra.Command, args []string) error {
	key := strings.ToLower(args[0])
	spec := integrations.Get(key)
	if spec == nil {
		return unknownIntegrationErr(key)
	}
	client := mustClient()

	switch spec.Kind {
	case integrations.KindToken:
		if spec.DisconnectPath == "" {
			return fmt.Errorf("%s cannot be disconnected from the CLI", spec.Name)
		}
		if err := client.DisconnectIntegration(spec.DisconnectPath); err != nil {
			return err
		}
		render.Success(fmt.Sprintf("%s disconnected", spec.Name))
		return nil
	case integrations.KindKnowledge:
		src, err := findKnowledgeSource(client, spec.SourceType)
		if err != nil {
			return err
		}
		if err := client.DeleteKnowledgeSource(src.ID); err != nil {
			return err
		}
		render.Success(fmt.Sprintf("%s disconnected", spec.Name))
		return nil
	default:
		return fmt.Errorf("%s must be disconnected in the UI: %s", spec.Name, client.FormatIntegrationPage("/integrations"))
	}
}

// collectFields resolves each registry field from flags, files, or an
// interactive prompt (hidden input for secrets). Required fields that remain
// empty produce an error.
func collectFields(cmd *cobra.Command, spec *integrations.Spec) (map[string]interface{}, error) {
	body := map[string]interface{}{}
	for _, f := range spec.Fields {
		val, _ := cmd.Flags().GetString(f.Flag)

		if val == "" && f.File {
			if path, _ := cmd.Flags().GetString(f.Flag + "-file"); path != "" {
				data, err := os.ReadFile(path)
				if err != nil {
					return nil, fmt.Errorf("read --%s-file: %w", f.Flag, err)
				}
				val = strings.TrimSpace(string(data))
			}
		}

		if val == "" && f.Required {
			v, err := promptField(f)
			if err != nil {
				return nil, err
			}
			val = v
		}

		if val == "" && f.Required {
			return nil, fmt.Errorf("--%s is required", f.Flag)
		}
		if val != "" {
			body[f.JSON] = val
		}
	}
	return body, nil
}

func promptField(f integrations.Field) (string, error) {
	if f.Secret {
		fmt.Printf("%s: ", f.Usage)
		b, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("read %s: %w", f.Flag, err)
		}
		return strings.TrimSpace(string(b)), nil
	}
	fmt.Printf("%s: ", f.Usage)
	var v string
	if _, err := fmt.Scanln(&v); err != nil {
		return "", fmt.Errorf("read %s: %w", f.Flag, err)
	}
	return strings.TrimSpace(v), nil
}

func connectToken(cmd *cobra.Command, client *api.Client, spec *integrations.Spec) error {
	body, err := collectFields(cmd, spec)
	if err != nil {
		return err
	}
	if _, err := client.ConnectIntegration(spec.ConnectPath, body); err != nil {
		return fmt.Errorf("connect %s: %w", spec.Name, err)
	}
	render.Success(fmt.Sprintf("%s connected", spec.Name))
	if spec.PostConnectHint != "" {
		fmt.Printf("  %s\n", render.Gray(spec.PostConnectHint))
	}
	if spec.TestPath != "" {
		fmt.Printf("  Verify anytime with: %s\n", render.Cyan("kestrel integrations test "+spec.Key))
	}
	return nil
}

func connectKnowledge(cmd *cobra.Command, client *api.Client, spec *integrations.Spec) error {
	body, err := collectFields(cmd, spec)
	if err != nil {
		return err
	}
	body["source_type"] = spec.SourceType
	body["name"] = spec.Name
	body["enabled"] = true

	src, err := client.CreateKnowledgeSource(body)
	if err != nil {
		return fmt.Errorf("connect %s: %w", spec.Name, err)
	}
	render.Success(fmt.Sprintf("%s connected", spec.Name))

	// Immediately run a connection test so bad credentials surface here.
	if out, tErr := client.TestKnowledgeSource(src.ID); tErr != nil {
		render.Warn(fmt.Sprintf("Connection test failed: %v", tErr))
	} else if msg, ok := out["message"].(string); ok && msg != "" {
		fmt.Printf("  %s\n", msg)
	}
	return nil
}

func findKnowledgeSource(client *api.Client, sourceType string) (*api.TribalKnowledgeSource, error) {
	sources, err := client.ListKnowledgeSources()
	if err != nil {
		return nil, err
	}
	for i := range sources {
		if sources[i].SourceType == sourceType {
			return &sources[i], nil
		}
	}
	return nil, fmt.Errorf("no %s source is connected", sourceType)
}

func unknownIntegrationErr(key string) error {
	names := make([]string, 0, len(integrations.Registry))
	for _, s := range integrations.Registry {
		names = append(names, s.Key)
	}
	sort.Strings(names)
	return fmt.Errorf("unknown integration %q — available: %s", key, strings.Join(names, ", "))
}
