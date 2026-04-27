package store

import (
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.Context(), t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	return s
}

func TestOpen_createsMigrations(t *testing.T) {
	t.Parallel()

	s := openTestStore(t)

	// Verify all three tables exist by querying sqlite_master.
	tables := []string{"statuses", "issues", "branches"}
	for _, tbl := range tables {
		var name string
		row := s.db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl,
		)
		if err := row.Scan(&name); err != nil {
			t.Errorf("table %q not found: %v", tbl, err)
		}
	}
}

func TestMigration_idempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s1, err := Open(t.Context(), dir)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	_ = s1.Close()

	s2, err := Open(t.Context(), dir)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	_ = s2.Close()
}

func TestInsertIssueWithBranch(t *testing.T) {
	t.Parallel()

	s := openTestStore(t)

	issue := Issue{IDSlug: "ABC-42", Title: "Add OAuth Login", StatusID: 1}
	branch := Branch{
		UUID:     "550e8400",
		Name:     "ABC-42@feat@add-oauth-login@550e8400",
		Type:     "feat",
		StatusID: 1,
	}

	if err := s.InsertIssueWithBranch(t.Context(), &issue, &branch); err != nil {
		t.Fatalf("InsertIssueWithBranch: %v", err)
	}

	// Verify the branch row exists.
	var name string
	row := s.db.QueryRow("SELECT name FROM branches WHERE uuid = ?", "550e8400")
	if err := row.Scan(&name); err != nil {
		t.Fatalf("branch not found: %v", err)
	}
	if name != "ABC-42@feat@add-oauth-login@550e8400" {
		t.Errorf("name = %q, want %q", name, "ABC-42@feat@add-oauth-login@550e8400")
	}

	// Verify the issue row exists.
	var idSlug string
	row = s.db.QueryRow("SELECT id_slug FROM issues WHERE id = (SELECT issue_id FROM branches WHERE uuid = ?)", "550e8400")
	if err := row.Scan(&idSlug); err != nil {
		t.Fatalf("issue not found: %v", err)
	}
	if idSlug != "ABC-42" {
		t.Errorf("id_slug = %q, want %q", idSlug, "ABC-42")
	}
}

func TestUpdateBranchStatus_merged(t *testing.T) {
	t.Parallel()

	s := openTestStore(t)

	issue := Issue{IDSlug: "ABC-1", Title: "Some issue", StatusID: 1}
	branch := Branch{UUID: "aabbccdd", Name: "ABC-1@fix@some-issue@aabbccdd", Type: "fix", StatusID: 1}
	if err := s.InsertIssueWithBranch(t.Context(), &issue, &branch); err != nil {
		t.Fatalf("InsertIssueWithBranch: %v", err)
	}

	// Updating to merged without merged_at must fail (trigger).
	if err := s.UpdateBranchStatus(t.Context(), "aabbccdd", 2, nil); err == nil {
		t.Error("expected error when merged_at is nil for merged status, got nil")
	}

	// Updating to merged with merged_at must succeed.
	now := time.Now()
	if err := s.UpdateBranchStatus(t.Context(), "aabbccdd", 2, &now); err != nil {
		t.Errorf("UpdateBranchStatus merged: %v", err)
	}

	var statusID int64
	row := s.db.QueryRow("SELECT status_id FROM branches WHERE uuid = ?", "aabbccdd")
	if err := row.Scan(&statusID); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if statusID != 2 {
		t.Errorf("status_id = %d, want 2", statusID)
	}
}
