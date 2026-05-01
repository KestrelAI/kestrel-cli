package cmd

import (
	"fmt"

	"cli/pkg/api"
	"cli/pkg/render"

	"github.com/spf13/cobra"
)

var requestsCmd = &cobra.Command{
	Use:   "requests",
	Short: "Manage workflow requests",
}

func init() {
	requestsCmd.AddCommand(requestsListCmd)
	requestsCmd.AddCommand(requestsApproveCmd)
	requestsCmd.AddCommand(requestsRejectCmd)
}

var requestsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List unmatched workflow requests needing review",
	RunE: func(cmd *cobra.Command, _ []string) error {
		client := mustClient()
		reqs, err := client.ListWorkflowRequests()
		if err != nil {
			return err
		}

		// Only show requests that weren't matched to an existing workflow.
		var filtered []api.WorkflowRequest
		for _, r := range reqs {
			switch r.Status {
			case "no_workflow", "approved", "rejected":
				filtered = append(filtered, r)
			}
		}

		if len(filtered) == 0 {
			fmt.Println("No unmatched workflow requests.")
			return nil
		}

		headers := []string{"ID", "STATUS", "SOURCE", "REQUESTER", "REQUEST", "CREATED"}
		rows := make([][]string, len(filtered))
		for i, r := range filtered {
			rows[i] = []string{
				r.ID,
				render.StatusColor(r.Status),
				r.Source,
				render.Truncate(r.RequesterName, 20),
				r.Prompt,
				render.TimeAgo(r.CreatedAt),
			}
		}
		fmt.Print(render.Table(headers, rows))
		fmt.Printf("\n  %s unmatched requests\n", render.Bold(fmt.Sprintf("%d", len(filtered))))
		return nil
	},
}

var requestsApproveCmd = &cobra.Command{
	Use:   "approve <request-id>",
	Short: "Approve a workflow request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()
		if err := client.WorkflowRequestAction(args[0], "approve"); err != nil {
			return err
		}
		fmt.Printf("%s Approved request %s\n", render.Green("✓"), args[0])
		return nil
	},
}

var requestsRejectCmd = &cobra.Command{
	Use:   "reject <request-id>",
	Short: "Reject a workflow request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()
		if err := client.WorkflowRequestAction(args[0], "reject"); err != nil {
			return err
		}
		fmt.Printf("%s Rejected request %s\n", render.Red("✗"), args[0])
		return nil
	},
}
