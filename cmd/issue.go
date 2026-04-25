package cmd

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/lintingzhen/commitizen-go/branch"
	"github.com/lintingzhen/commitizen-go/git"
	"github.com/lintingzhen/commitizen-go/store"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func getIssueCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "issue",
		Short: "Start work on an issue (pick issue → create branch)",
		Long: `Enter issue details, then a properly named branch is created and
checked out from the default base branch. Branch state is saved to .git/git-cz.db.`,
		RunE: issueRunE,
	}
}

func issueRunE(_ *cobra.Command, _ []string) error {
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

	typeOpts := make([]huh.Option[string], 0)
	if len(msgCfg.Items) > 0 {
		for _, opt := range msgCfg.Items[0].Options {
			typeOpts = append(typeOpts, huh.NewOption(opt.Name, opt.Name))
		}
	}
	if len(typeOpts) == 0 {
		return errors.New("message config: no type options found in first item")
	}

	group1 := huh.NewGroup(
		huh.NewInput().
			Title("Issue ID:").
			Placeholder("ABC-42").
			Validate(func(s string) error {
				if s == "" {
					return errors.New("required")
				}

				return nil
			}).
			Value(&issueID),
		huh.NewInput().
			Title("Title:").
			Placeholder("Short description of the issue").
			Validate(func(s string) error {
				if s == "" {
					return errors.New("required")
				}

				return nil
			}).
			Value(&title),
		huh.NewSelect[string]().
			Title("Type:").
			Options(typeOpts...).
			Value(&branchType),
	)

	if err := huh.NewForm(group1).Run(); err != nil {
		return fmt.Errorf("issue form: %w", err)
	}

	branchName, err := branch.Name(issueID, branchType, title)
	if err != nil {
		return fmt.Errorf("assemble branch name: %w", err)
	}

	var confirmed bool
	confirmTitle := fmt.Sprintf("Create branch %q based on %q?", branchName, base)

	group2 := huh.NewGroup(
		huh.NewConfirm().
			Title(confirmTitle).
			Value(&confirmed),
	)

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

	root, err := client.WorkingTreeRoot()
	if err != nil {
		log.Printf("branch %q created; could not open store: %v", branchName, err)
	} else {
		s, err := store.Open(filepath.Join(root, ".git"))
		if err != nil {
			log.Printf("open store failed: %v", err)
		} else {
			defer func() { _ = s.Close() }()

			_, _, _, uuid, _ := branch.Parse(branchName)
			if err := s.InsertIssueWithBranch(
				store.Issue{IDSlug: issueID, Title: title, StatusID: 1},
				store.Branch{UUID: uuid, Name: branchName, Type: branchType, StatusID: 1},
			); err != nil {
				log.Printf("store insert failed: %v", err)
			}
		}
	}

	fmt.Printf("Switched to new branch %q (based on %q)\n", branchName, base)

	return nil
}
