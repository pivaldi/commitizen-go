package cmd

import (
	"fmt"
	"log"

	"github.com/lintingzhen/commitizen-go/commit"
	"github.com/lintingzhen/commitizen-go/git"
	"github.com/spf13/cobra"
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
var CommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Record changes to the repository",
	Long:  "Open the commitizen TUI to compose a standardised commit message, then commit using go-git.",
	RunE:  commitRunE,
}

func init() {
	f := CommitCmd.Flags()
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
}

func commitRunE(_ *cobra.Command, _ []string) error {
	if ok, err := git.IsCurrentDirectoryGitRepo(); err != nil || !ok {
		if err != nil {
			return fmt.Errorf("does not seem to be a git repo: %w", err)
		}

		return fmt.Errorf("not a git repository")
	}

	defaults := commit.FormOptions{
		All:        commitAll,
		Amend:      commitAmend,
		NoVerify:   commitNoVerify,
		Signoff:    commitSignoff,
		AllowEmpty: commitAllowEmpty,
		Author:     commitAuthor,
	}

	authors, err := git.Authors()
	if err != nil {
		log.Printf("could not load author list: %v", err)
		authors = []string{}
	}

	msg, opts, err := commit.FillOutForm(defaults, authors)
	if err != nil {
		return err
	}

	return git.Commit(msg, git.CommitOptions{
		All:        opts.All,
		Amend:      opts.Amend,
		NoVerify:   opts.NoVerify,
		Signoff:    opts.Signoff,
		AllowEmpty: opts.AllowEmpty,
		Author:     opts.Author,
	})
}
