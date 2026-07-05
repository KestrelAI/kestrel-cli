# Kestrel CLI

Command-line interface for [Kestrel](https://usekestrel.ai) — AI Agents for Cloud Operations.

## Install

### macOS / Linux (Homebrew)
```bash
brew install KestrelAI/tap/kestrel
```

### Binary download
Download from [GitHub Releases](https://github.com/KestrelAI/kestrel-cli/releases).

### From source
```bash
go install github.com/KestrelAI/kestrel-cli@latest
```

## Authentication

### API key (recommended)
```bash
kestrel auth kestrel_sk_...
```

### Email/password
```bash
kestrel login
```

## Commands

```
kestrel workflows list              # List all workflows
kestrel workflows get <id>          # Show workflow details + DAG diagram
kestrel workflows create --file wf.json
kestrel workflows activate <id>
kestrel workflows pause <id>
kestrel workflows test <id>         # Dry-run execution
kestrel workflows generate "..."    # AI-generate from description
kestrel workflows executions <id>   # List executions
kestrel workflows stats             # Aggregate statistics

kestrel approvals list              # Pending approval gates
kestrel approvals approve <id>
kestrel approvals reject <id>
kestrel approvals request-changes <id>  # Re-run RCA with feedback (refine loop)

kestrel requests list               # Workflow requests
kestrel requests approve <id>

kestrel apikeys list                # List API keys
kestrel apikeys create <name>       # Create new key
kestrel apikeys revoke <id>         # Revoke a key
kestrel apikeys delete <id>         # Delete permanently

kestrel integrations list           # All integrations + connection status
kestrel integrations connect <name> # Connect an integration (flags vary per integration)
kestrel integrations test <name>    # Test an integration's credentials
kestrel integrations disconnect <name>

kestrel mcp                         # Start MCP server for Claude/Cursor
```

### Connecting integrations

Each integration has its own authentication requirements — run
`kestrel integrations connect <name> --help` to see the exact flags.

```bash
# Token integrations (credentials via flags; secrets prompt interactively if omitted)
kestrel integrations connect cloudflare --api-token <token> --account-id <id>
kestrel integrations connect pagerduty --api-token <token> --webhook-secret <secret>

# Knowledge sources
kestrel integrations connect confluence --base-url https://acme.atlassian.net --email me@acme.com --api-token <token>

# OAuth integrations print a browser URL to finish the flow
kestrel integrations connect github

# Kubernetes generates the operator token + Helm values file and prints the install command
kestrel integrations connect kubernetes --cluster-name prod

# AWS is a two-step IAM role flow
kestrel integrations connect aws                 # step 1: prints CloudFormation/CLI instructions
kestrel integrations connect aws --role-arn <arn> --external-id <id>  # step 2: verify
```

## MCP Integration

The CLI includes a built-in MCP (Model Context Protocol) server for AI assistant integration:

```bash
kestrel mcp
```

This exposes 26 workflow and integration management tools to Claude, Cursor, and other MCP-compatible AI assistants — including `list_integrations`, `connect_integration`, `test_integration`, and `disconnect_integration`.

## License

Apache 2.0
