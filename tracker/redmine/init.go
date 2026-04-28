package redmine

import "github.com/lintingzhen/commitizen-go/tracker"

func init() {
	tracker.Register(trackerType, New)
}
