package redmine

import "github.com/lintingzhen/commitizen-go/tracker"

//nolint:gochecknoinits // Register pattern need it
func init() {
	tracker.Register(trackerType, New)
}
