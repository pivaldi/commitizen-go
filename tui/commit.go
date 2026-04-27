package tui

import (
	"errors"
	"log"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// CommitItemOption is a single selectable option within a CommitItem select field.
type CommitItemOption struct {
	Name string // value stored in the commit message template
	Desc string // label shown to the user in the TUI
}

// CommitItem describes one field in the commit message form.
// Value is written by the form after the user submits.
type CommitItem struct {
	Name     string             // key used in the message template
	Desc     string             // prompt text shown to the user
	Form     string             // field type: "select", "input", or "multiline"
	Options  []CommitItemOption // options for "select" fields
	Required bool               // whether the field must be non-empty
	Value    string             // written by the form after submission
}

// CommitMessageConfig holds the ordered list of form fields and the Go template used to assemble the commit message.
type CommitMessageConfig struct {
	Items    []CommitItem
	Template string
}

// CommitOption holds commit option values for the commit options group of the TUI.
// Used both as flag-derived defaults (input) and as user selections (output).
type CommitOption struct {
	Authors    []string
	Author     string
	All        bool
	Amend      bool
	NoVerify   bool
	Signoff    bool
	AllowEmpty bool
}

// AnyOptionSet reports true if any commit-option flag was passed.
// When true, the CommitOptionsGroup of the TUI should be skipped.
func (o CommitOption) AnyOptionSet() bool {
	return o.All || o.Amend || o.NoVerify || o.Signoff || o.AllowEmpty || o.Author != ""
}

var descStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#888888")).
	PaddingLeft(1)

// CommitMessageGroup builds the first commit form group from the given items.
// Each item's Value is written by the form after the user submits.
func CommitMessageGroup(items []CommitItem) *huh.Group {
	requireValidator := func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("required")
		}

		return nil
	}

	msgFields := make([]huh.Field, 0, len(items))
	for i := range items {
		f := &items[i]
		switch f.Form {
		case "select":
			opts := make([]huh.Option[string], len(f.Options))
			for j, opt := range f.Options {
				name := titleCase(opt.Name)
				opts[j] = huh.NewOption(name+"\n"+descStyle.Render(opt.Desc), opt.Name)
			}
			sel := huh.NewSelect[string]().
				Title(f.Desc).
				Options(opts...).
				Value(&f.Value)
			if f.Required {
				sel = sel.Validate(requireValidator)
			}
			msgFields = append(msgFields, sel)
		case "input":
			inp := huh.NewInput().
				Title(titleCase(f.Name) + ":").
				Placeholder(f.Desc).
				Value(&f.Value)
			if f.Required {
				inp = inp.Validate(requireValidator)
			}
			msgFields = append(msgFields, inp)
		case "multiline":
			txt := huh.NewText().
				Lines(2).
				Title(strings.ToTitle(f.Name)).
				Placeholder(f.Desc).
				Value(&f.Value)
			if f.Required {
				txt = txt.Validate(requireValidator)
			}
			msgFields = append(msgFields, txt)
		default:
			log.Printf("unknown form type %q for field %q, skipping", f.Form, f.Name)
		}
	}

	return huh.NewGroup(msgFields...)
}

// CommitOptionsGroup builds the second commit form group (author + commit flags).
// opt is written directly by the form fields.
func CommitOptionsGroup(opt *CommitOption) *huh.Group {
	authorOpts := make([]huh.Option[string], 0, len(opt.Authors))
	for _, a := range opt.Authors {
		authorOpts = append(authorOpts, huh.NewOption(a, a))
	}
	if len(authorOpts) == 0 {
		authorOpts = []huh.Option[string]{huh.NewOption("(no authors found)", "")}
	}

	return huh.NewGroup(
		huh.NewSelect[string]().
			Title("Author:").
			Options(authorOpts...).
			Value(&opt.Author),
		huh.NewConfirm().Title("Stage all tracked modified/deleted files? (--all)").Value(&opt.All),
		huh.NewConfirm().Title("Amend last commit? (--amend)").Value(&opt.Amend),
		huh.NewConfirm().Title("Skip hooks? (--no-verify)").Value(&opt.NoVerify),
		huh.NewConfirm().Title("Add Signed-off-by trailer? (--signoff)").Value(&opt.Signoff),
		huh.NewConfirm().Title("Allow empty commit? (--allow-empty)").Value(&opt.AllowEmpty),
	)
}

func titleCase(s string) string {
	c := cases.Title(language.English, cases.NoLower)

	return c.String(s)
}
