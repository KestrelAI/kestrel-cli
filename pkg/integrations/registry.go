// Package integrations defines the CLI-side registry of Kestrel integrations:
// how each one is connected (credential fields, endpoint paths) so that
// `kestrel integrations connect <name>` can drive every supported integration
// with the right flags and validation.
package integrations

// Kind describes how an integration is connected.
type Kind string

const (
	// KindToken integrations connect with API credentials via a single POST.
	KindToken Kind = "token"
	// KindOAuth integrations require a browser flow; the CLI prints the URL.
	KindOAuth Kind = "oauth"
	// KindCluster integrations install the Kestrel operator via Helm.
	KindCluster Kind = "cluster"
	// KindCloud integrations use multi-step cloud IAM flows (AWS, OCI).
	KindCloud Kind = "cloud"
	// KindKnowledge integrations are tribal-knowledge sources (Confluence,
	// Jira, Linear, Notion, Glean) managed via /api/tribal-knowledge/sources.
	KindKnowledge Kind = "knowledge"
)

// Field is one credential/config input for a token or knowledge integration.
type Field struct {
	Flag     string // CLI flag name, kebab-case (e.g. "api-token")
	JSON     string // JSON body field name expected by the server (e.g. "api_token")
	Usage    string
	Required bool
	Secret   bool // prompt hidden / redact in output
	File     bool // value may be provided via --<flag>-file
}

// Spec describes one integration.
type Spec struct {
	Key         string
	Name        string
	Kind        Kind
	Description string

	// Token integrations: endpoint paths.
	ConnectPath    string
	DisconnectPath string
	TestPath       string

	// Token/knowledge integrations: accepted fields.
	Fields []Field

	// Knowledge integrations: the source_type value.
	SourceType string

	// Extra hint printed after a successful connect (e.g. webhook setup).
	PostConnectHint string
}

// Registry lists every integration the CLI can manage, keyed by Spec.Key.
var Registry = []Spec{
	// --- OAuth / browser flows ---
	{
		Key: "github", Name: "GitHub", Kind: KindOAuth,
		Description: "Pull request automation and CI/CD triggers (GitHub App install)",
	},
	{
		Key: "gitlab", Name: "GitLab", Kind: KindOAuth,
		Description: "Merge request automation and pipeline triggers (OAuth)",
	},
	{
		Key: "slack", Name: "Slack", Kind: KindOAuth,
		Description: "Incident alerts, approvals, and AI responses in Slack (app install)",
	},

	// --- Cluster ---
	{
		Key: "kubernetes", Name: "Kubernetes", Kind: KindCluster,
		Description: "Onboard a cluster by installing the Kestrel operator via Helm",
	},

	// --- Cloud IAM flows ---
	{
		Key: "aws", Name: "AWS", Kind: KindCloud,
		Description: "Connect an AWS account via IAM role (bootstrap + verify)",
	},
	{
		Key: "oci", Name: "Oracle Cloud (OCI)", Kind: KindCloud,
		Description: "Connect an OCI tenancy with API-key auth",
	},

	// --- Token integrations ---
	{
		Key: "cloudflare", Name: "Cloudflare", Kind: KindToken,
		Description:    "Zones, Workers, DNS, WAF, and tunnels",
		ConnectPath:    "/api/integrations/cloudflare/connect",
		DisconnectPath: "/api/integrations/cloudflare/disconnect",
		TestPath:       "/api/integrations/cloudflare/test",
		Fields: []Field{
			{Flag: "api-token", JSON: "api_token", Usage: "Cloudflare API token", Required: true, Secret: true},
			{Flag: "account-id", JSON: "account_id", Usage: "Cloudflare account ID", Required: true},
		},
	},
	{
		Key: "nebius", Name: "Nebius", Kind: KindToken,
		Description:    "Nebius AI Cloud resources and jobs",
		ConnectPath:    "/api/integrations/nebius/connect",
		DisconnectPath: "/api/integrations/nebius/disconnect",
		TestPath:       "/api/integrations/nebius/test",
		Fields: []Field{
			{Flag: "credentials", JSON: "credentials", Usage: "Service account authorized-key JSON document", Required: true, Secret: true, File: true},
			{Flag: "region", JSON: "region", Usage: "Nebius region"},
		},
	},
	{
		Key: "jenkins", Name: "Jenkins", Kind: KindToken,
		Description:    "Jenkins builds and job status",
		ConnectPath:    "/api/integrations/jenkins/connect",
		DisconnectPath: "/api/integrations/jenkins/disconnect",
		TestPath:       "/api/integrations/jenkins/test",
		Fields: []Field{
			{Flag: "base-url", JSON: "base_url", Usage: "Jenkins URL (https://jenkins.example.com)", Required: true},
			{Flag: "username", JSON: "username", Usage: "Jenkins username", Required: true},
			{Flag: "api-token", JSON: "api_token", Usage: "Jenkins API token", Required: true, Secret: true},
		},
	},
	{
		Key: "circleci", Name: "CircleCI", Kind: KindToken,
		Description:    "CircleCI workflow and job events",
		ConnectPath:    "/api/integrations/circleci/connect",
		DisconnectPath: "/api/integrations/circleci/disconnect",
		TestPath:       "/api/integrations/circleci/test",
		Fields: []Field{
			{Flag: "api-token", JSON: "api_token", Usage: "CircleCI personal API token", Required: true, Secret: true},
			{Flag: "org-slug", JSON: "org_slug", Usage: "Organization slug (e.g. gh/my-org)"},
		},
	},
	{
		Key: "terraform", Name: "Terraform Cloud", Kind: KindToken,
		Description:    "Terraform Cloud runs and plan/apply events",
		ConnectPath:    "/api/integrations/terraform/connect",
		DisconnectPath: "/api/integrations/terraform/disconnect",
		TestPath:       "/api/integrations/terraform/test",
		Fields: []Field{
			{Flag: "api-token", JSON: "api_token", Usage: "Terraform Cloud API token", Required: true, Secret: true},
			{Flag: "organization", JSON: "organization", Usage: "Terraform Cloud organization", Required: true},
			{Flag: "base-url", JSON: "base_url", Usage: "Base URL (default https://app.terraform.io)"},
		},
	},
	{
		Key: "pulumi", Name: "Pulumi Cloud", Kind: KindToken,
		Description:    "Pulumi stack updates and deployment events",
		ConnectPath:    "/api/integrations/pulumi/connect",
		DisconnectPath: "/api/integrations/pulumi/disconnect",
		TestPath:       "/api/integrations/pulumi/test",
		Fields: []Field{
			{Flag: "api-token", JSON: "api_token", Usage: "Pulumi access token", Required: true, Secret: true},
			{Flag: "organization", JSON: "organization", Usage: "Pulumi organization", Required: true},
			{Flag: "base-url", JSON: "base_url", Usage: "Base URL (default https://api.pulumi.com)"},
		},
	},
	{
		Key: "argocd", Name: "Argo CD", Kind: KindToken,
		Description:    "Argo CD application sync status",
		ConnectPath:    "/api/integrations/argocd/connect",
		DisconnectPath: "/api/integrations/argocd/disconnect",
		TestPath:       "/api/integrations/argocd/test",
		Fields: []Field{
			{Flag: "server-url", JSON: "server_url", Usage: "Argo CD server URL", Required: true},
			{Flag: "api-token", JSON: "api_token", Usage: "Argo CD API token", Required: true, Secret: true},
		},
	},
	{
		Key: "vercel", Name: "Vercel", Kind: KindToken,
		Description:    "Vercel deployments and rollbacks",
		ConnectPath:    "/api/integrations/vercel/connect",
		DisconnectPath: "/api/integrations/vercel/disconnect",
		TestPath:       "/api/integrations/vercel/test",
		Fields: []Field{
			{Flag: "api-token", JSON: "api_token", Usage: "Vercel API token", Required: true, Secret: true},
			{Flag: "team-id", JSON: "team_id", Usage: "Vercel team ID"},
		},
		PostConnectHint: "To receive deployment events, add a webhook in Vercel — run `kestrel integrations connect vercel --help` or see the Vercel integration page for the webhook URL.",
	},
	{
		Key: "railway", Name: "Railway", Kind: KindToken,
		Description:    "Railway services and deployments",
		ConnectPath:    "/api/integrations/railway/connect",
		DisconnectPath: "/api/integrations/railway/disconnect",
		TestPath:       "/api/integrations/railway/test",
		Fields: []Field{
			{Flag: "api-token", JSON: "api_token", Usage: "Railway API token", Required: true, Secret: true},
		},
	},
	{
		Key: "flyio", Name: "Fly.io", Kind: KindToken,
		Description:    "Fly.io apps, machines, and deployments",
		ConnectPath:    "/api/integrations/flyio/connect",
		DisconnectPath: "/api/integrations/flyio/disconnect",
		TestPath:       "/api/integrations/flyio/test",
		Fields: []Field{
			{Flag: "api-token", JSON: "api_token", Usage: "Fly.io API token", Required: true, Secret: true},
			{Flag: "org-slug", JSON: "org_slug", Usage: "Fly.io organization slug"},
		},
	},
	{
		Key: "beam", Name: "Beam", Kind: KindToken,
		Description:    "Beam serverless GPU workloads",
		ConnectPath:    "/api/integrations/beam/connect",
		DisconnectPath: "/api/integrations/beam/disconnect",
		TestPath:       "/api/integrations/beam/test",
		Fields: []Field{
			{Flag: "api-token", JSON: "api_token", Usage: "Beam API token", Required: true, Secret: true},
			{Flag: "gateway-base-url", JSON: "gateway_base_url", Usage: "Gateway base URL"},
		},
	},
	{
		Key: "daytona", Name: "Daytona", Kind: KindToken,
		Description:    "Daytona sandboxes and dev environments",
		ConnectPath:    "/api/integrations/daytona/connect",
		DisconnectPath: "/api/integrations/daytona/disconnect",
		TestPath:       "/api/integrations/daytona/test",
		Fields: []Field{
			{Flag: "api-key", JSON: "api_key", Usage: "Daytona API key", Required: true, Secret: true},
			{Flag: "webhook-secret", JSON: "webhook_secret", Usage: "Webhook signing secret (create a webhook endpoint in the Daytona dashboard)", Required: true, Secret: true},
			{Flag: "api-url", JSON: "api_url", Usage: "Daytona API URL"},
		},
	},
	{
		Key: "supabase", Name: "Supabase", Kind: KindToken,
		Description:    "Supabase projects, database health, and auth events",
		ConnectPath:    "/api/integrations/supabase/connect",
		DisconnectPath: "/api/integrations/supabase/disconnect",
		TestPath:       "/api/integrations/supabase/test",
		Fields: []Field{
			{Flag: "access-token", JSON: "access_token", Usage: "Supabase access token", Required: true, Secret: true},
			{Flag: "webhook-secret", JSON: "webhook_secret", Usage: "Webhook signing secret", Secret: true},
			{Flag: "api-url", JSON: "api_url", Usage: "Supabase API URL"},
		},
	},
	{
		Key: "planetscale", Name: "PlanetScale", Kind: KindToken,
		Description:    "PlanetScale branches, deploy requests, and database events",
		ConnectPath:    "/api/integrations/planetscale/connect",
		DisconnectPath: "/api/integrations/planetscale/disconnect",
		TestPath:       "/api/integrations/planetscale/test",
		Fields: []Field{
			{Flag: "token-id", JSON: "token_id", Usage: "Service token ID", Required: true},
			{Flag: "token", JSON: "token", Usage: "Service token", Required: true, Secret: true},
			{Flag: "organization", JSON: "organization", Usage: "PlanetScale organization", Required: true},
			{Flag: "webhook-secret", JSON: "webhook_secret", Usage: "Webhook signing secret", Secret: true},
		},
	},
	{
		Key: "neon", Name: "Neon", Kind: KindToken,
		Description:    "Neon Postgres projects, branches, and compute",
		ConnectPath:    "/api/integrations/neon/connect",
		DisconnectPath: "/api/integrations/neon/disconnect",
		TestPath:       "/api/integrations/neon/test",
		Fields: []Field{
			{Flag: "api-key", JSON: "api_key", Usage: "Neon API key", Required: true, Secret: true},
			{Flag: "org-id", JSON: "org_id", Usage: "Neon organization ID"},
			{Flag: "api-url", JSON: "api_url", Usage: "Neon API URL"},
		},
	},
	{
		Key: "clickhouse", Name: "ClickHouse", Kind: KindToken,
		Description:    "ClickHouse Cloud services and query performance",
		ConnectPath:    "/api/integrations/clickhouse/connect",
		DisconnectPath: "/api/integrations/clickhouse/disconnect",
		TestPath:       "/api/integrations/clickhouse/test",
		Fields: []Field{
			{Flag: "key-id", JSON: "key_id", Usage: "ClickHouse API key ID", Required: true},
			{Flag: "key-secret", JSON: "key_secret", Usage: "ClickHouse API key secret", Required: true, Secret: true},
			{Flag: "org-id", JSON: "org_id", Usage: "ClickHouse organization ID"},
			{Flag: "api-url", JSON: "api_url", Usage: "ClickHouse API URL"},
		},
	},
	{
		Key: "posthog", Name: "PostHog", Kind: KindToken,
		Description:    "PostHog product analytics, feature flags, and errors",
		ConnectPath:    "/api/integrations/posthog/connect",
		DisconnectPath: "/api/integrations/posthog/disconnect",
		TestPath:       "/api/integrations/posthog/test",
		Fields: []Field{
			{Flag: "api-key", JSON: "api_key", Usage: "PostHog personal API key", Required: true, Secret: true},
			{Flag: "project-id", JSON: "project_id", Usage: "PostHog project ID", Required: true},
			{Flag: "host", JSON: "host", Usage: "PostHog host (default https://us.posthog.com)"},
		},
	},
	{
		Key: "pagerduty", Name: "PagerDuty", Kind: KindToken,
		Description:    "Incident routing and on-call alerting",
		ConnectPath:    "/api/integrations/pagerduty/connect",
		DisconnectPath: "/api/integrations/pagerduty/disconnect",
		TestPath:       "/api/integrations/pagerduty/test",
		Fields: []Field{
			{Flag: "api-token", JSON: "api_token", Usage: "PagerDuty REST API token", Required: true, Secret: true},
			{Flag: "webhook-secret", JSON: "webhook_secret", Usage: "Webhook signing secret", Required: true, Secret: true},
		},
	},

	// --- Knowledge sources ---
	{
		Key: "confluence", Name: "Confluence", Kind: KindKnowledge, SourceType: "confluence",
		Description: "Confluence runbooks and docs for AI context",
		Fields: []Field{
			{Flag: "base-url", JSON: "base_url", Usage: "Atlassian site URL (https://your-site.atlassian.net)", Required: true},
			{Flag: "email", JSON: "api_key", Usage: "Atlassian account email", Required: true},
			{Flag: "api-token", JSON: "api_token", Usage: "Atlassian API token", Required: true, Secret: true},
		},
	},
	{
		Key: "jira", Name: "Jira", Kind: KindKnowledge, SourceType: "jira",
		Description: "Jira issues for incident context and ticket creation",
		Fields: []Field{
			{Flag: "base-url", JSON: "base_url", Usage: "Atlassian site URL (https://your-site.atlassian.net)", Required: true},
			{Flag: "email", JSON: "api_key", Usage: "Atlassian account email", Required: true},
			{Flag: "api-token", JSON: "api_token", Usage: "Atlassian API token", Required: true, Secret: true},
		},
	},
	{
		Key: "linear", Name: "Linear", Kind: KindKnowledge, SourceType: "linear",
		Description: "Linear issues for incident context and ticket creation",
		Fields: []Field{
			{Flag: "api-key", JSON: "api_key", Usage: "Linear API key", Required: true, Secret: true},
		},
	},
	{
		Key: "notion", Name: "Notion", Kind: KindKnowledge, SourceType: "notion",
		Description: "Notion pages and runbooks for AI context",
		Fields: []Field{
			{Flag: "api-token", JSON: "api_token", Usage: "Notion integration token", Required: true, Secret: true},
		},
	},
	{
		Key: "glean", Name: "Glean", Kind: KindKnowledge, SourceType: "glean",
		Description: "Company-wide knowledge search via Glean",
		Fields: []Field{
			{Flag: "api-key", JSON: "api_key", Usage: "Glean API key", Required: true, Secret: true},
			{Flag: "base-url", JSON: "base_url", Usage: "Glean instance URL"},
		},
	},
}

// Get returns the spec for a given key, or nil.
func Get(key string) *Spec {
	for i := range Registry {
		if Registry[i].Key == key {
			return &Registry[i]
		}
	}
	return nil
}
