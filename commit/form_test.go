package commit

import (
	"bytes"
	"testing"
)

func TestAssembleMessage(t *testing.T) {
	tests := []struct {
		name    string
		tmpl    string
		answers map[string]interface{}
		want    string
	}{
		{
			name: "type with scope and subject",
			tmpl: "{{.type}}{{with .scope}}({{.}}){{end}}: {{.subject}}",
			answers: map[string]interface{}{
				"type": "feat", "scope": "auth", "subject": "add login",
			},
			want: "feat(auth): add login",
		},
		{
			name: "empty scope omitted",
			tmpl: "{{.type}}{{with .scope}}({{.}}){{end}}: {{.subject}}",
			answers: map[string]interface{}{
				"type": "fix", "scope": "", "subject": "fix nil panic",
			},
			want: "fix: fix nil panic",
		},
		{
			name: "trims leading and trailing whitespace from subject",
			tmpl: "{{.type}}: {{.subject}}",
			answers: map[string]interface{}{
				"type": "docs", "subject": "  update readme  ",
			},
			want: "docs: update readme",
		},
		{
			name: "full conventional commit with body and footer",
			tmpl: "{{.type}}: {{.subject}}{{with .body}}\n\n{{.}}{{end}}{{with .footer}}\n\n{{.}}{{end}}",
			answers: map[string]interface{}{
				"type":    "feat",
				"subject": "add oauth",
				"body":    "implements google oauth flow",
				"footer":  "BREAKING CHANGE: removes basic auth",
			},
			want: "feat: add oauth\n\nimplements google oauth flow\n\nBREAKING CHANGE: removes basic auth",
		},
		{
			name: "empty body and footer omitted",
			tmpl: "{{.type}}: {{.subject}}{{with .body}}\n\n{{.}}{{end}}{{with .footer}}\n\n{{.}}{{end}}",
			answers: map[string]interface{}{
				"type": "chore", "subject": "update deps", "body": "", "footer": "",
			},
			want: "chore: update deps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := assembleMessage(&buf, tt.tmpl, tt.answers); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildAuthorList_deduplication(t *testing.T) {
	all := []string{
		"Alice <alice@example.com>",
		"Bob <bob@example.com>",
		"Alice <alice@example.com>",
	}
	got := BuildAuthorList(all, "")
	want := []string{"Alice <alice@example.com>", "Bob <bob@example.com>"}
	if len(got) != len(want) {
		t.Fatalf("len: got %d, want %d — list: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBuildAuthorList_sortOrder(t *testing.T) {
	all := []string{
		"Zoe <zoe@example.com>",
		"Alice <alice@example.com>",
		"Mia <mia@example.com>",
	}
	got := BuildAuthorList(all, "")
	for i := 1; i < len(got); i++ {
		if got[i] < got[i-1] {
			t.Errorf("not sorted at [%d]: %q < %q", i, got[i], got[i-1])
		}
	}
}

func TestBuildAuthorList_currentUserFirst(t *testing.T) {
	all := []string{
		"Alice <alice@example.com>",
		"Bob <bob@example.com>",
		"Current User <current@example.com>",
	}
	current := "Current User <current@example.com>"
	got := BuildAuthorList(all, current)

	if len(got) == 0 {
		t.Fatal("empty list")
	}
	if got[0] != current {
		t.Errorf("first entry: got %q, want %q", got[0], current)
	}
	for _, a := range got[1:] {
		if a == current {
			t.Errorf("current user duplicated in list: %v", got)
		}
	}
}

func TestBuildAuthorList_currentUserNotInHistory(t *testing.T) {
	all := []string{"Alice <alice@example.com>", "Bob <bob@example.com>"}
	current := "New User <new@example.com>"
	got := BuildAuthorList(all, current)
	if got[0] != current {
		t.Errorf("first entry: got %q, want %q", got[0], current)
	}
	if len(got) != 3 {
		t.Errorf("len: got %d, want 3 — list: %v", len(got), got)
	}
}

func TestFormOptions_anyOptionSet(t *testing.T) {
	cases := []struct {
		opts FormOptions
		want bool
	}{
		{FormOptions{}, false},
		{FormOptions{All: true}, true},
		{FormOptions{Amend: true}, true},
		{FormOptions{NoVerify: true}, true},
		{FormOptions{Signoff: true}, true},
		{FormOptions{AllowEmpty: true}, true},
		{FormOptions{Author: "Alice <a@b.com>"}, true},
	}
	for _, tc := range cases {
		if got := tc.opts.anyOptionSet(); got != tc.want {
			t.Errorf("%+v: anyOptionSet()=%v, want %v", tc.opts, got, tc.want)
		}
	}
}

func TestBuildAuthorList_nilInput(t *testing.T) {
	got := BuildAuthorList(nil, "")
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
	got2 := BuildAuthorList(nil, "Alice <alice@example.com>")
	if len(got2) != 1 || got2[0] != "Alice <alice@example.com>" {
		t.Errorf("expected [Alice], got %v", got2)
	}
}

func TestBuildAuthorList_currentIsOnlyEntry(t *testing.T) {
	got := BuildAuthorList([]string{"Alice <alice@example.com>"}, "Alice <alice@example.com>")
	if len(got) != 1 || got[0] != "Alice <alice@example.com>" {
		t.Errorf("expected [Alice], got %v", got)
	}
}

func TestBuildAuthorList_currentAppearsMultipleTimes(t *testing.T) {
	got := BuildAuthorList(
		[]string{"Alice <alice@example.com>", "Alice <alice@example.com>"},
		"Alice <alice@example.com>",
	)
	if len(got) != 1 || got[0] != "Alice <alice@example.com>" {
		t.Errorf("expected [Alice] (len 1), got %v", got)
	}
}
