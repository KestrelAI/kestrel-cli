package cmd

import (
	"fmt"
	"os"
	"strings"

	"cli/pkg/api"
	"cli/pkg/integrations"
	"cli/pkg/render"

	"github.com/spf13/cobra"
)

// connectOAuth handles browser-based integrations (GitHub, GitLab, Slack):
// the CLI fetches the authorization/install URL and prints it for the user.
func connectOAuth(client *api.Client, spec *integrations.Spec) error {
	var (
		authURL string
		err     error
		note    string
	)
	switch spec.Key {
	case "github":
		authURL, err = client.GetGitHubInstallURL()
		note = "Install the Kestrel GitHub App on your org and pick the repositories to grant access to."
	case "gitlab":
		authURL, err = client.GetGitLabAuthURL()
		note = "Authorize Kestrel with your GitLab account; you'll be redirected back automatically."
	case "slack":
		authURL, err = client.GetSlackInstallURL()
		note = "Install the Kestrel Slack app into your workspace."
	default:
		return fmt.Errorf("unsupported OAuth integration: %s", spec.Key)
	}
	if err != nil {
		return fmt.Errorf("get %s authorization URL: %w", spec.Name, err)
	}

	fmt.Printf("%s requires a browser step to finish connecting.\n\n", render.Bold(spec.Name))
	fmt.Printf("  1. Open this URL in your browser:\n\n     %s\n\n", render.Cyan(authURL))
	fmt.Printf("  2. %s\n", note)
	fmt.Printf("  3. Verify with: %s\n", render.Cyan("kestrel integrations list"))
	return nil
}

// clusterOptions captures the operator configuration the UI's "Generate
// Operator Credential" modal collects, so the CLI can produce an equivalent
// Helm values file.
type clusterOptions struct {
	FlowSource       string // cilium | istio | aws-vpc-cni
	MetricsSource    string // none | opentelemetry | datadog
	DatadogNamespace string
	ArgoCD           bool
	ArgoCDNamespace  string
	WorkloadCount    int
	SafeApply        bool
}

// connectCluster onboards a Kubernetes cluster: it generates an operator
// token, writes the Helm values file, and prints the install command.
func connectCluster(cmd *cobra.Command, client *api.Client) error {
	clusterName, _ := cmd.Flags().GetString("cluster-name")
	if clusterName == "" {
		printSetupHelp(integrations.Get("kubernetes"), client.BaseURL())
		v, err := promptString("Cluster name")
		if err != nil {
			return fmt.Errorf("read cluster-name: %w", err)
		}
		clusterName = v
	}
	if clusterName == "" {
		return fmt.Errorf("--cluster-name is required")
	}
	description, _ := cmd.Flags().GetString("description")
	valuesFile, _ := cmd.Flags().GetString("values-file")

	opts := clusterOptions{}
	opts.FlowSource, _ = cmd.Flags().GetString("flow-source")
	opts.MetricsSource, _ = cmd.Flags().GetString("metrics-source")
	opts.DatadogNamespace, _ = cmd.Flags().GetString("datadog-namespace")
	opts.ArgoCD, _ = cmd.Flags().GetBool("argocd")
	opts.ArgoCDNamespace, _ = cmd.Flags().GetString("argocd-namespace")
	opts.WorkloadCount, _ = cmd.Flags().GetInt("workload-count")
	opts.SafeApply, _ = cmd.Flags().GetBool("safe-apply")

	switch opts.FlowSource {
	case "cilium", "istio", "aws-vpc-cni":
	default:
		return fmt.Errorf("--flow-source must be one of: cilium, istio, aws-vpc-cni (got %q)", opts.FlowSource)
	}
	switch opts.MetricsSource {
	case "none", "opentelemetry", "datadog":
	default:
		return fmt.Errorf("--metrics-source must be one of: none, opentelemetry, datadog (got %q)", opts.MetricsSource)
	}
	if opts.WorkloadCount < 0 {
		return fmt.Errorf("--workload-count must be a positive number")
	}

	resp, err := client.OnboardCluster(api.OnboardClusterRequest{
		ClusterName: clusterName,
		Description: description,
		SafeApply:   opts.SafeApply,
	})
	if err != nil {
		return fmt.Errorf("onboard cluster: %w", err)
	}

	chartName := resp.ChartName
	if chartName == "" {
		chartName = "kestrel-operator"
	}
	if valuesFile == "" {
		valuesFile = fmt.Sprintf("kestrel-ai-operator-values-%s.yaml", clusterName)
	}

	values := buildOperatorValuesYAML(resp, opts)
	if err := os.WriteFile(valuesFile, []byte(values), 0o600); err != nil {
		return fmt.Errorf("write values file: %w", err)
	}

	render.Success(fmt.Sprintf("Cluster %q onboarded (id: %s)", clusterName, resp.ClusterID))
	fmt.Println()
	fmt.Printf("  Helm values written to %s (contains the operator token — keep it safe)\n\n", render.Bold(valuesFile))
	fmt.Println("  Install the operator with:")
	fmt.Println()
	fmt.Printf("    %s\n", render.Cyan(fmt.Sprintf(
		"helm install kestrel-operator oci://ghcr.io/kestrelai/charts/%s --version %s --namespace kestrel-ai --create-namespace -f %s",
		chartName, resp.ChartVersion, valuesFile)))
	fmt.Println()
	fmt.Println("  Then verify:")
	fmt.Println()
	fmt.Printf("    %s\n", render.Cyan("kubectl get pods -l app=kestrel-operator -n kestrel-ai"))
	fmt.Println()
	switch opts.MetricsSource {
	case "opentelemetry":
		fmt.Println(render.Gray("  OpenTelemetry enabled: point your OTEL Collector's OTLP exporter to kestrel-operator.kestrel-ai:4317."))
	case "datadog":
		fmt.Println(render.Gray(fmt.Sprintf("  Datadog enabled: the operator reads the API key from the Datadog secret in namespace %q — no manual key configuration needed.", opts.DatadogNamespace)))
	}
	if opts.ArgoCD {
		fmt.Println(render.Gray(fmt.Sprintf("  ArgoCD enabled: the operator auto-discovers the ArgoCD server in namespace %q and authenticates with the in-cluster admin secret.", opts.ArgoCDNamespace)))
	}
	if opts.FlowSource == "aws-vpc-cni" {
		fmt.Println(render.Gray("  AWS EKS VPC CNI: ensure VPC Flow Logs are enabled for the cluster's VPC (connect AWS with `kestrel integrations connect aws`)."))
	}
	fmt.Println(render.Gray("  The operator renews its token automatically every 24 hours."))
	return nil
}

// operatorResourceTier returns the operator resource sizing for a workload
// count, matching the UI's tiers.
func operatorResourceTier(count int) (limitsCPU, limitsMem, reqCPU, reqMem string) {
	switch {
	case count <= 250:
		return "500m", "2Gi", "250m", "1Gi"
	case count <= 1000:
		return "1000m", "4Gi", "500m", "2Gi"
	case count <= 4000:
		return "2000m", "8Gi", "1000m", "4Gi"
	default:
		return "4000m", "16Gi", "2000m", "8Gi"
	}
}

// buildOperatorValuesYAML mirrors the values file the UI generates
// (auth token, cluster identity, flow/metrics sources, ArgoCD, safe-apply,
// resource sizing, and mTLS certificates).
func buildOperatorValuesYAML(resp *api.OnboardClusterResponse, opts clusterOptions) string {
	var sb strings.Builder
	sb.WriteString("# Helm values for Kestrel AI Operator\n")
	sb.WriteString("# Generated by the Kestrel CLI\n\n")
	sb.WriteString(fmt.Sprintf("auth:\n  token: %q\n\n", resp.AccessToken))
	sb.WriteString("operator:\n")
	sb.WriteString(fmt.Sprintf("  cluster:\n    id: %q\n    name: %q\n", resp.ClusterID, resp.ClusterName))
	if opts.FlowSource == "istio" {
		sb.WriteString("  cilium:\n    disableFlows: true\n")
		sb.WriteString("  istio:\n    enabled: true\n")
	} else {
		sb.WriteString("  cilium:\n    disableFlows: false\n")
		sb.WriteString("  istio:\n    enabled: false\n")
	}
	sb.WriteString(fmt.Sprintf("  safeApply:\n    enabled: %t\n", opts.SafeApply))
	switch opts.MetricsSource {
	case "opentelemetry":
		sb.WriteString("  otel:\n    enabled: true\n    receiverPort: 4317\n")
	case "datadog":
		sb.WriteString(fmt.Sprintf("  datadog:\n    enabled: true\n    namespace: %q\n", opts.DatadogNamespace))
	}
	if opts.ArgoCD {
		sb.WriteString(fmt.Sprintf("  argocd:\n    enabled: true\n    namespace: %q\n", opts.ArgoCDNamespace))
	}

	if opts.WorkloadCount > 0 {
		limitsCPU, limitsMem, reqCPU, reqMem := operatorResourceTier(opts.WorkloadCount)
		sb.WriteString("\nresources:\n")
		sb.WriteString(fmt.Sprintf("  limits:\n    cpu: %s\n    memory: %s\n", limitsCPU, limitsMem))
		sb.WriteString(fmt.Sprintf("  requests:\n    cpu: %s\n    memory: %s\n", reqCPU, reqMem))
	}

	if certs := resp.MTLSCertificates; certs != nil {
		sb.WriteString("\ncertificates:\n")
		sb.WriteString("  client:\n")
		sb.WriteString("    cert: |\n" + indentPEM(certs.ClientCert))
		sb.WriteString("    key: |\n" + indentPEM(certs.ClientKey))
		sb.WriteString("  ca:\n")
		sb.WriteString("    cert: |\n" + indentPEM(certs.CACert))
	}
	return sb.String()
}

func indentPEM(pem string) string {
	lines := strings.Split(strings.TrimRight(pem, "\n"), "\n")
	var sb strings.Builder
	for _, line := range lines {
		sb.WriteString("      " + line + "\n")
	}
	return sb.String()
}

// connectCloud handles AWS (two-step IAM role flow) and OCI (API key auth).
func connectCloud(cmd *cobra.Command, client *api.Client, spec *integrations.Spec) error {
	switch spec.Key {
	case "aws":
		return connectAWS(cmd, client)
	case "oci":
		return connectOCI(cmd, client)
	}
	return fmt.Errorf("unsupported cloud integration: %s", spec.Key)
}

// connectAWS runs the two-step AWS flow:
//   - step 1 (no --role-arn): bootstrap, print CloudFormation/CLI instructions
//   - step 2 (--role-arn + --external-id): verify and store the connection
func connectAWS(cmd *cobra.Command, client *api.Client) error {
	region, _ := cmd.Flags().GetString("region")
	roleARN, _ := cmd.Flags().GetString("role-arn")
	externalID, _ := cmd.Flags().GetString("external-id")

	if roleARN == "" {
		resp, err := client.AWSBootstrap(region)
		if err != nil {
			return fmt.Errorf("bootstrap AWS integration: %w", err)
		}
		fmt.Printf("%s\n\n", render.Bold("AWS connection — step 1 of 2"))
		fmt.Printf("  External ID: %s\n\n", render.Bold(resp.ExternalID))
		fmt.Println("  Create the Kestrel IAM role with either option:")
		fmt.Println()
		fmt.Printf("  Option A — CloudFormation (browser):\n\n     %s\n\n", render.Cyan(resp.CFNLaunchURL))
		if resp.CLICreateCmd != "" {
			fmt.Printf("  Option B — AWS CLI:\n\n     %s\n\n", render.Cyan(resp.CLICreateCmd))
			if resp.CLIOutputCmd != "" {
				fmt.Printf("     Then get the role ARN:\n\n     %s\n\n", render.Cyan(resp.CLIOutputCmd))
			}
		}
		fmt.Println("  Once the role exists, finish with:")
		fmt.Println()
		fmt.Printf("    %s\n", render.Cyan(fmt.Sprintf(
			"kestrel integrations connect aws --role-arn <arn> --external-id %s --region %s",
			resp.ExternalID, resp.Region)))
		return nil
	}

	if externalID == "" {
		return fmt.Errorf("--external-id is required with --role-arn (printed during step 1)")
	}
	if _, err := client.AWSVerify(roleARN, externalID, region); err != nil {
		return fmt.Errorf("verify AWS role: %w", err)
	}
	render.Success("AWS account connected")
	return nil
}

func connectOCI(cmd *cobra.Command, client *api.Client) error {
	tenancy, _ := cmd.Flags().GetString("tenancy-ocid")
	user, _ := cmd.Flags().GetString("user-ocid")
	fingerprint, _ := cmd.Flags().GetString("fingerprint")
	keyFile, _ := cmd.Flags().GetString("private-key-file")
	region, _ := cmd.Flags().GetString("region")
	connName, _ := cmd.Flags().GetString("connection-name")

	// Prompt for anything missing so a bare `connect oci` works when pasted.
	printedHelp := false
	for _, p := range []struct {
		label string
		flag  string
		val   *string
	}{
		{"Tenancy OCID", "tenancy-ocid", &tenancy},
		{"User OCID", "user-ocid", &user},
		{"API key fingerprint", "fingerprint", &fingerprint},
		{"Path to PEM private key file", "private-key-file", &keyFile},
		{"OCI region", "region", &region},
	} {
		if *p.val != "" {
			continue
		}
		if !printedHelp {
			printSetupHelp(integrations.Get("oci"), client.BaseURL())
			printedHelp = true
		}
		v, err := promptString(p.label)
		if err != nil {
			return fmt.Errorf("read --%s: %w", p.flag, err)
		}
		if v == "" {
			return fmt.Errorf("--%s is required", p.flag)
		}
		*p.val = v
	}

	keyData, err := os.ReadFile(keyFile)
	if err != nil {
		return fmt.Errorf("read --private-key-file: %w", err)
	}

	if _, err := client.OCIVerify(api.OCIVerifyRequest{
		TenancyOCID:    tenancy,
		UserOCID:       user,
		Fingerprint:    fingerprint,
		PrivateKey:     string(keyData),
		Region:         region,
		ConnectionName: connName,
	}); err != nil {
		return fmt.Errorf("verify OCI credentials: %w", err)
	}
	render.Success("OCI tenancy connected")
	return nil
}
