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

// MessageField describes one field in the commit message form.
// Value is written by the form after the user submits.
type MessageField struct {
	Name     string
	Desc     string
	Form     string // "select", "input", or "multiline"
	Options  []MessageFieldOption
	Required bool
	Value    string
}

// MessageFieldOption is a single selectable option within a select field.
type MessageFieldOption struct {
	Name string
	Desc string
}

var descStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#888888")).
	PaddingLeft(1)

// CommitMessageGroup builds the first commit form group from the given fields.
// Each field's Value is written by the form after the user submits.
func CommitMessageGroup(fields []MessageField) *huh.Group {
	requireValidator := func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("required")
		}

		return nil
	}

	msgFields := make([]huh.Field, 0, len(fields))
	for i := range fields {
		f := &fields[i]
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
// All pointer arguments are written directly by the form fields.
func CommitOptionsGroup(
	authors []string,
	author *string,
	all, amend, noVerify, signoff, allowEmpty *bool,
) *huh.Group {
	authorOpts := make([]huh.Option[string], 0, len(authors))
	for _, a := range authors {
		authorOpts = append(authorOpts, huh.NewOption(a, a))
	}
	if len(authorOpts) == 0 {
		authorOpts = []huh.Option[string]{huh.NewOption("(no authors found)", "")}
	}

	return huh.NewGroup(
		huh.NewSelect[string]().
			Title("Author:").
			Options(authorOpts...).
			Value(author),
		huh.NewConfirm().Title("Stage all tracked modified/deleted files? (--all)").Value(all),
		huh.NewConfirm().Title("Amend last commit? (--amend)").Value(amend),
		huh.NewConfirm().Title("Skip hooks? (--no-verify)").Value(noVerify),
		huh.NewConfirm().Title("Add Signed-off-by trailer? (--signoff)").Value(signoff),
		huh.NewConfirm().Title("Allow empty commit? (--allow-empty)").Value(allowEmpty),
	)
}

func titleCase(s string) string {
	c := cases.Title(language.English, cases.NoLower)

	return c.String(s)
}
