package commit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"text/template"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// FillOutForm presents the commit TUI form.
// Group 1: commit message fields (type, scope, subject, body, footer).
// Group 2: commit options (author, all, amend, no-verify, signoff, allow-empty).
//
//	Group 2 is skipped when defaults.anyOptionSet() is true (flags were passed).
//
// Returns the assembled commit message bytes and the (possibly user-modified) options.
func FillOutForm(defaults FormOptions, authors []string) ([]byte, FormOptions, error) {
	form, extractMsg, extractOpts, tmplText, err := loadForm(defaults, authors)
	if err != nil {
		log.Printf("loadForm failed, err=%v\n", err)

		return nil, FormOptions{}, fmt.Errorf("load form: %w", err)
	}

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

func titleCase(s string) string {
	c := cases.Title(language.English, cases.NoLower)
	return c.String(s)
}

func loadForm(defaults FormOptions, authors []string) (*huh.Form, func() map[string]any, func() FormOptions, string, error) {
	config := struct{ Message MessageConfig }{}
	if err := json.Unmarshal([]byte(defaultConfig), &config); err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to unmarshal: %w", err)
	}

	msgConfig := config.Message
	log.Printf("default config message tmpl: %s", msgConfig.Template)

	sub := viper.Sub("message")
	if sub == nil {
		log.Print("no message in config file")
	} else {
		if err := sub.Unmarshal(&msgConfig, func(cfg *mapstructure.DecoderConfig) { cfg.ZeroFields = true }); err != nil {
			log.Printf("ill message in config file, err=%v", err)
		}
	}

	// --- Group 1: commit message fields ---
	values := make([]string, len(msgConfig.Items))
	msgFields := make([]huh.Field, 0, len(msgConfig.Items))
	requireValidator := func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("required")
		}

		return nil
	}

	var style = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		PaddingLeft(1)

	for i, item := range msgConfig.Items {
		switch item.Form {
		case "select":
			opts := make([]huh.Option[string], len(item.Options))
			for j, opt := range item.Options {
				name := titleCase(opt.Name)
				opts[j] = huh.NewOption(name+"\n"+style.Render(opt.Desc), opt.Name)
			}
			sel := huh.NewSelect[string]().
				Title(item.Desc).
				Options(opts...).
				Value(&values[i])
			if item.Required {
				sel = sel.Validate(requireValidator)
			}
			msgFields = append(msgFields, sel)
		case "input":
			inp := huh.NewInput().
				Title(titleCase(item.Name) + ":").
				Placeholder(item.Desc).
				Value(&values[i])
			if item.Required {
				inp = inp.Validate(requireValidator)
			}
			msgFields = append(msgFields, inp)
		case "multiline":
			txt := huh.NewText().
				Lines(2).
				Title(strings.ToTitle(item.Name)).
				Placeholder(item.Desc).
				Value(&values[i])
			if item.Required {
				txt = txt.Validate(requireValidator)
			}
			msgFields = append(msgFields, txt)
		default:
			log.Printf("unknown form type %q for item %q, skipping", item.Form, item.Name)
		}
	}

	items := msgConfig.Items
	extractMsg := func() map[string]any {
		m := make(map[string]any, len(items))
		for i, item := range items {
			m[item.Name] = values[i]
		}

		return m
	}

	groups := []*huh.Group{huh.NewGroup(msgFields...)}

	// --- Group 2: commit options (skipped when any flag was passed) ---
	opts := defaults
	if !defaults.anyOptionSet() {
		authorOpts := make([]huh.Option[string], len(authors))
		for i, a := range authors {
			authorOpts[i] = huh.NewOption(a, a)
		}
		if len(authorOpts) == 0 {
			authorOpts = []huh.Option[string]{huh.NewOption("(no authors found)", "")}
		}

		authorSel := huh.NewSelect[string]().
			Title("Author:").
			Options(authorOpts...).
			Value(&opts.Author)

		optFields := []huh.Field{
			authorSel,
			huh.NewConfirm().Title("Stage all tracked modified/deleted files? (--all)").Value(&opts.All),
			huh.NewConfirm().Title("Amend last commit? (--amend)").Value(&opts.Amend),
			huh.NewConfirm().Title("Skip hooks? (--no-verify)").Value(&opts.NoVerify),
			huh.NewConfirm().Title("Add Signed-off-by trailer? (--signoff)").Value(&opts.Signoff),
			huh.NewConfirm().Title("Allow empty commit? (--allow-empty)").Value(&opts.AllowEmpty),
		}
		groups = append(groups, huh.NewGroup(optFields...))
	}

	extractOpts := func() FormOptions { return opts }

	return huh.NewForm(groups...), extractMsg, extractOpts, msgConfig.Template, nil
}
