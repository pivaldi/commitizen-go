package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/lintingzhen/commitizen-go/store"
)

func openTestBranchStore(t *testing.T) *store.Store {
	t.Helper()

	s, err := store.Open(t.Context(), t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	return s
}

func TestBranchList_json(t *testing.T) {
	t.Parallel()

	s := openTestBranchStore(t)
	if err := s.InsertIssueWithBranch(t.Context(),
		&store.Issue{IDSlug: "ABC-42", Title: "Add OAuth login", StatusID: 1},
		&store.Branch{UUID: "550e8400", Name: "ABC-42@feat@add-oauth-login@550e8400", Type: "feat", StatusID: 1},
	); err != nil {
		t.Fatalf("insert: %v", err)
	}

	var buf bytes.Buffer
	if err := runBranchList(t.Context(), &buf, s, branchListFlags{jsonOut: true}); err != nil {
		t.Fatalf("runBranchList: %v", err)
	}

	var rows []store.BranchRow
	if err := json.Unmarshal(buf.Bytes(), &rows); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0].IssueSlug != "ABC-42" {
		t.Errorf("IssueSlug = %q, want ABC-42", rows[0].IssueSlug)
	}
	if rows[0].BranchName != "ABC-42@feat@add-oauth-login@550e8400" {
		t.Errorf("BranchName = %q", rows[0].BranchName)
	}
	if string(rows[0].Status) != "in_progress" {
		t.Errorf("Status = %q, want in_progress", rows[0].Status)
	}
}

func TestBranchList_json_emptyStore(t *testing.T) {
	t.Parallel()

	s := openTestBranchStore(t)

	var buf bytes.Buffer
	if err := runBranchList(t.Context(), &buf, s, branchListFlags{jsonOut: true}); err != nil {
		t.Fatalf("runBranchList: %v", err)
	}

	var rows []store.BranchRow
	if err := json.Unmarshal(buf.Bytes(), &rows); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("got %d rows, want 0", len(rows))
	}
}

func TestBranchList_stdout(t *testing.T) {
	t.Parallel()

	s := openTestBranchStore(t)
	if err := s.InsertIssueWithBranch(t.Context(),
		&store.Issue{IDSlug: "XY-1", Title: "Some feature", StatusID: 1},
		&store.Branch{UUID: "aabbccdd", Name: "XY-1@feat@some-feature@aabbccdd", Type: "feat", StatusID: 1},
	); err != nil {
		t.Fatalf("insert: %v", err)
	}

	var buf bytes.Buffer
	if err := runBranchList(t.Context(), &buf, s, branchListFlags{stdout: true}); err != nil {
		t.Fatalf("runBranchList: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "ISSUE ID") {
		t.Errorf("output missing ISSUE ID header: %q", out)
	}
	if !strings.Contains(out, "XY-1") {
		t.Errorf("output missing issue slug XY-1: %q", out)
	}
}

func TestBranchList_stdout_emptyStore(t *testing.T) {
	t.Parallel()

	s := openTestBranchStore(t)

	var buf bytes.Buffer
	if err := runBranchList(t.Context(), &buf, s, branchListFlags{stdout: true}); err != nil {
		t.Fatalf("runBranchList: %v", err)
	}

	if !strings.Contains(buf.String(), "No branches found.") {
		t.Errorf("expected 'No branches found.', got: %q", buf.String())
	}
}
