package branch

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

var (
	reSpaces           = regexp.MustCompile(`\s+`)
	nonAlphanumHyphen  = regexp.MustCompile(`[^a-z0-9-]+`)
	multiHyphen        = regexp.MustCompile(`-{2,}`)
)

func Slug(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = reSpaces.ReplaceAllString(s, "-")
	s = nonAlphanumHyphen.ReplaceAllString(s, "")
	s = multiHyphen.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")

	return s
}

func ShortUUID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("branch.ShortUUID: crypto/rand failed: %v", err))
	}

	return hex.EncodeToString(b)
}

func Name(issueID, branchType, title string) (string, error) {
	slug := Slug(title)
	if slug == "" {
		return "", fmt.Errorf("branch.Name: title %q produces an empty slug", title)
	}

	return issueID + "@" + branchType + "@" + slug + "@" + ShortUUID(), nil
}

func Parse(name string) (issueID, branchType, title, uuid string, err error) {
	parts := strings.Split(name, "@")
	if len(parts) != 4 {
		return "", "", "", "", fmt.Errorf("branch name %q: expected 4 parts, got %d", name, len(parts))
	}

	return parts[0], parts[1], parts[2], parts[3], nil
}
