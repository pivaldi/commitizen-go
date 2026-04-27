package branch

import (
	"regexp"
	"strings"
	"testing"
)

func TestSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Add OAuth Login", "add-oauth-login"},
		{"  leading and trailing  ", "leading-and-trailing"},
		{"special!@#chars", "specialchars"},
		{"multiple   spaces", "multiple-spaces"},
		{"already-kebab", "already-kebab"},
		{"UPPERCASE", "uppercase"},
		{"feat/scope", "featscope"},
		{"", ""},
		{"!!!", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := Slug(tt.input); got != tt.want {
				t.Errorf("Slug(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestShortUUID(t *testing.T) {
	t.Parallel()

	a := shortUUID()
	b := shortUUID()

	if len(a) != 8 {
		t.Errorf("ShortUUID len = %d, want 8", len(a))
	}
	if strings.ContainsAny(a, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		t.Errorf("ShortUUID %q contains uppercase", a)
	}
	if a == b {
		t.Errorf("two consecutive ShortUUID calls returned the same value: %q", a)
	}
}

func TestName(t *testing.T) {
	t.Parallel()

	// Name calls ShortUUID internally, so we only check structure.
	b, err := New("ABC-42", "feat", "Add OAuth Login")
	if err != nil {
		t.Fatalf("Name error: %v", err)
	}

	n := b.Name()
	parts := strings.Split(n, "@")
	if len(parts) != 4 {
		t.Fatalf("Name produced %d parts, want 4: %q", len(parts), n)
	}
	if parts[0] != "ABC-42" {
		t.Errorf("parts[0] = %q, want %q", parts[0], "ABC-42")
	}
	if parts[1] != "feat" {
		t.Errorf("parts[1] = %q, want %q", parts[1], "feat")
	}
	if parts[2] != "add-oauth-login" {
		t.Errorf("parts[2] = %q, want %q", parts[2], "add-oauth-login")
	}
	if len(parts[3]) != 8 {
		t.Errorf("parts[3] len = %d, want 8", len(parts[3]))
	}
	if matched, _ := regexp.MatchString(`^[0-9a-f]{8}$`, parts[3]); !matched {
		t.Errorf("parts[3] %q is not 8 lowercase hex chars", parts[3])
	}
}

func TestName_emptySlug(t *testing.T) {
	t.Parallel()

	_, err := New("ABC-42", "feat", "!!!")
	if err == nil {
		t.Error("branch with all-punctuation title: expected error, got nil")
	}
}

func TestName_emptyType(t *testing.T) {
	t.Parallel()

	_, err := New("ABC-42", "", "branch title")
	if err == nil {
		t.Error("branch with empty type: expected error, got nil")
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	b, err := Parse("ABC-42@feat@add-oauth-login@550e8400")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if b.issueID != "ABC-42" {
		t.Errorf("issueID = %q, want %q", b.issueID, "ABC-42")
	}
	if b.btype != "feat" {
		t.Errorf("branchType = %q, want %q", b.btype, "feat")
	}
	if b.title != "add-oauth-login" {
		t.Errorf("title = %q, want %q", b.title, "add-oauth-login")
	}
	if b.id != "550e8400" {
		t.Errorf("uuid = %q, want %q", b.id, "550e8400")
	}
}

func TestParse_invalid(t *testing.T) {
	t.Parallel()

	cases := []string{
		"",
		"no-separators",
		"only@two",
		"one@two@three",
		"one@two@three@four@five",
	}
	for _, c := range cases {
		if _, err := Parse(c); err == nil {
			t.Errorf("Parse(%q) expected error, got nil", c)
		}
	}
}
