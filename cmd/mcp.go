package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"cli/pkg/api"
	"cli/pkg/config"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the Kestrel MCP server (for Claude, Cursor, etc.)",
	Long: `Start a Model Context Protocol (MCP) server over stdio.
This allows LLM agents in Cursor, Claude Desktop, or other MCP clients
to interact with Kestrel workflows programmatically.

Configure in your MCP client:
  {
    "mcpServers": {
      "kestrel": {
        "command": "kestrel",
        "args": ["mcp"]
      }
    }
  }`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if !cfg.IsLoggedIn() {
			return fmt.Errorf("not logged in — run `kestrel login` first")
		}
		client, err := api.NewFromConfig(cfg)
		if err != nil {
			return err
		}

		server := mcp.NewServer(&mcp.Implementation{
			Name:    "kestrel-workflows",
			Version: "0.1.0",
		}, nil)

		registerTools(server, client, cfg)

		if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			log.Printf("MCP server error: %v", err)
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

func jsonResult(v interface{}) (*mcp.CallToolResult, any, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("marshal: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func textResult(msg string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}, nil, nil
}

func errResult(err error) (*mcp.CallToolResult, any, error) {
	return nil, nil, err
}

func registerTools(server *mcp.Server, client *api.Client, cfg *config.Config) {
	// --- list_workflows ---
	type listWorkflowsArgs struct {
		Status string `json:"status,omitempty" jsonschema:"Filter by status: draft, active, paused, or archived"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_workflows",
		Description: "List all Kestrel workflows. Optionally filter by status.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args listWorkflowsArgs) (*mcp.CallToolResult, any, error) {
		result, err := client.ListWorkflows(args.Status)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})

	// --- get_workflow ---
	type workflowIDArg struct {
		WorkflowID string `json:"workflow_id" jsonschema:"The workflow UUID"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_workflow",
		Description: "Get detailed information about a specific workflow including its definition, trigger config, and stats.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args workflowIDArg) (*mcp.CallToolResult, any, error) {
		result, err := client.GetWorkflow(args.WorkflowID)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})

	// --- create_workflow ---
	type createWorkflowArgs struct {
		Name          string          `json:"name" jsonschema:"Workflow name"`
		Description   string          `json:"description" jsonschema:"Workflow description"`
		Definition    json.RawMessage `json:"definition" jsonschema:"Workflow definition with nodes and edges"`
		TriggerConfig json.RawMessage `json:"trigger_config" jsonschema:"Trigger configuration"`
		NLPrompt      string          `json:"nl_prompt,omitempty" jsonschema:"Original natural language prompt"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_workflow",
		Description: "Create a new workflow from a structured definition.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args createWorkflowArgs) (*mcp.CallToolResult, any, error) {
		result, err := client.CreateWorkflow(api.CreateWorkflowRequest{
			Name: args.Name, Description: args.Description,
			Definition: args.Definition, TriggerConfig: args.TriggerConfig,
			NLPrompt: args.NLPrompt,
		})
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})

	// --- update_workflow ---
	type updateWorkflowArgs struct {
		WorkflowID    string          `json:"workflow_id" jsonschema:"The workflow UUID to update"`
		Name          string          `json:"name" jsonschema:"Updated name"`
		Description   string          `json:"description" jsonschema:"Updated description"`
		Definition    json.RawMessage `json:"definition" jsonschema:"Updated definition"`
		TriggerConfig json.RawMessage `json:"trigger_config" jsonschema:"Updated trigger config"`
		NLPrompt      string          `json:"nl_prompt,omitempty"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_workflow",
		Description: "Update an existing workflow's name, description, or definition.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args updateWorkflowArgs) (*mcp.CallToolResult, any, error) {
		result, err := client.UpdateWorkflow(args.WorkflowID, api.UpdateWorkflowRequest{
			Name: args.Name, Description: args.Description,
			Definition: args.Definition, TriggerConfig: args.TriggerConfig,
			NLPrompt: args.NLPrompt,
		})
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})

	// --- delete_workflow ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_workflow",
		Description: "Permanently delete a workflow.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args workflowIDArg) (*mcp.CallToolResult, any, error) {
		if err := client.DeleteWorkflow(args.WorkflowID); err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("Workflow %s deleted.", args.WorkflowID))
	})

	// --- activate_workflow ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "activate_workflow",
		Description: "Activate a draft or paused workflow so it responds to triggers.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args workflowIDArg) (*mcp.CallToolResult, any, error) {
		if err := client.ActivateWorkflow(args.WorkflowID); err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("Workflow %s activated.", args.WorkflowID))
	})

	// --- pause_workflow ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "pause_workflow",
		Description: "Pause an active workflow. It stops responding to triggers until re-activated.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args workflowIDArg) (*mcp.CallToolResult, any, error) {
		if err := client.PauseWorkflow(args.WorkflowID); err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("Workflow %s paused.", args.WorkflowID))
	})

	// --- duplicate_workflow ---
	type duplicateArgs struct {
		WorkflowID string `json:"workflow_id" jsonschema:"The workflow UUID to duplicate"`
		Name       string `json:"name" jsonschema:"Name for the new copy"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "duplicate_workflow",
		Description: "Create a copy of an existing workflow as a new draft.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args duplicateArgs) (*mcp.CallToolResult, any, error) {
		result, err := client.DuplicateWorkflow(args.WorkflowID, args.Name)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})

	// --- generate_workflow ---
	type promptArg struct {
		Prompt string `json:"prompt" jsonschema:"Natural language description of the desired workflow"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "generate_workflow",
		Description: "Generate a workflow from a natural language description using the Kestrel AI agent. Returns a complete workflow definition that can be saved with create_workflow.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args promptArg) (*mcp.CallToolResult, any, error) {
		result, err := client.GenerateWorkflow(args.Prompt)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})

	// --- request_workflow ---
	type requestArgs struct {
		Prompt        string `json:"prompt" jsonschema:"What you need, e.g. provision an MSK cluster with 3 brokers"`
		RequesterName string `json:"requester_name,omitempty" jsonschema:"Name of the requester"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "request_workflow",
		Description: "Submit a natural language request to trigger an existing workflow (like Slack /kestrel-workflow). If matched, the workflow executes immediately. If parameters are missing, returns what's needed. If no match, the request is logged for the infrastructure team.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args requestArgs) (*mcp.CallToolResult, any, error) {
		name := args.RequesterName
		if name == "" {
			name = cfg.Email
		}
		if name == "" {
			name = "mcp-agent"
		}
		result, err := client.TriggerWorkflowRequest(args.Prompt, name)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})

	// --- get_workflow_stats ---
	type emptyArgs struct{}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_workflow_stats",
		Description: "Get aggregate workflow statistics: total/active workflows, execution counts, status breakdown, and daily execution trend.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args emptyArgs) (*mcp.CallToolResult, any, error) {
		result, err := client.GetWorkflowStats()
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})

	// --- list_executions ---
	type listExecArgs struct {
		WorkflowID string `json:"workflow_id" jsonschema:"The workflow UUID"`
		Page       int    `json:"page,omitempty" jsonschema:"Page number (default 1)"`
		PageSize   int    `json:"page_size,omitempty" jsonschema:"Results per page (default 20)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_executions",
		Description: "List executions (runs) for a specific workflow with pagination.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args listExecArgs) (*mcp.CallToolResult, any, error) {
		page := args.Page
		if page < 1 {
			page = 1
		}
		size := args.PageSize
		if size < 1 {
			size = 20
		}
		result, err := client.ListExecutions(args.WorkflowID, page, size)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})

	// --- get_execution ---
	type execIDArg struct {
		ExecutionID string `json:"execution_id" jsonschema:"The execution UUID"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_execution",
		Description: "Get details of a specific workflow execution including step results.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args execIDArg) (*mcp.CallToolResult, any, error) {
		result, err := client.GetExecution(args.ExecutionID)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})

	// --- cancel_execution ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "cancel_execution",
		Description: "Cancel a running workflow execution.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args execIDArg) (*mcp.CallToolResult, any, error) {
		if err := client.CancelExecution(args.ExecutionID); err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("Execution %s cancelled.", args.ExecutionID))
	})

	// --- list_pending_approvals ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_pending_approvals",
		Description: "List all pending workflow approval gates across all executions. Includes manual approvals, PR approvals, and justification requests.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args emptyArgs) (*mcp.CallToolResult, any, error) {
		result, err := client.ListPendingApprovals()
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})

	// --- approve_step ---
	type approveArgs struct {
		ApprovalID    string `json:"approval_id" jsonschema:"The approval UUID"`
		Justification string `json:"justification,omitempty" jsonschema:"Required for justification-type approvals"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "approve_step",
		Description: "Approve a pending workflow approval gate. For justification-type approvals, include the justification text.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args approveArgs) (*mcp.CallToolResult, any, error) {
		var err error
		if args.Justification != "" {
			err = client.ApproveStepWithJustification(args.ApprovalID, args.Justification)
		} else {
			err = client.ApproveStep(args.ApprovalID)
		}
		if err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("Approval %s approved.", args.ApprovalID))
	})

	// --- reject_step ---
	type approvalIDArg struct {
		ApprovalID string `json:"approval_id" jsonschema:"The approval UUID"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "reject_step",
		Description: "Reject a pending workflow approval gate.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args approvalIDArg) (*mcp.CallToolResult, any, error) {
		if err := client.RejectStep(args.ApprovalID); err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("Approval %s rejected.", args.ApprovalID))
	})

	// --- list_unmatched_requests ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_unmatched_requests",
		Description: "List workflow requests that couldn't be matched to an existing workflow and need the infrastructure team to create one.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args emptyArgs) (*mcp.CallToolResult, any, error) {
		all, err := client.ListWorkflowRequests()
		if err != nil {
			return errResult(err)
		}
		var filtered []api.WorkflowRequest
		for _, r := range all {
			if r.Status == "no_workflow" || r.Status == "approved" || r.Status == "rejected" {
				filtered = append(filtered, r)
			}
		}
		return jsonResult(filtered)
	})

	// --- approve_workflow_request ---
	type requestIDArg struct {
		RequestID string `json:"request_id" jsonschema:"The workflow request UUID"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "approve_workflow_request",
		Description: "Approve an unmatched workflow request (infra team action).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args requestIDArg) (*mcp.CallToolResult, any, error) {
		if err := client.WorkflowRequestAction(args.RequestID, "approve"); err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("Request %s approved.", args.RequestID))
	})

	// --- reject_workflow_request ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "reject_workflow_request",
		Description: "Reject an unmatched workflow request.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args requestIDArg) (*mcp.CallToolResult, any, error) {
		if err := client.WorkflowRequestAction(args.RequestID, "reject"); err != nil {
			return errResult(err)
		}
		return textResult(fmt.Sprintf("Request %s rejected.", args.RequestID))
	})

	// --- get_catalog ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_catalog",
		Description: "Get the full catalog of available signal triggers and action blocks for building workflows.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args emptyArgs) (*mcp.CallToolResult, any, error) {
		result, err := client.GetCatalog()
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})

	// --- get_integrations_status ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_integrations_status",
		Description: "Check which integrations are connected (GitHub, GitLab, Slack, PagerDuty, Datadog, Jira, Confluence).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args emptyArgs) (*mcp.CallToolResult, any, error) {
		result, err := client.GetIntegrationsStatus()
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})

	// --- list_suggested_workflows ---
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_suggested_workflows",
		Description: "Get AI-suggested workflows based on connected integrations.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args emptyArgs) (*mcp.CallToolResult, any, error) {
		result, err := client.ListSuggestions()
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})
}
