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
func DefaultMessageConfig() (MessageConfig, error) {
	config := struct{ Message MessageConfig }{}
	if err := json.Unmarshal([]byte(defaultConfig), &config); err != nil {
		return MessageConfig{}, fmt.Errorf("parse default config: %w", err)
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
func FillOutForm(cfg MessageConfig, defaults FormOptions, authors []string) ([]byte, FormOptions, error) {
	form, extractMsg, extractOpts, tmplText := loadForm(cfg, defaults, authors)

	if err := form.Run(); err != nil {
		return nil, FormOptions{}, fmt.Errorf("failed to run the form: %w", err)
	}

	answers := extractMsg()
	opts := extractOpts()

	var buf bytes.Buffer
	if err := assembleMessage(&buf, tmplText, answers); err != nil {
		log.Printf("assemble failed, err=%v\n", err)

		return nil, FormOptions{}, fmt.Errorf("assemble message: %w", err)
	}

	return buf.Bytes(), opts, nil
}

// FormItemOption represents a single selectable option within a select form field.
type FormItemOption struct {
	Name string // value stored in the commit message template
	Desc string // label shown to the user in the TUI
}

// FormItem describes one field in the commit form as defined in the config file.
type FormItem struct {
	Name     string            // key used in the message template
	Desc     string            // prompt text shown to the user
	Form     string            // field type: "select", "input", or "multiline"
	Options  []*FormItemOption // options for "select" fields
	Required bool              // whether the field must be non-empty
}

// MessageConfig holds the ordered list of form fields and the Go template used to assemble the commit message.
type MessageConfig struct {
	Items    []*FormItem
	Template string
}

// FormOptions holds commit option values for Group 2 of the TUI.
// Used both as flag-derived defaults (input) and as user selections (output).
type FormOptions struct {
	All        bool
	Amend      bool
	NoVerify   bool
	Signoff    bool
	AllowEmpty bool
	Author     string // "Name <email>"
}

// anyOptionSet reports true if any commit-option flag was passed.
// When true, Group 2 of the TUI is skipped.
func (o FormOptions) anyOptionSet() bool {
	return o.All || o.Amend || o.NoVerify || o.Signoff || o.AllowEmpty || o.Author != ""
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

func loadForm(cfg MessageConfig,
	defaults FormOptions,
	authors []string) (*huh.Form, func() map[string]any, func() FormOptions, string) {
	log.Printf("message tmpl: %s", cfg.Template)

	// --- Group 1: commit message fields ---
	fields := make([]tui.MessageField, len(cfg.Items))
	for i, item := range cfg.Items {
		opts := make([]tui.MessageFieldOption, len(item.Options))
		for j, o := range item.Options {
			opts[j] = tui.MessageFieldOption{Name: o.Name, Desc: o.Desc}
		}

		fields[i] = tui.MessageField{
			Name:     item.Name,
			Desc:     item.Desc,
			Form:     item.Form,
			Options:  opts,
			Required: item.Required,
		}
	}

	extractMsg := func() map[string]any {
		m := make(map[string]any, len(fields))
		for i := range fields {
			m[fields[i].Name] = fields[i].Value
		}

		return m
	}

	groups := []*huh.Group{tui.CommitMessageGroup(fields)}

	// --- Group 2: commit options (skipped when any flag was passed) ---
	opts := defaults
	if !defaults.anyOptionSet() {
		groups = append(groups, tui.CommitOptionsGroup(
			authors,
			&opts.Author,
			&opts.All, &opts.Amend, &opts.NoVerify, &opts.Signoff, &opts.AllowEmpty,
		))
	}

	extractOpts := func() FormOptions { return opts }

	return huh.NewForm(groups...), extractMsg, extractOpts, cfg.Template
}
