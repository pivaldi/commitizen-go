package commit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"strings"
	"text/template"

	"github.com/charmbracelet/huh"
	"github.com/lintingzhen/commitizen-go/tui"
)

// DefaultMessageConfig returns the parsed built-in message configuration.
// Callers may overlay user config on top before passing it to FillOutForm.
func DefaultMessageConfig() (tui.CommitMessageConfig, error) {
	config := struct{ Message tui.CommitMessageConfig }{}
	if err := json.Unmarshal([]byte(defaultConfig), &config); err != nil {
		return tui.CommitMessageConfig{}, fmt.Errorf("parse default config: %w", err)
	}

	return config.Message, nil
}

// FillOutForm presents the commit TUI form.
// Group 1: commit message fields (type, scope, subject, body, footer).
// Group 2: commit options (author, all, amend, no-verify, signoff, allow-empty).
//
//	Group 2 is skipped when defaults.anyOptionSet() is true (flags were passed).
//
// Returns the assembled commit message bytes and the (possibly user-modified) options.
func FillOutForm(cfg tui.CommitMessageConfig, defaults tui.CommitOption) ([]byte, tui.CommitOption, error) {
	form, extractMsg, extractOpts := loadForm(cfg, defaults)
	tmplText := cfg.Template

	if err := form.Run(); err != nil {
		return nil, tui.CommitOption{}, fmt.Errorf("failed to run the form: %w", err)
	}

	answers := extractMsg()
	opts := extractOpts()

	var buf bytes.Buffer
	if err := assembleMessage(&buf, tmplText, answers); err != nil {
		log.Printf("assemble failed, err=%v\n", err)

		return nil, tui.CommitOption{}, fmt.Errorf("assemble message: %w", err)
	}

	return buf.Bytes(), opts, nil
}

// BuildAuthorList deduplicates all (may contain duplicates), sorts alphabetically,
// then prepends current as the first entry (removing it from its sorted position if present).
// If current is empty, the sorted deduplicated list is returned as-is.
func BuildAuthorList(all []string, current string) []string {
	seen := make(map[string]struct{})
	var unique []string
	for _, a := range all {
		if _, ok := seen[a]; !ok {
			seen[a] = struct{}{}
			unique = append(unique, a)
		}
	}
	slices.Sort(unique)

	if current == "" {
		return unique
	}

	filtered := make([]string, 0, len(unique))
	for _, a := range unique {
		if a != current {
			filtered = append(filtered, a)
		}
	}

	return append([]string{current}, filtered...)
}

// assembleMessage trims whitespace from all string answers, then executes tmplText writing the result to buf.
func assembleMessage(buf *bytes.Buffer, tmplText string, answers map[string]any) error {
	tmpl, err := template.New("").Parse(tmplText)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}
	for k, v := range answers {
		if s, ok := v.(string); ok {
			answers[k] = strings.TrimSpace(s)
		}
	}

	err = tmpl.Execute(buf, answers)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

func loadForm(
	cfg tui.CommitMessageConfig,
	defaults tui.CommitOption,
) (form *huh.Form, extractMsg func() map[string]any, extractOpts func() tui.CommitOption) {
	log.Printf("message tmpl: %s", cfg.Template)

	// --- Group 1: commit message fields ---
	extractMsg = func() map[string]any {
		m := make(map[string]any, len(cfg.Items))
		for i := range cfg.Items {
			m[cfg.Items[i].Name] = cfg.Items[i].Value
		}

		return m
	}

	groups := []*huh.Group{tui.CommitMessageGroup(cfg.Items)}

	// --- Group 2: commit options (skipped when any flag was passed) ---
	opts := defaults
	if !defaults.AnyOptionSet() {
		groups = append(groups, tui.CommitOptionsGroup(&opts))
	}

	extractOpts = func() tui.CommitOption { return opts }

	return huh.NewForm(groups...), extractMsg, extractOpts
}
