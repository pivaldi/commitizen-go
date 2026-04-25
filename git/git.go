package git

import (
	"fmt"
	"sort"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
)

// CommitOptions configures Client.Commit.
type CommitOptions struct {
	All  bool
	Amend bool
	// NoVerify — go-git v6 does not execute hooks; reserved for a future subprocess fallback.
	NoVerify   bool
	Signoff    bool
	AllowEmpty bool
	Author     string // "Name <email>"; empty = git config identity
}

// Client wraps a go-git repository and exposes commit operations.
type Client struct {
	repo *gogit.Repository
}

// NewClient opens the git repository that contains the current directory.
func NewClient() (*Client, error) {
	repo, err := gogit.PlainOpenWithOptions(".", &gogit.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, fmt.Errorf("open git repository: %w", err)
	}

	return &Client{repo: repo}, nil
}

// WorkingTreeRoot returns the absolute path of the repository's working tree root.
func (c *Client) WorkingTreeRoot() (string, error) {
	wt, err := c.repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("get worktree: %w", err)
	}

	// billy.Filesystem embeds the Chroot interface which exposes Root().
	type rooter interface{ Root() string }
	if r, ok := wt.Filesystem.(rooter); ok {
		return r.Root(), nil
	}

	return "", fmt.Errorf("filesystem type %T does not expose Root()", wt.Filesystem)
}

// Authors returns a deduplicated, alphabetically sorted list of commit author strings
// ("Name <email>") from the repository history.
// The current git config identity is prepended as the first (default) entry.
func (c *Client) Authors() ([]string, error) {
	iter, err := c.repo.Log(&gogit.LogOptions{})
	if err != nil {
		// empty repo (no commits yet) — not an error
		return []string{}, nil
	}

	seen := make(map[string]struct{})
	var list []string
	if err := iter.ForEach(func(commit *object.Commit) error {
		entry := commit.Author.Name + " <" + commit.Author.Email + ">"
		if _, ok := seen[entry]; !ok {
			seen[entry] = struct{}{}
			list = append(list, entry)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk commits: %w", err)
	}

	sort.Strings(list)

	cfg, err := c.repo.Config()
	if err == nil && cfg.User.Name != "" {
		current := cfg.User.Name + " <" + cfg.User.Email + ">"
		filtered := make([]string, 0, len(list))
		for _, a := range list {
			if a != current {
				filtered = append(filtered, a)
			}
		}
		list = append([]string{current}, filtered...)
	}

	return list, nil
}

// Commit records a commit with msg and the given options.
func (c *Client) Commit(msg []byte, opts CommitOptions) error {
	wt, err := c.repo.Worktree()
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}

	finalMsg := string(msg)
	if opts.Signoff {
		signer := opts.Author
		if signer == "" {
			if cfg, err := c.repo.Config(); err == nil && cfg.User.Name != "" {
				signer = cfg.User.Name + " <" + cfg.User.Email + ">"
			}
		}
		if signer != "" {
			finalMsg = strings.TrimRight(finalMsg, "\n") + "\n\nSigned-off-by: " + signer
		}
	}

	// CommitOptions.All stages only tracked modified/deleted files, matching
	// `git commit --all` semantics (untracked files are NOT included).
	commitOpts := &gogit.CommitOptions{
		All:               opts.All,
		AllowEmptyCommits: opts.AllowEmpty,
		Amend:             opts.Amend,
	}

	if opts.Author != "" {
		name, email := parseAuthor(opts.Author)
		commitOpts.Author = &object.Signature{
			Name:  name,
			Email: email,
			When:  time.Now(),
		}
	}

	if _, err = wt.Commit(finalMsg, commitOpts); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

// parseAuthor splits "Name <email>" into name and email parts.
func parseAuthor(s string) (name, email string) {
	lt := strings.LastIndex(s, "<")
	gt := strings.LastIndex(s, ">")
	if lt >= 0 && gt > lt {
		return strings.TrimSpace(s[:lt]), s[lt+1 : gt]
	}

	return s, ""
}
