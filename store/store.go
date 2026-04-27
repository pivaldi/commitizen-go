package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"time"

	_ "modernc.org/sqlite" // Import for side-effect: registers sqlite driver
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Store wraps a SQLite database for branch and issue persistence.
type Store struct {
	db *sql.DB
}

// Issue represents a tracked issue record.
type Issue struct {
	ID       int64
	IDSlug   string // tracker string ID: "ABC-42", "42", …
	Title    string
	StatusID int64
}

// Branch represents a tracked branch record.
type Branch struct {
	UUID      string
	Name      string
	IssueID   int64
	Type      string
	StatusID  int64
	CreatedAt time.Time
	MergedAt  *time.Time
}

// Open opens (or creates) the SQLite database at dir/git-cz.db and runs pending migrations.
func Open(ctx context.Context, dir string) (*Store, error) {
	path := filepath.Join(dir, "git-cz.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(1)

	// Enable foreign keys.
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("migrate: %w", err)
	}

	return s, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("close db: %w", err)
	}

	return nil
}

// InsertIssueWithBranch inserts an issue and its linked branch in a single transaction.
func (s *Store) InsertIssueWithBranch(ctx context.Context, issue *Issue, branch *Branch) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		`INSERT INTO issues (id_slug, title, status_id) VALUES (?, ?, ?)`,
		issue.IDSlug, issue.Title, issue.StatusID,
	)
	if err != nil {
		return fmt.Errorf("insert issue: %w", err)
	}

	issueID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("last insert id: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO branches (uuid, name, issue_id, type, status_id) VALUES (?, ?, ?, ?, ?)`,
		branch.UUID, branch.Name, issueID, branch.Type, branch.StatusID,
	)
	if err != nil {
		return fmt.Errorf("insert branch: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

// UpdateBranchStatus updates a branch's status. mergedAt must be non-nil when statusID == 2 (merged).
func (s *Store) UpdateBranchStatus(ctx context.Context, uuid string, statusID int64, mergedAt *time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE branches SET status_id = ?, merged_at = ? WHERE uuid = ?`,
		statusID, mergedAt, uuid,
	)
	if err != nil {
		return fmt.Errorf("update branch status: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("update branch status: no branch with uuid %q", uuid)
	}

	return nil
}

// UpdateIssueStatus updates an issue's status.
func (s *Store) UpdateIssueStatus(ctx context.Context, issueID, statusID int64) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE issues SET status_id = ? WHERE id = ?`,
		statusID, issueID,
	)
	if err != nil {
		return fmt.Errorf("update issue status: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("update issue status: no issue with id %d", issueID)
	}

	return nil
}

func (s *Store) migrate(ctx context.Context) error {
	var version int
	if err := s.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("read user_version: %w", err)
	}

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}

	slices.Sort(names)

	for i, name := range names {
		if i < version {
			continue
		}

		content, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration tx %s: %w", name, err)
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			_ = tx.Rollback()

			return fmt.Errorf("exec migration %s: %w", name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}

		// PRAGMA user_version does not support parameter binding; i is compile-time-bounded.
		// Executed after Commit because PRAGMA user_version is not transactional in SQLite.
		if _, err := s.db.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", i+1)); err != nil {
			return fmt.Errorf("bump user_version: %w", err)
		}
	}

	return nil
}
