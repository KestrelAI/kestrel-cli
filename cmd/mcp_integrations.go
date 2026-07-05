package cmd

import (
	"context"
	"fmt"
	"strings"

	"cli/pkg/api"
	"cli/pkg/integrations"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerIntegrationTools adds MCP tools for connecting, testing, and
// disconnecting integrations, driven by the shared integration registry.
func registerIntegrationTools(server *mcp.Server, client *api.Client) {
	type noArgs struct{}

	// --- list_integrations ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_integrations",
		Description: "List every Kestrel integration with its type (token, oauth, cluster, cloud, knowledge), required credential fields, and current connection status. Call this before connect_integration to learn which credentials are needed.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args noArgs) (*mcp.CallToolResult, any, error) {
		statuses, err := client.GetIntegrationsStatus()
		if err != nil {
			return errResult(err)
		}
		connected := map[string]bool{}
		for _, s := range statuses {
			connected[s.ID] = s.Connected
		}

		type fieldInfo struct {
			Name     string `json:"name"`
			Usage    string `json:"usage"`
			Required bool   `json:"required"`
			Secret   bool   `json:"secret"`
		}
		type integrationInfo struct {
			Key         string      `json:"key"`
			Name        string      `json:"name"`
			Kind        string      `json:"kind"`
			Description string      `json:"description"`
			Connected   bool        `json:"connected"`
			Fields      []fieldInfo `json:"credential_fields,omitempty"`
			Note        string      `json:"note,omitempty"`
		}

		out := make([]integrationInfo, 0, len(integrations.Registry))
		for _, spec := range integrations.Registry {
			info := integrationInfo{
				Key: spec.Key, Name: spec.Name, Kind: string(spec.Kind),
				Description: spec.Description, Connected: connected[spec.Key],
			}
			for _, f := range spec.Fields {
				info.Fields = append(info.Fields, fieldInfo{
					Name: f.JSON, Usage: f.Usage, Required: f.Required, Secret: f.Secret,
				})
			}
			switch spec.Kind {
			case integrations.KindOAuth:
				info.Note = "Requires a browser step: connect_integration returns a URL the user must open."
			case integrations.KindCluster:
				info.Note = "Use the CLI: kestrel integrations connect kubernetes --cluster-name <name> (writes Helm values and prints the install command)."
			case integrations.KindCloud:
				info.Note = "Multi-step IAM flow — use the CLI: kestrel integrations connect " + spec.Key + " --help."
			}
			out = append(out, info)
		}
		return jsonResult(out)
	})

	// --- connect_integration ---
	type connectArgs struct {
		Name        string            `json:"name" jsonschema:"Integration key from list_integrations (e.g. cloudflare, pagerduty, confluence, github)"`
		Credentials map[string]string `json:"credentials,omitempty" jsonschema:"Credential fields keyed by the field names from list_integrations (e.g. api_token, account_id). Never invent values; ask the user for them."`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "connect_integration",
		Description: "Connect a token or knowledge-source integration using credentials, or get the browser URL for OAuth integrations (GitHub, GitLab, Slack). Check list_integrations first for required credential fields. Kubernetes/AWS/OCI must be connected via the kestrel CLI instead.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args connectArgs) (*mcp.CallToolResult, any, error) {
		spec := integrations.Get(strings.ToLower(args.Name))
		if spec == nil {
			return errResult(fmt.Errorf("unknown integration %q — call list_integrations for valid keys", args.Name))
		}

		switch spec.Kind {
		case integrations.KindToken, integrations.KindKnowledge:
			body := map[string]interface{}{}
			var missing []string
			for _, f := range spec.Fields {
				val := args.Credentials[f.JSON]
				if val == "" && f.Required {
					missing = append(missing, f.JSON)
				}
				if val != "" {
					body[f.JSON] = val
				}
			}
			if len(missing) > 0 {
				return errResult(fmt.Errorf("missing required credentials for %s: %s — ask the user for these values", spec.Name, strings.Join(missing, ", ")))
			}

			if spec.Kind == integrations.KindKnowledge {
				body["source_type"] = spec.SourceType
				body["name"] = spec.Name
				body["enabled"] = true
				src, err := client.CreateKnowledgeSource(body)
				if err != nil {
					return errResult(fmt.Errorf("connect %s: %w", spec.Name, err))
				}
				if out, tErr := client.TestKnowledgeSource(src.ID); tErr != nil {
					return textResult(fmt.Sprintf("%s connected, but the connection test failed: %v — verify the credentials.", spec.Name, tErr))
				} else if msg, ok := out["message"].(string); ok && msg != "" {
					return textResult(fmt.Sprintf("%s connected and verified: %s", spec.Name, msg))
				}
				return textResult(fmt.Sprintf("%s connected and verified.", spec.Name))
			}

			if _, err := client.ConnectIntegration(spec.ConnectPath, body); err != nil {
				return errResult(fmt.Errorf("connect %s: %w", spec.Name, err))
			}
			msg := fmt.Sprintf("%s connected.", spec.Name)
			if spec.PostConnectHint != "" {
				msg += " " + spec.PostConnectHint
			}
			return textResult(msg)

		case integrations.KindOAuth:
			var (
				authURL string
				err     error
			)
			switch spec.Key {
			case "github":
				authURL, err = client.GetGitHubInstallURL()
			case "gitlab":
				authURL, err = client.GetGitLabAuthURL()
			case "slack":
				authURL, err = client.GetSlackInstallURL()
			}
			if err != nil {
				return errResult(fmt.Errorf("get %s authorization URL: %w", spec.Name, err))
			}
			return textResult(fmt.Sprintf("%s requires a browser step. Ask the user to open this URL and complete the authorization, then verify with list_integrations:\n\n%s", spec.Name, authURL))

		default:
			return errResult(fmt.Errorf("%s must be connected via the kestrel CLI: kestrel integrations connect %s --help", spec.Name, spec.Key))
		}
	})

	// --- test_integration ---
	type nameArg struct {
		Name string `json:"name" jsonschema:"Integration key (e.g. cloudflare, confluence)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "test_integration",
		Description: "Test a connected integration's credentials. Works for token integrations and knowledge sources.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args nameArg) (*mcp.CallToolResult, any, error) {
		spec := integrations.Get(strings.ToLower(args.Name))
		if spec == nil {
			return errResult(fmt.Errorf("unknown integration %q", args.Name))
		}
		switch spec.Kind {
		case integrations.KindToken:
			if spec.TestPath == "" {
				return errResult(fmt.Errorf("%s does not support connection tests", spec.Name))
			}
			out, err := client.TestIntegration(spec.TestPath)
			if err != nil {
				return errResult(fmt.Errorf("%s test failed: %w", spec.Name, err))
			}
			return jsonResult(out)
		case integrations.KindKnowledge:
			sources, err := client.ListKnowledgeSources()
			if err != nil {
				return errResult(err)
			}
			for _, s := range sources {
				if s.SourceType == spec.SourceType {
					out, err := client.TestKnowledgeSource(s.ID)
					if err != nil {
						return errResult(fmt.Errorf("%s test failed: %w", spec.Name, err))
					}
					return jsonResult(out)
				}
			}
			return errResult(fmt.Errorf("no %s source is connected", spec.Name))
		default:
			return errResult(fmt.Errorf("%s does not support tests — check status with list_integrations", spec.Name))
		}
	})

	// --- disconnect_integration ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "disconnect_integration",
		Description: "Disconnect a token integration or knowledge source. OAuth, cluster, and cloud integrations must be disconnected in the Kestrel UI.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args nameArg) (*mcp.CallToolResult, any, error) {
		spec := integrations.Get(strings.ToLower(args.Name))
		if spec == nil {
			return errResult(fmt.Errorf("unknown integration %q", args.Name))
		}
		switch spec.Kind {
		case integrations.KindToken:
			if spec.DisconnectPath == "" {
				return errResult(fmt.Errorf("%s cannot be disconnected via MCP", spec.Name))
			}
			if err := client.DisconnectIntegration(spec.DisconnectPath); err != nil {
				return errResult(err)
			}
			return textResult(fmt.Sprintf("%s disconnected.", spec.Name))
		case integrations.KindKnowledge:
			sources, err := client.ListKnowledgeSources()
			if err != nil {
				return errResult(err)
			}
			for _, s := range sources {
				if s.SourceType == spec.SourceType {
					if err := client.DeleteKnowledgeSource(s.ID); err != nil {
						return errResult(err)
					}
					return textResult(fmt.Sprintf("%s disconnected.", spec.Name))
				}
			}
			return errResult(fmt.Errorf("no %s source is connected", spec.Name))
		default:
			return errResult(fmt.Errorf("%s must be disconnected in the Kestrel UI", spec.Name))
		}
	})
}
