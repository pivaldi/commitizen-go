package git

import (
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/storage/memory"

	// go-git v6 depends on go-billy/v6.
	"github.com/go-git/go-billy/v6/memfs"
)

// newTestRepo creates an in-memory git repository with one initial commit.
// User is configured as "Test User <test@example.com>" in the repo config so
// that commits without an explicit Author succeed (go-git v6 requires an author
// identity to be resolvable from config when none is supplied explicitly).
func newTestRepo(t *testing.T) *gogit.Repository {
	t.Helper()

	repo, err := gogit.Init(memory.NewStorage(), gogit.WithWorkTree(memfs.New()))
	if err != nil {
		t.Fatalf("init in-memory repo: %v", err)
	}

	// Configure user identity so commits without explicit Author work.
	cfg, err := repo.Config()
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	cfg.User.Name = "Test User"
	cfg.User.Email = "test@example.com"
	if err := repo.SetConfig(cfg); err != nil {
		t.Fatalf("set config: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	f, err := wt.Filesystem.Create("README.md")
	if err != nil {
		t.Fatalf("create README.md: %v", err)
	}
	_, _ = f.Write([]byte("# test"))
	_ = f.Close()

	if _, err := wt.Add("README.md"); err != nil {
		t.Fatalf("stage README.md: %v", err)
	}

	_, err = wt.Commit("chore: init", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("initial commit: %v", err)
	}

	return repo
}

// withRepo overrides the package-level openRepoFn for the duration of t.
func withRepo(t *testing.T, repo *gogit.Repository) {
	t.Helper()

	orig := openRepoFn
	openRepoFn = func() (*gogit.Repository, error) { return repo, nil }
	t.Cleanup(func() { openRepoFn = orig })
}

// stageNewFile creates filename in wt, writes content, and stages it.
func stageNewFile(t *testing.T, wt *gogit.Worktree, filename, content string) {
	t.Helper()

	f, err := wt.Filesystem.Create(filename)
	if err != nil {
		t.Fatalf("create %s: %v", filename, err)
	}
	_, _ = f.Write([]byte(content))
	_ = f.Close()

	if _, err := wt.Add(filename); err != nil {
		t.Fatalf("stage %s: %v", filename, err)
	}
}

func TestCommit_basic(t *testing.T) {
	repo := newTestRepo(t)
	wt, _ := repo.Worktree()
	stageNewFile(t, wt, "file.txt", "hello")
	withRepo(t, repo)

	if err := Commit([]byte("feat: basic commit"), CommitOptions{}); err != nil {
		t.Fatalf("Commit error: %v", err)
	}

	ref, err := repo.Head()
	if err != nil {
		t.Fatalf("repo.Head: %v", err)
	}
	c, err := repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatalf("CommitObject: %v", err)
	}
	if c.Message != "feat: basic commit" {
		t.Errorf("got message %q, want %q", c.Message, "feat: basic commit")
	}
}

func TestCommit_all_stagesTrackedOnly(t *testing.T) {
	repo := newTestRepo(t)
	wt, _ := repo.Worktree()

	// Modify the tracked file (README.md was in the initial commit).
	f, _ := wt.Filesystem.Create("README.md")
	_, _ = f.Write([]byte("# modified"))
	_ = f.Close()

	// Create an untracked file — must NOT end up in the commit.
	u, _ := wt.Filesystem.Create("untracked.txt")
	_, _ = u.Write([]byte("should not be staged"))
	_ = u.Close()

	withRepo(t, repo)

	if err := Commit([]byte("chore: all flag"), CommitOptions{All: true}); err != nil {
		t.Fatalf("Commit error: %v", err)
	}

	ref, err := repo.Head()
	if err != nil {
		t.Fatalf("repo.Head: %v", err)
	}
	c, err := repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatalf("CommitObject: %v", err)
	}
	tree, err := repo.TreeObject(c.TreeHash)
	if err != nil {
		t.Fatalf("TreeObject: %v", err)
	}

	if _, err := tree.File("README.md"); err != nil {
		t.Error("README.md not found in commit tree")
	}
	if _, err := tree.File("untracked.txt"); err == nil {
		t.Error("untracked.txt must not be in commit tree")
	}
}

func TestCommit_signoff(t *testing.T) {
	repo := newTestRepo(t)
	wt, _ := repo.Worktree()
	stageNewFile(t, wt, "file.txt", "x")
	withRepo(t, repo)

	err := Commit([]byte("docs: readme"), CommitOptions{
		Signoff: true,
		Author:  "Alice Dev <alice@example.com>",
	})
	if err != nil {
		t.Fatalf("Commit error: %v", err)
	}

	ref, err := repo.Head()
	if err != nil {
		t.Fatalf("repo.Head: %v", err)
	}
	c, err := repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatalf("CommitObject: %v", err)
	}
	if !strings.Contains(c.Message, "Signed-off-by: Alice Dev <alice@example.com>") {
		t.Errorf("signoff trailer not found in: %q", c.Message)
	}
}

func TestCommit_author(t *testing.T) {
	repo := newTestRepo(t)
	wt, _ := repo.Worktree()
	stageNewFile(t, wt, "file.txt", "x")
	withRepo(t, repo)

	err := Commit([]byte("fix: author override"), CommitOptions{
		Author: "Bob Builder <bob@example.com>",
	})
	if err != nil {
		t.Fatalf("Commit error: %v", err)
	}

	ref, err := repo.Head()
	if err != nil {
		t.Fatalf("repo.Head: %v", err)
	}
	c, err := repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatalf("CommitObject: %v", err)
	}
	if c.Author.Name != "Bob Builder" || c.Author.Email != "bob@example.com" {
		t.Errorf("author: got %q <%s>, want Bob Builder <bob@example.com>",
			c.Author.Name, c.Author.Email)
	}
}

func TestCommit_amend(t *testing.T) {
	repo := newTestRepo(t)
	wt, _ := repo.Worktree()
	stageNewFile(t, wt, "file.txt", "x")
	withRepo(t, repo)

	// Second commit — the one we will amend.
	if err := Commit([]byte("feat: to be amended"), CommitOptions{}); err != nil {
		t.Fatalf("Commit error: %v", err)
	}

	iter, err := repo.Log(&gogit.LogOptions{})
	if err != nil {
		t.Fatalf("repo.Log: %v", err)
	}
	countBefore := 0
	_ = iter.ForEach(func(_ *object.Commit) error { countBefore++; return nil })

	// Amend: replace the tip commit message.
	if err := Commit([]byte("feat: amended message"), CommitOptions{Amend: true}); err != nil {
		t.Fatalf("amend error: %v", err)
	}

	iter2, err := repo.Log(&gogit.LogOptions{})
	if err != nil {
		t.Fatalf("repo.Log: %v", err)
	}
	countAfter := 0
	_ = iter2.ForEach(func(_ *object.Commit) error { countAfter++; return nil })
	if countAfter != countBefore {
		t.Errorf("commit count changed: %d → %d (expected no change)", countBefore, countAfter)
	}

	ref, err := repo.Head()
	if err != nil {
		t.Fatalf("repo.Head: %v", err)
	}
	c, err := repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatalf("CommitObject: %v", err)
	}
	if c.Message != "feat: amended message" {
		t.Errorf("tip message after amend: got %q", c.Message)
	}
}
