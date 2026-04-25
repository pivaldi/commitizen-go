package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// IssueCmd is the "git cz issue" subcommand stub.
// Full implementation is in Step 2 (tracker integration spec).
func getIssueCmd() *cobra.Command {
	var issueCmd = &cobra.Command{
		Use:   "issue",
		Short: "Start work on an issue (pick issue → create branch)",
		Long: `Browse issues from the configured tracker (Plane, Redmine, …),
select one, and create a properly named feature branch.

Not yet implemented — coming in Step 2.`,
		RunE: issueRunE,
	}

	return issueCmd
}

func issueRunE(_ *cobra.Command, _ []string) error {
	fmt.Println("not yet implemented")

	return nil
}
