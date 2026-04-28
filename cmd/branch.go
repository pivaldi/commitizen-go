package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	btable "github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	lgtable "github.com/charmbracelet/lipgloss/table"
	"github.com/lintingzhen/commitizen-go/git"
	"github.com/lintingzhen/commitizen-go/store"
	"github.com/lintingzhen/commitizen-go/tui"
	"github.com/spf13/cobra"
)

type branchListFlags struct {
	status  string
	stdout  bool
	jsonOut bool
}

const (
	branchTableColWidthIssueID = 10
	branchTableColWidthTitle   = 28
	branchTableColWidthBranch  = 38
	branchTableColWidthType    = 8
	branchTableColWidthStatus  = 12
	branchTableColWidthCreated = 10
	branchTableHeight          = 20
	branchTableHeaderColor     = lipgloss.Color("63")
)

func getBranchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch",
		Short: "Manage local branches",
		RunE:  branchRunE,
	}
	cmd.AddCommand(getBranchListCmd(), getBranchNewCmd(), getBranchMergeCmd())

	return cmd
}

func branchRunE(cmd *cobra.Command, args []string) error {
	var action string
	if err := huh.NewForm(tui.BranchActionSelect(&action)).Run(); err != nil {
		return fmt.Errorf("action select: %w", err)
	}

	switch action {
	case tui.BranchActionNameList:
		// zero flags → TUI path (status filter presented interactively)
		return branchListRunE(cmd, branchListFlags{})
	case tui.BranchActionNameNew:
		return branchNewRunE(cmd, args)
	default:
		fmt.Println("Not yet implemented.")

		return nil
	}
}

func getBranchListCmd() *cobra.Command {
	var flags branchListFlags

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List branches",
	}

	f := cmd.Flags()
	f.StringVar(&flags.status, "status", "", "filter by status: in_progress, merged, all")
	f.BoolVar(&flags.stdout, "stdout", false, "print table to stdout without TUI")
	f.BoolVar(&flags.jsonOut, "json", false, "print JSON array to stdout")

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return branchListRunE(cmd, flags)
	}

	return cmd
}

func branchListRunE(cmd *cobra.Command, flags branchListFlags) error {
	client, err := git.NewClient()
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}

	root, err := client.WorkingTreeRoot()
	if err != nil {
		return fmt.Errorf("working tree root: %w", err)
	}

	s, err := store.Open(cmd.Context(), filepath.Join(root, ".git"))
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer func() { _ = s.Close() }()

	return runBranchList(cmd.Context(), os.Stdout, s, flags)
}

// runBranchList executes the branch list logic. w receives stdout/non-TUI output.
// When neither --json nor --stdout is set, runBranchList runs the interactive TUI.
func runBranchList(ctx context.Context, w io.Writer, s *store.Store, flags branchListFlags) error {
	queryStatus := toBranchStatus(flags.status)

	if flags.jsonOut {
		rows, err := s.ListBranches(ctx, queryStatus)
		if err != nil {
			return fmt.Errorf("list branches: %w", err)
		}
		if err := json.NewEncoder(w).Encode(rows); err != nil {
			return fmt.Errorf("encode json: %w", err)
		}

		return nil
	}

	if flags.stdout {
		rows, err := s.ListBranches(ctx, queryStatus)
		if err != nil {
			return fmt.Errorf("list branches: %w", err)
		}
		if len(rows) == 0 {
			fmt.Fprintln(w, "No branches found.")

			return nil
		}
		renderLipglossTable(w, rows)

		return nil
	}

	// TUI path: status filter then interactive table.
	statusStr := flags.status
	if err := huh.NewForm(tui.BranchStatusFilter(&statusStr, statusStr)).Run(); err != nil {
		return fmt.Errorf("status filter: %w", err)
	}

	queryStatus = toBranchStatus(statusStr)

	rows, err := s.ListBranches(ctx, queryStatus)
	if err != nil {
		return fmt.Errorf("list branches: %w", err)
	}

	if len(rows) == 0 {
		fmt.Fprintln(w, "No branches found.")

		return nil
	}

	return runBranchTable(rows)
}

func renderLipglossTable(w io.Writer, rows []store.BranchRow) {
	t := lgtable.New().
		Headers("ISSUE ID", "TITLE", "BRANCH", "TYPE", "STATUS", "CREATED").
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == lgtable.HeaderRow {
				return lipgloss.NewStyle().Bold(true)
			}

			return lipgloss.NewStyle()
		})

	for _, r := range rows {
		t.Row(r.IssueSlug, r.Title, r.BranchName, r.Type, string(r.Status), r.CreatedAt.Format("2006-01-02"))
	}

	fmt.Fprintln(w, t.Render())
}

func runBranchTable(rows []store.BranchRow) error {
	cols := []btable.Column{
		{Title: "Issue ID", Width: branchTableColWidthIssueID},
		{Title: "Title", Width: branchTableColWidthTitle},
		{Title: "Branch", Width: branchTableColWidthBranch},
		{Title: "Type", Width: branchTableColWidthType},
		{Title: "Status", Width: branchTableColWidthStatus},
		{Title: "Created", Width: branchTableColWidthCreated},
	}

	tableRows := make([]btable.Row, len(rows))
	for i, r := range rows {
		tableRows[i] = btable.Row{
			r.IssueSlug,
			r.Title,
			r.BranchName,
			r.Type,
			string(r.Status),
			r.CreatedAt.Format("2006-01-02"),
		}
	}

	t := btable.New(
		btable.WithColumns(cols),
		btable.WithRows(tableRows),
		btable.WithFocused(true),
		btable.WithHeight(branchTableHeight),
	)

	st := btable.DefaultStyles()
	st.Header = lipgloss.NewStyle().Bold(true).Foreground(branchTableHeaderColor).Padding(0, 1)
	t.SetStyles(st)

	if _, err := tea.NewProgram(branchTableModel{table: t}).Run(); err != nil {
		return fmt.Errorf("run table: %w", err)
	}

	return nil
}

func toBranchStatus(s string) store.BranchStatus {
	switch s {
	case "in_progress":
		return store.BranchStatusInProgress
	case "merged":
		return store.BranchStatusMerged
	case "all", "":
		return store.BranchStatusAll
	default:
		return store.BranchStatusAll
	}
}

// branchTableModel wraps bubbles/table as a minimal Bubble Tea program.
type branchTableModel struct {
	table btable.Model
}

func (m branchTableModel) Init() tea.Cmd { return nil }

func (m branchTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)

	return m, cmd
}

func (m branchTableModel) View() string {
	return m.table.View() + "\n\nPress q to quit."
}

func getBranchNewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new",
		Short: "Create a new branch (manual input)",
		Long:  "Enter issue details manually, then a named branch is created and checked out.",
		RunE:  branchNewRunE,
	}
}

// branchNewRunE delegates to issueStartRunE (manual input form).
// Step 2b: add toggle to tracker-picker.
func branchNewRunE(cmd *cobra.Command, args []string) error {
	return issueStartRunE(cmd, args)
}

func getBranchMergeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "merge",
		Short: "Merge a branch",
		RunE:  branchMergeRunE,
	}
}

func branchMergeRunE(_ *cobra.Command, _ []string) error {
	fmt.Println("Not yet implemented.")

	return nil
}
