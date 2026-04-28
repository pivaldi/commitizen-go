package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/lintingzhen/commitizen-go/branch"
	"github.com/lintingzhen/commitizen-go/git"
	"github.com/lintingzhen/commitizen-go/store"
	"github.com/lintingzhen/commitizen-go/tracker"
	_ "github.com/lintingzhen/commitizen-go/tracker/redmine" // registers redmine adapter
	"github.com/lintingzhen/commitizen-go/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type issueStartFlags struct {
	trackerFirst bool
}

func getIssueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Manage issues",
		RunE:  issueRunE,
	}
	cmd.AddCommand(getIssueStartCmd())

	return cmd
}

func issueRunE(cmd *cobra.Command, args []string) error {
	var action string
	if err := huh.NewForm(tui.IssueActionSelect(&action)).Run(); err != nil {
		return fmt.Errorf("action select: %w", err)
	}

	switch action {
	case tui.IssueActionNameStart:
		return issueStartRunE(cmd, args)
	default:
		fmt.Println("Not yet implemented.")

		return nil
	}
}

func getIssueStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start work on an issue (create branch)",
		Long: `Enter issue details, then a properly named branch is created and
checked out from the default base branch. Branch state is saved to .git/git-cz.db.`,
		RunE: issueStartRunE,
	}
}

func issueStartRunE(cmd *cobra.Command, _ []string) error {
	return runIssueStart(cmd.Context(), issueStartFlags{trackerFirst: true})
}

// runIssueStart contains the full issue-start flow. trackerFirst=true for
// `issue start` (tracker pre-selected); false for `branch new` (manual pre-selected).
func runIssueStart(ctx context.Context, flags issueStartFlags) error {
	client, err := git.NewClient()
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}

	msgCfg, err := loadMessageConfig()
	if err != nil {
		return fmt.Errorf("load message config: %w", err)
	}

	base := viper.GetString("branch.base")
	if base == "" {
		base, err = client.DefaultBaseBranch()
		if err != nil {
			return fmt.Errorf("detect base branch: %w", err)
		}
	}

	allowedBranchTypes := getAllowedBranchType(msgCfg.Items)
	if len(allowedBranchTypes) == 0 {
		return errors.New("message config: no type options found in first item")
	}

	trackerCfg := tracker.Config{
		Type:             viper.GetString("tracker.type"),
		URL:              viper.GetString("tracker.url"),
		Token:            viper.GetString("tracker.token"),
		InProgressStatus: viper.GetString("tracker.in_progress_status"),
	}
	if trackerCfg.InProgressStatus == "" {
		trackerCfg.InProgressStatus = "In Progress"
	}

	var issueID, title, branchType string
	var fromTracker bool
	var pickedIssue tracker.Issue
	var t tracker.Tracker

	if trackerCfg.Type != "" {
		t, err = tracker.New(trackerCfg)
		if err != nil {
			return fmt.Errorf("create tracker: %w", err)
		}

		var useTracker bool
		if err := huh.NewForm(tui.IssueTrackerToggle(&useTracker, flags.trackerFirst, trackerCfg.Type)).Run(); err != nil {
			return fmt.Errorf("tracker toggle: %w", err)
		}

		if useTracker {
			issues, listErr := t.ListIssues(ctx)
			if listErr != nil {
				if noteErr := huh.NewForm(tui.IssueTrackerError(listErr.Error())).Run(); noteErr != nil {
					return fmt.Errorf("error note: %w", noteErr)
				}
				// fallthrough to manual input
			} else if len(issues) == 0 {
				if noteErr := huh.NewForm(tui.IssueTrackerError("no open issues assigned to you")).Run(); noteErr != nil {
					return fmt.Errorf("error note: %w", noteErr)
				}
				// fallthrough to manual input
			} else {
				if err := huh.NewForm(tui.IssueTrackerPicker(issues, &pickedIssue, allowedBranchTypes, &branchType)).Run(); err != nil {
					return fmt.Errorf("tracker picker: %w", err)
				}

				issueID = pickedIssue.ID
				title = pickedIssue.Subject
				fromTracker = true
			}
		}
	}

	if !fromTracker {
		if err := huh.NewForm(tui.IssueInput(&issueID, &title, &branchType, allowedBranchTypes)).Run(); err != nil {
			return fmt.Errorf("issue form: %w", err)
		}
	}

	b, err := branch.New(issueID, branchType, title)
	if err != nil {
		return fmt.Errorf("assemble branch name: %w", err)
	}

	branchName := b.Name()

	var confirmed bool
	if err := huh.NewForm(tui.IssueConfirm(
		fmt.Sprintf("Create branch %q based on %q?", branchName, base), &confirmed,
	)).Run(); err != nil {
		return fmt.Errorf("confirm form: %w", err)
	}

	if !confirmed {
		fmt.Println("Aborted.")

		return nil
	}

	if err := client.CreateBranch(branchName, base); err != nil {
		return fmt.Errorf("create branch: %w", err)
	}

	var tt *string
	if fromTracker {
		tt = &trackerCfg.Type
	}

	if err := persist(ctx, client, b, title, tt); err != nil {
		fmt.Fprintf(os.Stderr, "warning: branch created but store record failed: %v\n", err)
	}

	fmt.Printf("Switched to new branch %q (based on %q)\n", branchName, base)

	if fromTracker {
		var updateStatus bool
		if err := huh.NewForm(tui.IssueUpdateStatusConfirm(
			pickedIssue.ID, trackerCfg.InProgressStatus, trackerCfg.Type, &updateStatus,
		)).Run(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: status confirm form: %v\n", err)
		} else if updateStatus {
			if err := t.UpdateIssueStatus(ctx, pickedIssue.ID, trackerCfg.InProgressStatus); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not update tracker status: %v\n", err)
			}
		}
	}

	return nil
}

func getAllowedBranchType(items []tui.CommitItem) []string {
	if len(items) == 0 {
		return nil
	}

	allowedBranchTypes := make([]string, 0, len(items[0].Options))
	for _, opt := range items[0].Options {
		allowedBranchTypes = append(allowedBranchTypes, opt.Name)
	}

	return allowedBranchTypes
}

func persist(ctx context.Context, client *git.Client, b *branch.Branch, rawTitle string, trackerType *string) error {
	root, err := client.WorkingTreeRoot()
	if err != nil {
		return fmt.Errorf("working tree root: %w", err)
	}

	s, err := store.Open(ctx, filepath.Join(root, ".git"))
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer func() { _ = s.Close() }()

	if err := s.InsertIssueWithBranch(ctx,
		&store.Issue{IDSlug: b.IssueID(), Title: rawTitle, StatusID: 1, TrackerType: trackerType},
		&store.Branch{UUID: b.ID(), Name: b.Name(), Type: b.Type(), StatusID: 1},
	); err != nil {
		return fmt.Errorf("insert issue with branch: %w", err)
	}

	return nil
}
