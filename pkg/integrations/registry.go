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

	// SetupHelp explains where to create the credentials this integration
	// needs (mirrors the platform UI instructions). Printed before interactive
	// credential prompts and included in `connect <name> --help`.
	SetupHelp string

	// Extra hint printed after a successful connect (e.g. webhook setup).
	// `{server}` is replaced with the configured Kestrel server URL.
	PostConnectHint string

	// WebhookSecretPath is the endpoint that saves vendor-generated webhook
	// signing secret(s) after connect, for integrations where the third party
	// generates the secret (e.g. PlanetScale, Vercel). Enables
	// `kestrel integrations webhook-secret <name>`.
	WebhookSecretPath string
}

// Registry lists every integration the CLI can manage, keyed by Spec.Key.
var Registry = []Spec{
	// --- OAuth / browser flows ---
	{
		Key: "github", Name: "GitHub", Kind: KindOAuth,
		Description: "Pull request automation and CI/CD triggers (GitHub App install)",
		SetupHelp:   "Connecting GitHub installs the Kestrel GitHub App. The CLI prints an install URL — open it in your browser, pick the org/repos to grant, and approve.",
	},
	{
		Key: "gitlab", Name: "GitLab", Kind: KindOAuth,
		Description: "Merge request automation and pipeline triggers (OAuth)",
		SetupHelp:   "Connecting GitLab uses an OAuth flow. The CLI prints an authorization URL — open it in your browser and approve access.",
	},
	{
		Key: "slack", Name: "Slack", Kind: KindOAuth,
		Description: "Incident alerts, approvals, and AI responses in Slack (app install)",
		SetupHelp:   "Connecting Slack installs the Kestrel Slack app. The CLI prints an install URL — open it in your browser, pick the workspace, and approve.",
	},

	// --- Cluster ---
	{
		Key: "kubernetes", Name: "Kubernetes", Kind: KindCluster,
		Description: "Onboard a cluster by installing the Kestrel operator via Helm",
		SetupHelp: `No credentials needed up front: the CLI mints an operator token, writes a Helm values
file, and prints the 'helm install' command to run against your cluster.
Optional flags mirror the UI's operator credential form:
  --flow-source cilium|istio|aws-vpc-cni   network flow collection source (default cilium)
  --metrics-source none|opentelemetry|datadog   metrics source (default: K8s Metrics Server)
  --datadog-namespace <ns>                 where Datadog runs (default datadog)
  --argocd [--argocd-namespace <ns>]       enable ArgoCD GitOps deployment sync
  --workload-count <n>                     sizes operator CPU/memory; get it with:
      kubectl get deploy,sts,ds,job,cj -A --no-headers | wc -l
  --safe-apply                             allow approved YAML changes to be applied`,
	},

	// --- Cloud IAM flows ---
	{
		Key: "aws", Name: "AWS", Kind: KindCloud,
		Description: "Connect an AWS account via IAM role (bootstrap + verify)",
		SetupHelp: `Two steps: (1) run without flags to bootstrap — the CLI prints a CloudFormation/IAM role
setup with an External ID; (2) create the role in your AWS account, then re-run with
--role-arn and --external-id to verify. No long-lived AWS keys are stored.`,
	},
	{
		Key: "oci", Name: "Oracle Cloud (OCI)", Kind: KindCloud,
		Description: "Connect an OCI tenancy with API-key auth",
		SetupHelp: `API key: OCI Console -> Profile -> My profile -> API keys -> Add API key. Download the
private key (PEM) and note the fingerprint, tenancy OCID, and user OCID from the
configuration file preview it shows.`,
	},

	// --- Token integrations ---
	{
		Key: "cloudflare", Name: "Cloudflare", Kind: KindToken,
		Description:    "Zones, Workers, DNS, WAF, and tunnels",
		ConnectPath:    "/api/integrations/cloudflare/connect",
		DisconnectPath: "/api/integrations/cloudflare/disconnect",
		TestPath:       "/api/integrations/cloudflare/test",
		SetupHelp: `API token: Cloudflare dashboard -> My Profile -> API Tokens -> Create Token -> Custom Token.
  Grant: Zone (DNS Edit, Zone Settings Edit, Zone Read, Analytics Read, Firewall Services
  Edit, Health Checks Edit) and Account (Workers Scripts Edit, Workers KV Storage Edit,
  Account Analytics Read, Account Settings Read). Scope Account Resources to your account.
Account ID: in the dashboard URL — dash.cloudflare.com/<account-id>/home.`,
		PostConnectHint: "To receive alerts, add a webhook in Cloudflare: Manage account -> Notifications -> Destinations -> Webhooks -> Create, with URL {server}/api/webhooks/cloudflare and the webhook secret printed above, then route notifications to it.",
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
		SetupHelp: `Authorized key: Nebius console -> Administration -> IAM -> Service accounts -> your account ->
  Keys tab -> Authorized keys (NOT Access keys) -> Upload authorized key.
  Or via CLI: nebius iam auth-public-key generate --service-account-id <id> --output authorized-key.json
Provide the full authorized-key JSON document (use --credentials-file to read it from a file).`,
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
		SetupHelp: `API token: Jenkins -> your user (top right) -> Security -> API Token -> Add new token.
The user needs permission to read jobs and trigger builds. Auth is username + token (HTTP Basic).
The Jenkins URL must be reachable from the Kestrel server.`,
		PostConnectHint: "Optional: to receive build events, add a webhook via the Notification plugin (job -> Configure -> Job Notifications) posting JSON to {server}/api/webhooks/jenkins with the webhook secret printed above in the X-Kestrel-Webhook-Secret header.",
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
		SetupHelp: `API token: CircleCI -> User Settings -> Personal API Tokens -> Create New Token.
Org slug (optional): Organization Settings -> Overview (e.g. gh/my-org).`,
		PostConnectHint: "To receive workflow/job events, add a webhook per project: Project Settings -> Webhooks -> Add Webhook with URL {server}/api/webhooks/circleci, the signing secret printed above, and the Workflow Completed + Job Completed events.",
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
		SetupHelp: `API token: Terraform Cloud -> Organization Settings -> API Tokens -> Team Tokens (recommended),
or a user token under Account Settings -> Tokens.
Organization: the name in your URL — app.terraform.io/app/<organization>.`,
		PostConnectHint: "To receive run events, add a notification per workspace: workspace -> Settings -> Notifications -> Create a Notification -> Webhook, URL {server}/api/webhooks/terraform, the token printed above, and select the run events. Repeat for each workspace.",
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
		SetupHelp: `Access token: Pulumi Cloud -> Organization Settings -> Access Tokens (recommended),
or a personal token under Personal Settings -> Access Tokens (pul-...).
Organization: the name in your URL — app.pulumi.com/<organization>.`,
		PostConnectHint: "To receive stack/deployment events, add a webhook: org Settings -> Integrations -> Webhooks -> Add webhook (destination: Webhook), payload URL {server}/api/webhooks/pulumi, the secret printed above, and check all trigger groups.",
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
		SetupHelp: `API token: generate one in Argo CD -> Settings -> Accounts -> Generate New Token.
The Argo CD server URL must be reachable from the Kestrel server.`,
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
		SetupHelp: `API token: Vercel -> your avatar -> Account Settings -> Tokens. Scope it to your team for
team-level access. Team ID (optional, team_...): Vercel -> Settings -> General; leave blank
for personal accounts.`,
		PostConnectHint:   "To receive deployment events, add a webhook in Vercel: Settings -> Webhooks -> Create Webhook with URL {server}/api/webhooks/vercel and the deployment/alert events, then save the signing secret (shown once) with: kestrel integrations webhook-secret vercel",
		WebhookSecretPath: "/api/integrations/vercel/webhook-secret",
		Fields: []Field{
			{Flag: "api-token", JSON: "api_token", Usage: "Vercel API token", Required: true, Secret: true},
			{Flag: "team-id", JSON: "team_id", Usage: "Vercel team ID"},
		},
	},
	{
		Key: "railway", Name: "Railway", Kind: KindToken,
		Description:    "Railway services and deployments",
		ConnectPath:    "/api/integrations/railway/connect",
		DisconnectPath: "/api/integrations/railway/disconnect",
		TestPath:       "/api/integrations/railway/test",
		SetupHelp: `API token: railway.com/account/tokens (Account Settings -> Tokens). Leave the Workspace
dropdown at "No workspace" to mint an account-scoped token that can read your projects,
services, deployments, and logs.`,
		PostConnectHint:   "To receive deployment events, choose a secret and save it with `kestrel integrations webhook-secret railway`, then add a webhook per project: Project -> Settings -> Webhooks with URL {server}/api/webhooks/railway?secret=<your-secret> (Railway doesn't sign webhooks).",
		WebhookSecretPath: "/api/integrations/railway/webhook-secret",
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
		SetupHelp: `API token: run 'fly tokens create org' (or create a token in the Fly.io dashboard) with
read access to your organization.
Org slug: find with 'fly orgs list' — use the slug, not the display name (default: personal).
No webhooks needed — Kestrel polls the Fly Machines API.`,
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
		SetupHelp: `API token: Beam dashboard -> Settings -> API Keys & Workspace ID -> Create Key.
Or via CLI: pip install beam-client && beam config create kestrel, then copy the token
from ~/.beam/config.ini. Permissions: Full Access, or Restricted with Read+Write+Delete
on Deployments, Containers, Tasks, Machines and Read on Images, Volumes, Logs.
No webhooks needed — Kestrel polls the Beam API.`,
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
		SetupHelp: `API key: Daytona dashboard -> API Keys -> Create API Key (shown once). Permissions: Full
Access, or Read+Write+Delete on Sandboxes and Snapshots plus Read on Volumes.
Webhook secret (required): Daytona dashboard -> Webhooks -> Enable webhooks -> Create
Endpoint with your Kestrel webhook URL (<server>/api/webhooks/daytona) and all sandbox/
snapshot/volume events, then open the endpoint and copy its Signing Secret (whsec_...).`,
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
		SetupHelp: `Access token: Supabase dashboard -> Account -> Access Tokens -> Generate new token (sbp_...).
Kestrel polls the Management API for project health, backups, replicas, and branches.`,
		PostConnectHint:   "Optional: for row-level DB events, add a Database Webhook per project (Integrations -> Database Webhooks) posting to {server}/api/webhooks/supabase with your secret in the X-Supabase-Webhook-Secret header, then save it with: kestrel integrations webhook-secret supabase",
		WebhookSecretPath: "/api/integrations/supabase/webhook-secret",
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
		SetupHelp: `Service token: app.planetscale.com -> Settings -> Service tokens -> New service token
(or 'pscale service-token create'). You get a token ID + token (pscale_tkn_...).
Permissions: org-level read_databases (required), plus branch/deploy-request/backup
permissions on the databases you want Kestrel to manage.`,
		PostConnectHint:   "To receive branch/deploy events, add a webhook per database: Settings -> Webhooks -> Add webhook with URL {server}/api/webhooks/planetscale. PlanetScale shows a unique signing secret per webhook — save each with: kestrel integrations webhook-secret planetscale",
		WebhookSecretPath: "/api/integrations/planetscale/webhook-secret",
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
		SetupHelp: `API key: Neon Console (console.neon.tech) -> Account settings -> API keys -> Create new
API key (napi_..., shown once). Personal or organization keys both work.
Org ID (org-...): auto-detected when the key sees one org; for multiple orgs find it in
the Console URL (console.neon.tech/app/org-.../projects) or Organization settings.
No webhooks needed — Kestrel polls the Neon API.`,
		Fields: []Field{
			{Flag: "api-key", JSON: "api_key", Usage: "Neon API key", Required: true, Secret: true},
			{Flag: "org-id", JSON: "org_id", Usage: "Neon organization ID (auto-detected for keys with a single org; required if the key can see multiple)"},
			{Flag: "api-url", JSON: "api_url", Usage: "Neon API URL"},
		},
	},
	{
		Key: "clickhouse", Name: "ClickHouse", Kind: KindToken,
		Description:    "ClickHouse Cloud services and query performance",
		ConnectPath:    "/api/integrations/clickhouse/connect",
		DisconnectPath: "/api/integrations/clickhouse/disconnect",
		TestPath:       "/api/integrations/clickhouse/test",
		SetupHelp: `API key: ClickHouse Cloud console (console.clickhouse.cloud) -> click your organization
name (bottom left) -> API keys -> New API key. Assign the Admin role (Developer keys are
read-only). Key ID and secret are shown once. Org ID is auto-detected from the key.
No webhooks needed — Kestrel polls the ClickHouse Cloud API.`,
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
		SetupHelp: `Personal API key: PostHog -> Settings -> Personal API Keys (phx_...). Required scopes:
project:read, session_recording:write, query:read, error_tracking:read.
Project ID: PostHog -> Settings. Host: us.posthog.com / eu.posthog.com / self-hosted.`,
		PostConnectHint: "To receive events (exceptions, rage clicks), add a destination: PostHog -> Data -> Destinations -> New Destination -> HTTP Webhook posting to {server}/api/webhooks/posthog.",
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
		SetupHelp: `API token: PagerDuty -> your user icon -> My Profile -> User Settings -> Create API User
Token. Must be a USER-level token — account-level API Access Keys will not work.
Webhook secret (required): PagerDuty -> Integrations -> Generic Webhooks (V3) -> New
Webhook with your Kestrel webhook URL (<server>/api/webhooks/pagerduty), Scope Type =
Account, subscribed to incident events; copy the signing secret it shows.`,
		Fields: []Field{
			{Flag: "api-token", JSON: "api_token", Usage: "PagerDuty REST API token", Required: true, Secret: true},
			{Flag: "webhook-secret", JSON: "webhook_secret", Usage: "Webhook signing secret", Required: true, Secret: true},
		},
	},

	{
		Key: "vault", Name: "HashiCorp Vault", Kind: KindToken,
		Description:    "Vault KV secrets, policies, leases, and rotation",
		ConnectPath:    "/api/integrations/vault/connect",
		DisconnectPath: "/api/integrations/vault/disconnect",
		TestPath:       "/api/integrations/vault/test",
		SetupHelp: `Token: create a periodic, renewable token bound to a dedicated Kestrel policy, e.g.
  vault token create -policy=kestrel -period=768h
Grant the policy read/list on sys/health, sys/mounts, sys/policies/acl, sys/auth, and the
KV metadata/ paths you want monitored, plus write only where workflows need it.
Namespace (Enterprise/HCP only): HCP Vault uses "admin"; leave blank for OSS Vault.
No webhooks needed — Kestrel polls the Vault API (secret values are never read by triggers).`,
		Fields: []Field{
			{Flag: "address", JSON: "address", Usage: "Vault address (https://vault.example.com:8200)", Required: true},
			{Flag: "token", JSON: "token", Usage: "Vault token", Required: true, Secret: true},
			{Flag: "namespace", JSON: "namespace", Usage: "Vault namespace (Enterprise/HCP; e.g. admin)"},
		},
	},
	{
		Key: "infisical", Name: "Infisical", Kind: KindToken,
		Description:    "Infisical secrets, approvals, and secret syncs",
		ConnectPath:    "/api/integrations/infisical/connect",
		DisconnectPath: "/api/integrations/infisical/disconnect",
		TestPath:       "/api/integrations/infisical/test",
		SetupHelp: `Machine Identity: Infisical -> Organization -> Access Control -> Machine Identities -> Create
identity, then add it to the projects you want to automate. Open its Universal Auth method and
copy the Client ID (not the identity's ID) and create a Client Secret.
Site URL: leave blank for Infisical Cloud US; use https://eu.infisical.com for EU Cloud
or your own URL for self-hosted.
No webhooks needed — Kestrel polls the Infisical audit log and project APIs (secret-change
triggers need a paid plan for audit log access; secret values are never read by triggers).`,
		Fields: []Field{
			{Flag: "client-id", JSON: "client_id", Usage: "Machine Identity client ID", Required: true},
			{Flag: "client-secret", JSON: "client_secret", Usage: "Machine Identity client secret", Required: true, Secret: true},
			{Flag: "site-url", JSON: "site_url", Usage: "Site URL (default https://app.infisical.com)"},
		},
	},
	{
		Key: "sonarcloud", Name: "SonarCloud", Kind: KindToken,
		Description:    "SonarCloud code quality gates, issues, and security hotspots",
		ConnectPath:    "/api/integrations/sonarcloud/connect",
		DisconnectPath: "/api/integrations/sonarcloud/disconnect",
		TestPath:       "/api/integrations/sonarcloud/test",
		SetupHelp: `API token: sonarcloud.io -> your avatar -> My Account -> Access Tokens -> Generate Token.
Organization key: sonarcloud.io -> your organization -> the key in the URL
(sonarcloud.io/organizations/<key>) or Administration -> Organization settings.`,
		PostConnectHint: "To receive analysis events, add a webhook in SonarCloud: open your organization and select Webhooks in the left sidebar (for a single project: Administration -> Webhooks), then Create, with URL {server}/api/webhooks/sonarcloud and the secret printed above (SonarCloud signs deliveries with it via X-Sonar-Webhook-HMAC-SHA256).",
		Fields: []Field{
			{Flag: "organization", JSON: "organization", Usage: "SonarCloud organization key", Required: true},
			{Flag: "api-token", JSON: "api_token", Usage: "SonarCloud API token", Required: true, Secret: true},
		},
	},
	{
		Key: "okta", Name: "Okta", Kind: KindToken,
		Description:    "Okta identity security: users, groups, sessions, and System Log triggers",
		ConnectPath:    "/api/integrations/okta/connect",
		DisconnectPath: "/api/integrations/okta/disconnect",
		TestPath:       "/api/integrations/okta/test",
		SetupHelp: `API token (SSWS): Okta Admin Console -> Security -> API -> Tokens -> Create token.
Org URL: your Okta domain, e.g. https://your-org.okta.com (find it in the Admin Console header).
The token inherits the permissions of the admin who created it; a read-only admin token
works for triggers and read blocks, while lifecycle/session blocks need a super admin
or org admin token.
No webhooks needed — Kestrel polls the Okta System Log for security events.`,
		Fields: []Field{
			{Flag: "org-url", JSON: "org_url", Usage: "Okta org URL (https://your-org.okta.com)", Required: true},
			{Flag: "api-token", JSON: "api_token", Usage: "Okta API token (SSWS)", Required: true, Secret: true},
		},
	},

	// --- Knowledge sources ---
	{
		Key: "confluence", Name: "Confluence", Kind: KindKnowledge, SourceType: "confluence",
		Description: "Confluence runbooks and docs for AI context",
		SetupHelp: `API token: id.atlassian.com/manage-profile/security/api-tokens -> Create API token.
Auth is your Atlassian account email + the token (Basic auth).`,
		Fields: []Field{
			{Flag: "base-url", JSON: "base_url", Usage: "Atlassian site URL (https://your-site.atlassian.net)", Required: true},
			{Flag: "email", JSON: "api_key", Usage: "Atlassian account email", Required: true},
			{Flag: "api-token", JSON: "api_token", Usage: "Atlassian API token", Required: true, Secret: true},
		},
	},
	{
		Key: "jira", Name: "Jira", Kind: KindKnowledge, SourceType: "jira",
		Description: "Jira issues for incident context and ticket creation",
		SetupHelp: `API token: id.atlassian.com/manage-profile/security/api-tokens -> Create API token.
Auth is your Atlassian account email + the token (Basic auth).`,
		Fields: []Field{
			{Flag: "base-url", JSON: "base_url", Usage: "Atlassian site URL (https://your-site.atlassian.net)", Required: true},
			{Flag: "email", JSON: "api_key", Usage: "Atlassian account email", Required: true},
			{Flag: "api-token", JSON: "api_token", Usage: "Atlassian API token", Required: true, Secret: true},
		},
	},
	{
		Key: "linear", Name: "Linear", Kind: KindKnowledge, SourceType: "linear",
		Description: "Linear issues for incident context and ticket creation",
		SetupHelp: `API key: linear.app/settings/account/security -> Personal API keys -> New API Key.
Personal keys inherit your permissions — consider a service account for production.`,
		Fields: []Field{
			{Flag: "api-key", JSON: "api_key", Usage: "Linear API key", Required: true, Secret: true},
		},
	},
	{
		Key: "notion", Name: "Notion", Kind: KindKnowledge, SourceType: "notion",
		Description: "Notion pages and runbooks for AI context",
		SetupHelp: `Integration token: notion.so/my-integrations -> New integration -> select workspace ->
copy the Internal Integration Secret. Then share each page/database you want searchable
with the integration (Notion pages are not visible to integrations by default).`,
		Fields: []Field{
			{Flag: "api-token", JSON: "api_token", Usage: "Notion integration token", Required: true, Secret: true},
		},
	},
	{
		Key: "glean", Name: "Glean", Kind: KindKnowledge, SourceType: "glean",
		Description: "Company-wide knowledge search via Glean",
		SetupHelp: `API key: Glean Admin Console -> API -> API Keys -> Create API Key with the search:read
scope. API access requires an Enterprise plan — ask your Glean administrator to enable it.`,
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
