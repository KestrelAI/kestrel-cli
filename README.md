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

kestrel requests list               # Workflow requests
kestrel requests approve <id>

kestrel apikeys list                # List API keys
kestrel apikeys create <name>       # Create new key
kestrel apikeys revoke <id>         # Revoke a key
kestrel apikeys delete <id>         # Delete permanently

kestrel mcp                         # Start MCP server for Claude/Cursor
```

## MCP Integration

The CLI includes a built-in MCP (Model Context Protocol) server for AI assistant integration:

```bash
kestrel mcp
```

This exposes 22 workflow management tools to Claude, Cursor, and other MCP-compatible AI assistants.

## License

Apache 2.0
