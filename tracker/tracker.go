package tracker

import (
	"context"
	"fmt"
	"sync"
)

// Issue is the tracker-agnostic representation of a work item.
type Issue struct {
	TrackerType string
	ID          string
	Subject     string
	Description string
	Status      string
}

// Config holds the connection parameters for one tracker instance.
type Config struct {
	Type             string
	URL              string
	Token            string
	InProgressStatus string
}

// Tracker is the contract every adapter must satisfy.
type Tracker interface {
	ListIssues(ctx context.Context) ([]Issue, error)
	UpdateIssueStatus(ctx context.Context, issueID, statusName string) error
}

var (
	registryMu sync.RWMutex
	registry   = map[string]func(Config) (Tracker, error){}
)

// Register adds a factory function for the named tracker type.
func Register(name string, fn func(Config) (Tracker, error)) {
	registryMu.Lock()
	defer registryMu.Unlock()

	registry[name] = fn
}

// New constructs a Tracker from cfg using the registered factory.
func New(cfg Config) (Tracker, error) {
	registryMu.RLock()
	fn, ok := registry[cfg.Type]
	registryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown tracker type %q: adapter not registered", cfg.Type)
	}

	return fn(cfg)
}
