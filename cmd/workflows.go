package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"cli/pkg/api"
	"cli/pkg/config"
	"cli/pkg/render"

	"github.com/spf13/cobra"
)

var workflowsCmd = &cobra.Command{
	Use:     "workflows",
	Aliases: []string{"wf"},
	Short:   "Manage Kestrel workflows",
}

func init() {
	workflowsCmd.AddCommand(wfListCmd)
	workflowsCmd.AddCommand(wfGetCmd)
	workflowsCmd.AddCommand(wfCreateCmd)
	workflowsCmd.AddCommand(wfDeleteCmd)
	workflowsCmd.AddCommand(wfActivateCmd)
	workflowsCmd.AddCommand(wfPauseCmd)
	workflowsCmd.AddCommand(wfDuplicateCmd)
	workflowsCmd.AddCommand(wfTestCmd)
	workflowsCmd.AddCommand(wfGenerateCmd)
	workflowsCmd.AddCommand(wfEditCmd)
	workflowsCmd.AddCommand(wfStatsCmd)
	workflowsCmd.AddCommand(wfExecutionsCmd)
	workflowsCmd.AddCommand(wfCatalogCmd)
	workflowsCmd.AddCommand(wfIntegrationsCmd)
	workflowsCmd.AddCommand(wfSuggestionsCmd)
	workflowsCmd.AddCommand(wfRequestCmd)
}

// --- list ---

var wfListStatus string

var wfListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all workflows",
	RunE: func(cmd *cobra.Command, _ []string) error {
		client := mustClient()
		workflows, err := client.ListWorkflows(wfListStatus)
		if err != nil {
			return err
		}
		if len(workflows) == 0 {
			fmt.Println("No workflows found.")
			return nil
		}

		headers := []string{"ID", "NAME", "STATUS", "TRIGGERS", "UPDATED"}
		rows := make([][]string, len(workflows))
		for i, wf := range workflows {
			rows[i] = []string{
				wf.ID,
				wf.Name,
				render.StatusColor(wf.Status),
				fmt.Sprintf("%d", wf.TriggerCount),
				render.TimeAgo(wf.UpdatedAt),
			}
		}
		fmt.Print(render.Table(headers, rows))
		fmt.Printf("\n  %s workflows total\n", render.Bold(fmt.Sprintf("%d", len(workflows))))
		return nil
	},
}

func init() {
	wfListCmd.Flags().StringVarP(&wfListStatus, "status", "s", "", "Filter by status (draft, active, paused, archived)")
}

// --- get ---

var wfGetCmd = &cobra.Command{
	Use:   "get <workflow-id>",
	Short: "Show detailed info and diagram for a workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()
		wf, err := client.GetWorkflow(args[0])
		if err != nil {
			return err
		}
		printWorkflowDetail(wf)
		return nil
	},
}

func printWorkflowDetail(wf *api.Workflow) {
	fmt.Printf("\n  %s %s\n", render.Bold(wf.Name), render.StatusColor(wf.Status))
	if wf.Description != "" {
		fmt.Printf("  %s\n", render.Gray(wf.Description))
	}
	fmt.Printf("\n  %-14s %s\n", render.Gray("ID:"), wf.ID)
	fmt.Printf("  %-14s %d\n", render.Gray("Triggers:"), wf.TriggerCount)
	fmt.Printf("  %-14s %s\n", render.Gray("Created:"), render.TimeAgo(wf.CreatedAt))
	fmt.Printf("  %-14s %s\n", render.Gray("Updated:"), render.TimeAgo(wf.UpdatedAt))
	if wf.LastTriggeredAt != nil {
		fmt.Printf("  %-14s %s\n", render.Gray("Last trigger:"), render.TimeAgo(*wf.LastTriggeredAt))
	}
	if wf.NLPrompt != "" {
		fmt.Printf("  %-14s %s\n", render.Gray("Description:"), wf.NLPrompt)
	}

	// Trigger config summary
	var tc map[string]interface{}
	if err := json.Unmarshal(wf.TriggerConfig, &tc); err == nil {
		if src, ok := tc["source"].(string); ok {
			fmt.Printf("  %-14s %s\n", render.Gray("Source:"), src)
		}
	}

	fmt.Printf("\n  %s\n\n", render.Bold("Workflow Diagram"))
	fmt.Println(render.WorkflowDiagram(wf.Definition))
}

// --- create (from file) ---

var wfCreateFile string

var wfCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a workflow from a JSON file",
	Long:  "Create a workflow from a JSON file. Use `kestrel workflows generate` to create workflows from natural language.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if wfCreateFile == "" {
			return fmt.Errorf("--file is required")
		}
		data, err := os.ReadFile(wfCreateFile)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
		var req api.CreateWorkflowRequest
		if err := json.Unmarshal(data, &req); err != nil {
			return fmt.Errorf("parse JSON: %w", err)
		}
		if req.Name == "" {
			return fmt.Errorf("workflow JSON must include a 'name' field")
		}

		client := mustClient()
		wf, err := client.CreateWorkflow(req)
		if err != nil {
			return err
		}
		fmt.Printf("%s Created workflow %s (%s)\n", render.Green("✓"), render.Bold(wf.Name), wf.ID)
		return nil
	},
}

func init() {
	wfCreateCmd.Flags().StringVarP(&wfCreateFile, "file", "f", "", "Path to workflow JSON file")
}

// --- delete ---

var wfDeleteCmd = &cobra.Command{
	Use:   "delete <workflow-id>",
	Short: "Delete a workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()
		if err := client.DeleteWorkflow(args[0]); err != nil {
			return err
		}
		fmt.Printf("%s Deleted workflow %s\n", render.Green("✓"), args[0])
		return nil
	},
}

// --- activate ---

var wfActivateCmd = &cobra.Command{
	Use:   "activate <workflow-id>",
	Short: "Activate a workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()
		if err := client.ActivateWorkflow(args[0]); err != nil {
			return err
		}
		fmt.Printf("%s Workflow %s activated\n", render.Green("✓"), args[0])
		return nil
	},
}

// --- pause ---

var wfPauseCmd = &cobra.Command{
	Use:   "pause <workflow-id>",
	Short: "Pause an active workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()
		if err := client.PauseWorkflow(args[0]); err != nil {
			return err
		}
		fmt.Printf("%s Workflow %s paused\n", render.Green("✓"), args[0])
		return nil
	},
}

// --- duplicate ---

var wfDuplicateName string

var wfDuplicateCmd = &cobra.Command{
	Use:   "duplicate <workflow-id>",
	Short: "Duplicate a workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()
		dup, err := client.DuplicateWorkflow(args[0], wfDuplicateName)
		if err != nil {
			return err
		}
		fmt.Printf("%s Duplicated as %s (%s)\n", render.Green("✓"), render.Bold(dup.Name), dup.ID)
		return nil
	},
}

func init() {
	wfDuplicateCmd.Flags().StringVarP(&wfDuplicateName, "name", "n", "", "Name for the duplicate workflow")
}

// --- edit ---

var (
	wfEditName        string
	wfEditDescription string
)

var wfEditCmd = &cobra.Command{
	Use:   "edit <workflow-id>",
	Short: "Update a workflow's name or description",
	Long: `Update properties of an existing workflow. Only the provided flags are changed.

Example:
  kestrel workflows edit <id> --description "When a pod crashloops, run RCA and notify Slack"
  kestrel workflows edit <id> --name "New Name" --description "New description"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if wfEditName == "" && wfEditDescription == "" {
			return fmt.Errorf("at least one of --name or --description is required")
		}

		client := mustClient()
		wf, err := client.GetWorkflow(args[0])
		if err != nil {
			return fmt.Errorf("fetch workflow: %w", err)
		}

		req := api.UpdateWorkflowRequest{
			Name:          wf.Name,
			Description:   wf.Description,
			Definition:    wf.Definition,
			TriggerConfig: wf.TriggerConfig,
			NLPrompt:      wf.NLPrompt,
		}
		if wfEditName != "" {
			req.Name = wfEditName
		}
		if wfEditDescription != "" {
			req.Description = wfEditDescription
		}

		updated, err := client.UpdateWorkflow(args[0], req)
		if err != nil {
			return err
		}
		fmt.Printf("%s Updated workflow %s (%s)\n", render.Green("✓"), render.Bold(updated.Name), updated.ID)
		return nil
	},
}

func init() {
	wfEditCmd.Flags().StringVar(&wfEditName, "name", "", "New workflow name")
	wfEditCmd.Flags().StringVar(&wfEditDescription, "description", "", "New workflow description (NL prompt)")
}

// --- test ---

var wfTestCmd = &cobra.Command{
	Use:   "test <workflow-id>",
	Short: "Trigger a dry-run test execution of a workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()
		exec, err := client.TestWorkflow(args[0])
		if err != nil {
			return err
		}
		fmt.Printf("%s Test execution started: %s\n", render.Green("✓"), exec.ID)
		return nil
	},
}

// --- generate (NL) ---

var wfGenerateCmd = &cobra.Command{
	Use:   "generate <prompt>",
	Short: "Generate a workflow from natural language description",
	Long: `Use the Kestrel AI agent to generate a workflow from a plain English description.
The generated workflow is displayed as an ASCII diagram and can be saved.

Example:
  kestrel workflows generate "When a pod crashloops, run RCA, create a Jira ticket, and notify #incidents on Slack"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prompt := strings.Join(args, " ")
		fmt.Printf("  Generating workflow from: %s\n\n", render.Cyan(prompt))

		client := mustClient()
		resp, err := client.GenerateWorkflow(prompt)
		if err != nil {
			return err
		}

		if resp.Error != nil && *resp.Error != "" {
			return fmt.Errorf("agent error: %s", *resp.Error)
		}

		fmt.Printf("  %s\n\n", render.Bold("AI Explanation"))
		fmt.Printf("  %s\n\n", resp.Explanation)

		fmt.Printf("  %s %s\n", render.Bold("Name:"), resp.Name)
		if resp.Description != "" {
			fmt.Printf("  %s %s\n", render.Bold("Description:"), resp.Description)
		}
		fmt.Printf("\n  %s\n\n", render.Bold("Workflow Diagram"))
		fmt.Println(render.WorkflowDiagram(resp.Definition))

		if wfGenerateSave {
			req := api.CreateWorkflowRequest{
				Name:          resp.Name,
				Description:   resp.Description,
				Definition:    resp.Definition,
				TriggerConfig: resp.TriggerConfig,
				NLPrompt:      prompt,
			}
			wf, err := client.CreateWorkflow(req)
			if err != nil {
				return fmt.Errorf("save workflow: %w", err)
			}
			fmt.Printf("\n%s Saved as workflow %s (%s)\n", render.Green("✓"), render.Bold(wf.Name), wf.ID)
		} else {
			fmt.Printf("\n  Add %s to save this workflow as a draft.\n", render.Bold("--save"))
		}
		return nil
	},
}

var wfGenerateSave bool

func init() {
	wfGenerateCmd.Flags().BoolVar(&wfGenerateSave, "save", false, "Immediately save the generated workflow as a draft")
}

// --- stats ---

var wfStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show workflow execution statistics",
	RunE: func(cmd *cobra.Command, _ []string) error {
		client := mustClient()
		stats, err := client.GetWorkflowStats()
		if err != nil {
			return err
		}

		fmt.Printf("\n  %s\n\n", render.Bold("Workflow Statistics"))
		fmt.Printf("  %-24s %s\n", "Total workflows:", render.Bold(fmt.Sprintf("%d", stats.TotalWorkflows)))
		fmt.Printf("  %-24s %s\n", "Active workflows:", render.Green(fmt.Sprintf("%d", stats.ActiveWorkflows)))
		fmt.Printf("  %-24s %s\n", "Total executions:", render.Bold(fmt.Sprintf("%d", stats.TotalExecutions)))

		if len(stats.StatusBreakdown) > 0 {
			fmt.Printf("\n  %s\n", render.Bold("Execution Breakdown"))
			for status, count := range stats.StatusBreakdown {
				fmt.Printf("    %-22s %d\n", render.StatusColor(status)+":", count)
			}
		}

		if len(stats.DailyExecutions) > 0 {
			fmt.Printf("\n  %s\n", render.Bold("Daily Executions (last 7 days)"))
			headers := []string{"DATE", "COMPLETED", "FAILED", "RUNNING", "WAITING"}
			rows := make([][]string, len(stats.DailyExecutions))
			for i, d := range stats.DailyExecutions {
				rows[i] = []string{
					d.Date,
					render.Green(fmt.Sprintf("%d", d.Completed)),
					render.Red(fmt.Sprintf("%d", d.Failed)),
					render.Yellow(fmt.Sprintf("%d", d.Running)),
					render.Yellow(fmt.Sprintf("%d", d.Waiting)),
				}
			}
			fmt.Println()
			fmt.Print(render.Table(headers, rows))
		}
		fmt.Println()
		return nil
	},
}

// --- executions ---

var (
	execPage     int
	execPageSize int
)

var wfExecutionsCmd = &cobra.Command{
	Use:   "executions <workflow-id>",
	Short: "List executions for a workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()
		resp, err := client.ListExecutions(args[0], execPage, execPageSize)
		if err != nil {
			return err
		}
		if len(resp.Executions) == 0 {
			fmt.Println("No executions found.")
			return nil
		}

		headers := []string{"ID", "STATUS", "STARTED", "COMPLETED", "ERROR"}
		rows := make([][]string, len(resp.Executions))
		for i, e := range resp.Executions {
			completed := "-"
			if e.CompletedAt != nil {
				completed = render.TimeAgo(*e.CompletedAt)
			}
			rows[i] = []string{
				e.ID,
				render.StatusColor(e.Status),
				render.TimeAgo(e.StartedAt),
				completed,
				render.Truncate(e.ErrorMessage, 120),
			}
		}
		fmt.Print(render.Table(headers, rows))
		totalPages := (resp.Total + execPageSize - 1) / execPageSize
		fmt.Printf("\n  Page %d of %d (%d executions total)\n", resp.Page, totalPages, resp.Total)
		if resp.Page*execPageSize < resp.Total {
			fmt.Printf("  Next page: kestrel workflows executions %s --page %d\n", args[0], resp.Page+1)
		}
		return nil
	},
}

func init() {
	wfExecutionsCmd.Flags().IntVar(&execPage, "page", 1, "Page number")
	wfExecutionsCmd.Flags().IntVar(&execPageSize, "page-size", 20, "Results per page")
}

// --- catalog ---

var wfCatalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Show available signal triggers and actions",
	RunE: func(cmd *cobra.Command, _ []string) error {
		client := mustClient()
		cat, err := client.GetCatalog()
		if err != nil {
			return err
		}

		fmt.Printf("\n  %s\n\n", render.Bold("Signal Triggers"))
		headers := []string{"ID", "SOURCE", "NAME", "DESCRIPTION"}
		rows := make([][]string, len(cat.Signals))
		for i, s := range cat.Signals {
			rows[i] = []string{s.ID, render.Cyan(s.Source), s.Name, render.Truncate(s.Description, 120)}
		}
		fmt.Print(render.Table(headers, rows))

		fmt.Printf("\n  %s\n\n", render.Bold("Actions"))
		headers = []string{"ID", "INTEGRATION", "NAME", "DESCRIPTION"}
		rows = make([][]string, len(cat.Actions))
		for i, a := range cat.Actions {
			rows[i] = []string{a.ID, render.Blue(a.Integration), a.Name, render.Truncate(a.Description, 120)}
		}
		fmt.Print(render.Table(headers, rows))
		fmt.Println()
		return nil
	},
}

// --- integrations ---

var wfIntegrationsCmd = &cobra.Command{
	Use: "integrations",

	Short: "Show integration connection status",
	RunE: func(cmd *cobra.Command, _ []string) error {
		client := mustClient()
		integrations, err := client.GetIntegrationsStatus()
		if err != nil {
			return err
		}

		fmt.Printf("\n  %s\n\n", render.Bold("Integration Status"))
		for _, ig := range integrations {
			status := render.Red("disconnected")
			icon := "✗"
			if ig.Connected {
				status = render.Green("connected")
				icon = "✓"
			}
			fmt.Printf("  %s %-20s %s\n", icon, ig.Name, status)
		}
		fmt.Println()
		return nil
	},
}

// --- suggestions ---

var wfSuggestionsCmd = &cobra.Command{
	Use:   "suggestions",
	Short: "Show AI-suggested workflows",
	RunE: func(cmd *cobra.Command, _ []string) error {
		client := mustClient()
		suggestions, err := client.ListSuggestions()
		if err != nil {
			return err
		}
		if len(suggestions) == 0 {
			fmt.Println("No workflow suggestions available.")
			return nil
		}

		for i, s := range suggestions {
			fmt.Printf("\n  %s %s\n", render.Bold(fmt.Sprintf("[%d]", i+1)), render.Bold(s.Name))
			fmt.Printf("  %s\n", render.Gray(s.Description))
			if s.NLPrompt != "" {
				fmt.Printf("  %s %s\n", render.Gray("Description:"), s.NLPrompt)
			}
			fmt.Println()
			fmt.Println(render.WorkflowDiagram(s.Definition))
		}
		return nil
	},
}

// --- request ---

var wfRequestCmd = &cobra.Command{
	Use:   "request <description>",
	Short: "Request a workflow by describing what you need",
	Long: `Submit a natural language request to trigger an existing workflow.
If a matching workflow is found, it executes immediately. If parameters are
missing, you'll be prompted for each one interactively. If no workflow matches,
the request is logged for your infrastructure team to review.

This is the CLI equivalent of the Slack /kestrel-workflow command.

Examples:
  kestrel workflows request "provision an MSK cluster with 3 brokers in us-east-1"
  kestrel workflows request "scale the payments deployment to 5 replicas"
  kestrel workflows request "create a config map"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prompt := strings.Join(args, " ")
		fmt.Printf("  Routing request: %s\n\n", render.Cyan(prompt))

		client := mustClient()
		cfg, _ := config.Load()
		requesterName := cfg.Email
		if requesterName == "" {
			requesterName = "cli-user"
		}

		reader := bufio.NewReader(os.Stdin)
		currentPrompt := prompt
		const maxRounds = 5

		for round := 0; round < maxRounds; round++ {
			result, err := client.TriggerWorkflowRequest(currentPrompt, requesterName)
			if err != nil {
				return err
			}

			status, _ := result["status"].(string)

			switch status {
			case "executing":
				wfName, _ := result["workflow_name"].(string)
				requestID, _ := result["request_id"].(string)
				explanation, _ := result["explanation"].(string)

				fmt.Printf("  %s Matched workflow: %s\n", render.Green("✓"), render.Bold(wfName))
				if explanation != "" {
					fmt.Printf("  %s\n", render.Gray(explanation))
				}
				if params, ok := result["parameters"].(map[string]interface{}); ok && len(params) > 0 {
					fmt.Printf("\n  %s\n", render.Bold("Parameters"))
					for k, v := range params {
						fmt.Printf("    %-20s %v\n", k+":", v)
					}
				}
				fmt.Printf("\n  %s Workflow is now executing.\n", render.Green("✓"))
				if requestID != "" {
					fmt.Printf("  Track it with: kestrel requests list\n")
				}
				return nil

			case "missing_parameters":
				wfName, _ := result["workflow_name"].(string)

				fmt.Printf("  %s Matched workflow: %s\n", render.Yellow("!"), render.Bold(wfName))

				if params, ok := result["extracted_parameters"].(map[string]interface{}); ok && len(params) > 0 {
					fmt.Printf("  %s", render.Gray("Extracted: "))
					parts := make([]string, 0, len(params))
					for k, v := range params {
						parts = append(parts, fmt.Sprintf("%s=%v", k, v))
					}
					fmt.Printf("%s\n", render.Gray(strings.Join(parts, ", ")))
				}

				missing, _ := result["missing_parameters"].([]interface{})
				if len(missing) == 0 {
					return fmt.Errorf("server reported missing parameters but did not specify which ones")
				}

				fmt.Printf("\n  Please provide the following details:\n\n")
				collected := make(map[string]string)
				for _, p := range missing {
					param := fmt.Sprintf("%v", p)
					fmt.Printf("  %s: ", render.Bold(param))
					line, _ := reader.ReadString('\n')
					line = strings.TrimSpace(line)
					if line == "" {
						return fmt.Errorf("parameter %q is required", param)
					}
					collected[param] = line
				}

				var paramParts []string
				for k, v := range collected {
					paramParts = append(paramParts, fmt.Sprintf("%s: %s", k, v))
				}
				currentPrompt = prompt + ". " + strings.Join(paramParts, ", ")

				fmt.Printf("\n  Submitting with parameters...\n\n")
				continue

			case "no_workflow":
				category, _ := result["category"].(string)
				explanation, _ := result["explanation"].(string)
				requestURL, _ := result["request_url"].(string)
				requestID, _ := result["request_id"].(string)

				fmt.Printf("  %s No matching workflow found\n", render.Red("✗"))
				if category != "" {
					fmt.Printf("  %s %s\n", render.Gray("Category:"), category)
				}
				if explanation != "" {
					fmt.Printf("  %s\n", render.Gray(explanation))
				}
				fmt.Printf("\n  Your request has been logged for your infrastructure team to review.\n")
				if requestURL != "" {
					fmt.Printf("  View requests: %s\n", requestURL)
				}
				if requestID != "" {
					fmt.Printf("  Request ID: %s\n", requestID)
				}
				fmt.Printf("\n  Once a workflow is created for this type of request, it will be\n")
				fmt.Printf("  available for future requests via this command.\n")
				return nil

			default:
				if errMsg, ok := result["error"].(string); ok {
					return fmt.Errorf("server error: %s", errMsg)
				}
				fmt.Printf("  Response: %v\n", result)
				return nil
			}
		}
		return fmt.Errorf("could not collect all required parameters after %d rounds", maxRounds)
	},
}
