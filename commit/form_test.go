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
