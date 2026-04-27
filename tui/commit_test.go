package tui

import "testing"

func TestCommitOption_anyOptionSet(t *testing.T) {
	cases := []struct {
		opts CommitOption
		want bool
	}{
		{CommitOption{}, false},
		{CommitOption{Authors: []string{"Alice"}}, false},
		{CommitOption{All: true}, true},
		{CommitOption{Amend: true}, true},
		{CommitOption{NoVerify: true}, true},
		{CommitOption{Signoff: true}, true},
		{CommitOption{AllowEmpty: true}, true},
		{CommitOption{Author: "Alice <a@b.com>"}, true},
	}
	for _, tc := range cases {
		if got := tc.opts.AnyOptionSet(); got != tc.want {
			t.Errorf("%+v: anyOptionSet()=%v, want %v", tc.opts, got, tc.want)
		}
	}
}
