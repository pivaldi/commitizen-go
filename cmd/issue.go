package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/lintingzhen/commitizen-go/branch"
	"github.com/lintingzhen/commitizen-go/git"
	"github.com/lintingzhen/commitizen-go/store"
	"github.com/lintingzhen/commitizen-go/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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

	var issueID, title, branchType string

	allowedBranchTypes := getAllowedBranchType(msgCfg.Items)
	if len(allowedBranchTypes) == 0 {
		return errors.New("message config: no type options found in first item")
	}

	group1 := tui.IssueInput(&issueID, &title, &branchType, allowedBranchTypes)

	if err := huh.NewForm(group1).Run(); err != nil {
		return fmt.Errorf("issue form: %w", err)
	}

	b, err := branch.New(issueID, branchType, title)
	if err != nil {
		return fmt.Errorf("assemble branch name: %w", err)
	}

	branchName := b.Name()

	var confirmed bool
	confirmTitle := fmt.Sprintf("Create branch %q based on %q?", branchName, base)

	group2 := tui.IssueConfirm(confirmTitle, &confirmed)

	if err := huh.NewForm(group2).Run(); err != nil {
		return fmt.Errorf("confirm form: %w", err)
	}

	if !confirmed {
		fmt.Println("Aborted.")

		return nil
	}

	if err := client.CreateBranch(branchName, base); err != nil {
		return fmt.Errorf("create branch: %w", err)
	}

	persist(cmd.Context(), client, b, title)

	fmt.Printf("Switched to new branch %q (based on %q)\n", branchName, base)

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

func persist(ctx context.Context, client *git.Client, b *branch.Branch, rawTitle string) {
	root, err := client.WorkingTreeRoot()
	if err != nil {
		log.Printf("branch %q created; could not open store: %v", b.Name(), err)

		return
	}

	s, err := store.Open(ctx, filepath.Join(root, ".git"))
	if err != nil {
		log.Printf("open store failed: %v", err)
	} else {
		defer func() { _ = s.Close() }()

		if err := s.InsertIssueWithBranch(ctx,
			&store.Issue{IDSlug: b.IssueID(), Title: rawTitle, StatusID: 1},
			&store.Branch{UUID: b.ID(), Name: b.Name(), Type: b.Type(), StatusID: 1},
		); err != nil {
			log.Printf("store insert failed: %v", err)
		}
	}
}
