package cmd

import (
	"fmt"
	"log"

	"github.com/lintingzhen/commitizen-go/commit"
	"github.com/lintingzhen/commitizen-go/git"
	"github.com/lintingzhen/commitizen-go/tui"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	commitAll        bool
	commitAmend      bool
	commitNoVerify   bool
	commitSignoff    bool
	commitAllowEmpty bool
	commitAuthor     string
)

// CommitCmd is the "git cz commit" subcommand.
func getCommitCmd() *cobra.Command {
	var commitCmd = &cobra.Command{
		Use:   "commit",
		Short: "Record changes to the repository",
		Long:  "Open the commitizen TUI to compose a standardised commit message, then commit using go-git.",
		RunE:  commitRunE,
	}

	f := commitCmd.Flags()
	f.BoolVarP(&commitAll, "all", "a", false,
		"stage all tracked modified/deleted files before committing")
	f.BoolVar(&commitAmend, "amend", false,
		"replace the tip of the current branch")
	f.BoolVarP(&commitNoVerify, "no-verify", "n", false,
		"bypass pre-commit and commit-msg hooks")
	f.BoolVarP(&commitSignoff, "signoff", "s", false,
		"add Signed-off-by trailer to the commit message")
	f.BoolVar(&commitAllowEmpty, "allow-empty", false,
		"allow a commit with no changes")
	f.StringVar(&commitAuthor, "author", "",
		`override commit author as "Name <email>"`)

	return commitCmd
}

// loadMessageConfig merges the built-in default with any user override from .git-czrc.
func loadMessageConfig() (tui.CommitMessageConfig, error) {
	cfg, err := commit.DefaultMessageConfig()
	if err != nil {
		return tui.CommitMessageConfig{}, fmt.Errorf("failed to load config: %w", err)
	}

	sub := viper.Sub("message")
	if sub == nil {
		log.Print("no message config in config file, using defaults")
	} else {
		if err := sub.Unmarshal(&cfg, func(dc *mapstructure.DecoderConfig) { dc.ZeroFields = true }); err != nil {
			log.Printf("ill message in config file, err=%v", err)
		}
	}

	return cfg, nil
}

func commitRunE(_ *cobra.Command, _ []string) error {
	client, err := git.NewClient()
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}

	authors, err := client.Authors()
	if err != nil {
		log.Printf("could not load author list: %v", err)
		authors = []string{}
	}

	defaults := tui.CommitOption{
		Authors:    authors,
		All:        commitAll,
		Amend:      commitAmend,
		NoVerify:   commitNoVerify,
		Signoff:    commitSignoff,
		AllowEmpty: commitAllowEmpty,
		Author:     commitAuthor,
	}

	msgCfg, err := loadMessageConfig()
	if err != nil {
		return fmt.Errorf("load message config: %w", err)
	}

	msg, opts, err := commit.FillOutForm(msgCfg, defaults)
	if err != nil {
		return fmt.Errorf("failed to fill form: %w", err)
	}

	summary, err := client.Commit(msg, git.CommitOptions{
		All:        opts.All,
		Amend:      opts.Amend,
		NoVerify:   opts.NoVerify,
		Signoff:    opts.Signoff,
		AllowEmpty: opts.AllowEmpty,
		Author:     opts.Author,
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	printCommitSummary(summary)

	return nil
}

func printCommitSummary(s git.CommitSummary) {
	ref := s.Branch
	if s.IsRoot {
		ref += " (root-commit)"
	}
	fmt.Printf("[%s %s] %s\n", ref, s.ShortHash, s.Subject)

	if s.Files == 0 {
		return
	}

	fileWord := "files"
	if s.Files == 1 {
		fileWord = "file"
	}
	line := fmt.Sprintf(" %d %s changed", s.Files, fileWord)

	if s.Additions > 0 {
		word := "insertions"
		if s.Additions == 1 {
			word = "insertion"
		}
		line += fmt.Sprintf(", %d %s(+)", s.Additions, word)
	}
	if s.Deletions > 0 {
		word := "deletions"
		if s.Deletions == 1 {
			word = "deletion"
		}
		line += fmt.Sprintf(", %d %s(-)", s.Deletions, word)
	}
	fmt.Println(line)
}
