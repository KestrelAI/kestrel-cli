package cmd

import (
	"bufio"
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

var integrationsWebhookSecretCmd = &cobra.Command{
	Use:   "webhook-secret <name>",
	Short: "Save a webhook signing secret for a connected integration",
	Long: `Save a webhook signing secret for an integration where the third party
generates the secret (Vercel, Railway, PlanetScale, Supabase). Create the
webhook in the third-party product first, then paste its signing secret here —
no need to re-enter your API token.

For PlanetScale (one secret per database webhook) run this once per secret;
new secrets are merged with the ones already stored.

Examples:
  kestrel integrations webhook-secret vercel        # prompts (hidden input)
  kestrel integrations webhook-secret planetscale --secret <secret>`,
	Args: cobra.ExactArgs(1),
	RunE: runIntegrationsWebhookSecret,
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
	integrationsWebhookSecretCmd.Flags().String("secret", "", "Webhook signing secret (prompted with hidden input if omitted)")
	integrationsCmd.AddCommand(integrationsWebhookSecretCmd)
	rootCmd.AddCommand(integrationsCmd)

	// One connect subcommand per integration, with flags generated from the registry.
	for i := range integrations.Registry {
		spec := &integrations.Registry[i]
		integrationsConnectCmd.AddCommand(buildConnectCommand(spec))
	}
}

// buildConnectCommand creates the `connect <name>` subcommand for one integration.
func buildConnectCommand(spec *integrations.Spec) *cobra.Command {
	long := fmt.Sprintf("Connect %s — %s.", spec.Name, spec.Description)
	if spec.SetupHelp != "" {
		long += "\n\nSetup:\n" + indentLines(expandServer(spec.SetupHelp, "<kestrel-server>"), "  ")
	}
	cmd := &cobra.Command{
		Use:   spec.Key,
		Short: fmt.Sprintf("Connect %s — %s", spec.Name, spec.Description),
		Long:  long,
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
		cmd.Flags().String("flow-source", "cilium", "Flow collection source: cilium, istio, or aws-vpc-cni")
		cmd.Flags().String("metrics-source", "none", "Metrics source: none (K8s Metrics Server), opentelemetry, or datadog")
		cmd.Flags().String("datadog-namespace", "datadog", "Namespace where Datadog is deployed (with --metrics-source datadog)")
		cmd.Flags().Bool("argocd", false, "Enable ArgoCD deployment sync (auto-discovers the ArgoCD server in-cluster)")
		cmd.Flags().String("argocd-namespace", "argocd", "Namespace where ArgoCD is deployed (with --argocd)")
		cmd.Flags().Int("workload-count", 0, "Workload count for operator resource sizing (kubectl get deploy,sts,ds,job,cj -A --no-headers | wc -l)")
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

func runIntegrationsWebhookSecret(cmd *cobra.Command, args []string) error {
	key := strings.ToLower(args[0])
	spec := integrations.Get(key)
	if spec == nil {
		return unknownIntegrationErr(key)
	}
	if spec.WebhookSecretPath == "" {
		supported := make([]string, 0, 4)
		for _, s := range integrations.Registry {
			if s.WebhookSecretPath != "" {
				supported = append(supported, s.Key)
			}
		}
		sort.Strings(supported)
		return fmt.Errorf("%s does not take a pasted webhook secret — supported: %s", spec.Name, strings.Join(supported, ", "))
	}

	secret, _ := cmd.Flags().GetString("secret")
	if secret == "" {
		v, err := promptField(integrations.Field{Flag: "secret", Usage: "Webhook signing secret", Secret: true})
		if err != nil {
			return err
		}
		secret = v
	}
	if secret == "" {
		return fmt.Errorf("webhook secret is required")
	}

	client := mustClient()
	out, err := client.ConnectIntegration(spec.WebhookSecretPath, map[string]interface{}{"webhook_secret": secret})
	if err != nil {
		return fmt.Errorf("save %s webhook secret: %w", spec.Name, err)
	}
	render.Success(fmt.Sprintf("%s webhook secret saved", spec.Name))
	if n, ok := out["webhook_secret_count"].(float64); ok && n > 1 {
		fmt.Printf("  %d webhook secrets stored (one per database webhook)\n", int(n))
	}
	return nil
}

// collectFields resolves each registry field from flags, files, or an
// interactive prompt (hidden input for secrets). Before the first prompt, the
// integration's setup instructions are printed so the user knows where to
// create the credential (mirroring the platform UI). Optional webhook-secret
// fields are offered interactively too (Enter skips them — the secret can be
// added later with `kestrel integrations webhook-secret <name>`). Required
// fields that remain empty produce an error.
func collectFields(cmd *cobra.Command, spec *integrations.Spec, serverURL string) (map[string]interface{}, error) {
	body := map[string]interface{}{}
	printedHelp := false
	printHelpOnce := func() {
		if !printedHelp {
			printSetupHelp(spec, serverURL)
			printedHelp = true
		}
	}
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
			printHelpOnce()
			v, err := promptField(f)
			if err != nil {
				return nil, err
			}
			val = v
		}

		// Optional webhook secrets: offer them interactively so webhook setup
		// can be completed from the CLI in one pass, but let Enter skip.
		if val == "" && !f.Required && f.Flag == "webhook-secret" && isInteractive() {
			printHelpOnce()
			v, err := promptField(integrations.Field{
				Flag: f.Flag, JSON: f.JSON, Secret: f.Secret,
				Usage: f.Usage + " (optional — press Enter to skip, add later with `kestrel integrations webhook-secret " + spec.Key + "`)",
			})
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

// isInteractive reports whether stdin is a terminal (i.e. we can prompt).
func isInteractive() bool {
	return term.IsTerminal(int(syscall.Stdin))
}

// indentLines prefixes every line of s with the given indent.
func indentLines(s, indent string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = indent + l
	}
	return strings.Join(lines, "\n")
}

// expandServer substitutes the {server} and <server> placeholders in setup
// help / post-connect hints with the configured Kestrel server URL.
func expandServer(s, serverURL string) string {
	s = strings.ReplaceAll(s, "{server}", serverURL)
	s = strings.ReplaceAll(s, "<server>", serverURL)
	return s
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
	v, err := promptString(f.Usage)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", f.Flag, err)
	}
	return v, nil
}

// promptString reads a single line of non-secret input from stdin.
func promptString(label string) (string, error) {
	fmt.Printf("%s: ", label)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// printSetupHelp prints an integration's setup instructions (once) before
// interactive prompts, so users pasting bare connect commands know where to
// find each value.
func printSetupHelp(spec *integrations.Spec, serverURL string) {
	if spec == nil || spec.SetupHelp == "" {
		return
	}
	fmt.Printf("Connecting %s\n\n", spec.Name)
	fmt.Println(render.Gray(indentLines(expandServer(spec.SetupHelp, serverURL), "  ")))
	fmt.Println()
}

func connectToken(cmd *cobra.Command, client *api.Client, spec *integrations.Spec) error {
	body, err := collectFields(cmd, spec, client.BaseURL())
	if err != nil {
		return err
	}
	out, err := client.ConnectIntegration(spec.ConnectPath, body)
	if err != nil {
		return fmt.Errorf("connect %s: %w", spec.Name, err)
	}
	render.Success(fmt.Sprintf("%s connected", spec.Name))
	// Some integrations (CircleCI, Pulumi, Jenkins, Terraform, Cloudflare)
	// return a Kestrel-generated webhook secret that the user must paste into
	// the third-party product when creating the webhook — print it.
	for _, key := range []string{"webhook_secret", "notification_token"} {
		if secret, ok := out[key].(string); ok && secret != "" {
			label := "Webhook secret"
			if key == "notification_token" {
				label = "Notification token"
			}
			fmt.Printf("  %s: %s\n", label, render.Cyan(secret))
		}
	}
	if spec.PostConnectHint != "" {
		fmt.Printf("  %s\n", render.Gray(expandServer(spec.PostConnectHint, client.BaseURL())))
	}
	if spec.TestPath != "" {
		fmt.Printf("  Verify anytime with: %s\n", render.Cyan("kestrel integrations test "+spec.Key))
	}
	return nil
}

func connectKnowledge(cmd *cobra.Command, client *api.Client, spec *integrations.Spec) error {
	body, err := collectFields(cmd, spec, client.BaseURL())
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
