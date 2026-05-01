package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"cli/pkg/render"

	"github.com/spf13/cobra"
)

var approvalsCmd = &cobra.Command{
	Use:   "approvals",
	Short: "Manage workflow approval gates",
}

func init() {
	approvalsCmd.AddCommand(approvalsListCmd)
	approvalsCmd.AddCommand(approvalsApproveCmd)
	approvalsCmd.AddCommand(approvalsRejectCmd)
}

var approvalsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pending workflow approvals",
	RunE: func(cmd *cobra.Command, _ []string) error {
		client := mustClient()
		approvals, err := client.ListPendingApprovals()
		if err != nil {
			return err
		}
		if len(approvals) == 0 {
			fmt.Println("No pending approvals.")
			return nil
		}

		headers := []string{"ID", "TYPE", "STEP", "EXECUTION", "REQUESTED", "EXPIRES"}
		rows := make([][]string, len(approvals))
		for i, a := range approvals {
			expires := "-"
			if a.ExpiresAt != nil {
				expires = render.TimeAgo(*a.ExpiresAt)
			}
			approvalType := a.ApprovalType
			if approvalType == "justification" {
				approvalType = render.Yellow("justification")
			}
			rows[i] = []string{
				a.ID,
				approvalType,
				a.StepID,
				a.ExecutionID,
				render.TimeAgo(a.RequestedAt),
				expires,
			}
		}
		fmt.Print(render.Table(headers, rows))
		fmt.Printf("\n  %s pending approvals\n", render.Bold(fmt.Sprintf("%d", len(approvals))))

		hasJustification := false
		for _, a := range approvals {
			if a.ApprovalType == "justification" {
				hasJustification = true
				break
			}
		}
		if hasJustification {
			fmt.Printf("\n  Justification approvals require a reason. Use:\n")
			fmt.Printf("    kestrel approvals approve <id> --justification \"your reason here\"\n")
		}
		return nil
	},
}

var approvalJustification string

var approvalsApproveCmd = &cobra.Command{
	Use:   "approve <approval-id>",
	Short: "Approve a workflow step",
	Long: `Approve a pending workflow approval gate. For justification-type approvals,
provide a reason with --justification or you will be prompted interactively.

Examples:
  kestrel approvals approve <id>
  kestrel approvals approve <id> --justification "Emergency fix for production outage"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()

		// Check if this is a justification approval
		approvals, _ := client.ListPendingApprovals()
		isJustification := false
		for _, a := range approvals {
			if a.ID == args[0] && a.ApprovalType == "justification" {
				isJustification = true
				break
			}
		}

		if isJustification && approvalJustification == "" {
			// Check context for prompt message
			for _, a := range approvals {
				if a.ID == args[0] {
					var ctx map[string]interface{}
					if err := json.Unmarshal(a.Context, &ctx); err == nil {
						if prompt, ok := ctx["prompt_message"].(string); ok && prompt != "" {
							fmt.Printf("  %s\n\n", prompt)
						}
					}
					break
				}
			}
			fmt.Printf("  %s This approval requires a justification.\n\n", render.Yellow("!"))
			fmt.Print("  Justification: ")
			reader := bufio.NewReader(os.Stdin)
			line, _ := reader.ReadString('\n')
			approvalJustification = strings.TrimSpace(line)
			if approvalJustification == "" {
				return fmt.Errorf("justification is required for this approval")
			}
		}

		if approvalJustification != "" {
			if err := client.ApproveStepWithJustification(args[0], approvalJustification); err != nil {
				return err
			}
		} else {
			if err := client.ApproveStep(args[0]); err != nil {
				return err
			}
		}
		fmt.Printf("%s Approved %s\n", render.Green("✓"), args[0])
		if approvalJustification != "" {
			fmt.Printf("  Justification: %s\n", approvalJustification)
		}
		return nil
	},
}

func init() {
	approvalsApproveCmd.Flags().StringVarP(&approvalJustification, "justification", "j", "", "Justification reason (required for justification-type approvals)")
}

var approvalsRejectCmd = &cobra.Command{
	Use:   "reject <approval-id>",
	Short: "Reject a workflow step",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()
		if err := client.RejectStep(args[0]); err != nil {
			return err
		}
		fmt.Printf("%s Rejected %s\n", render.Red("✗"), args[0])
		return nil
	},
}
