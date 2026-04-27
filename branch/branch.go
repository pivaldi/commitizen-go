package branch

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	reSpaces          = regexp.MustCompile(`\s+`)
	nonAlphanumHyphen = regexp.MustCompile(`[^a-z0-9-]+`)
	multiHyphen       = regexp.MustCompile(`-{2,}`)
)

type Branch struct {
	issueID string
	btype   string
	title   string
	id      string
}

func New(issueID, branchType, title string) (*Branch, error) {
	slug := Slug(title)
	if slug == "" {
		return nil, fmt.Errorf("branch.Name: title %q produces an empty slug", title)
	}

	if branchType == "" {
		return nil, errors.New("branche type can not be empty")
	}

	return &Branch{
		issueID: issueID,
		btype:   branchType,
		title:   slug,
		id:      shortUUID(),
	}, nil
}

func (b Branch) Name() string {
	return b.issueID + "@" + b.btype + "@" + b.title + "@" + b.id
}

func (b Branch) IssueID() string {
	return b.issueID
}

func (b Branch) Title() string {
	return b.title
}

func (b Branch) ID() string {
	return b.id
}

func (b Branch) Type() string {
	return b.btype
}

func Slug(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = reSpaces.ReplaceAllString(s, "-")
	s = nonAlphanumHyphen.ReplaceAllString(s, "")
	s = multiHyphen.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")

	return s
}

func shortUUID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("branch.ShortUUID: crypto/rand failed: %v", err))
	}

	return hex.EncodeToString(b)
}

func Parse(name string) (*Branch, error) {
	parts := strings.Split(name, "@")
	if len(parts) != 4 {
		return nil, fmt.Errorf("branch name %q: expected 4 parts, got %d", name, len(parts))
	}

	b, err := New(parts[0], parts[1], parts[2])
	if err != nil {
		return nil, err
	}

	b.id = parts[3]

	return b, nil
}
