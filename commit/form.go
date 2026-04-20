package commit

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"text/template"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// FillOutForm presents the commit form to the user and returns the assembled commit message bytes.
func FillOutForm() ([]byte, error) {
	form, extract, tmplText, err := loadForm()
	if err != nil {
		log.Printf("loadForm failed, err=%v\n", err)
		return nil, err
	}

	if err := form.Run(); err != nil {
		return nil, err
	}

	answers := extract()

	var buf bytes.Buffer
	if err := assembleMessage(&buf, tmplText, answers); err != nil {
		log.Printf("assemble failed, err=%v\n", err)
		return nil, err
	}

	return buf.Bytes(), nil
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

// assembleMessage trims whitespace from all string answers, then executes tmplText writing the result to buf.
func assembleMessage(buf *bytes.Buffer, tmplText string, answers map[string]any) error {
	tmpl, err := template.New("").Parse(tmplText)
	if err != nil {
		return err
	}
	for k, v := range answers {
		if s, ok := v.(string); ok {
			answers[k] = strings.TrimSpace(s)
		}
	}
	return tmpl.Execute(buf, answers)
}

func titleCase(s string) string {
	c := cases.Title(language.English, cases.NoLower)
	return c.String(s)
}

// loadForm builds a huh.Form from the merged default and user config. It returns the form, an extractor
// that produces the answers map after form.Run(), and the commit message template string.
func loadForm() (*huh.Form, func() map[string]any, string, error) {
	config := struct{ Message MessageConfig }{}
	if err := json.Unmarshal([]byte(defaultConfig), &config); err != nil {
		return nil, nil, "", err
	}

	msgConfig := config.Message
	log.Printf("default config message tmpl: %s", msgConfig.Template)

	sub := viper.Sub("message")
	if sub == nil {
		log.Printf("no message in config file")
	} else {
		if err := sub.Unmarshal(&msgConfig, func(cfg *mapstructure.DecoderConfig) { cfg.ZeroFields = true }); err != nil {
			log.Printf("ill message in config file, err=%v", err)
		}
	}

	values := make([]string, len(msgConfig.Items))
	fields := make([]huh.Field, 0, len(msgConfig.Items))
	requireValidator := func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("required")
		}
		return nil
	}

	var style = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		// Background(lipgloss.Color("#3F3F3F")).
		PaddingLeft(1)
	// Width(80)

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
			fields = append(fields, sel)
		case "input":
			inp := huh.NewInput().
				Title(titleCase(item.Name) + ":").
				Placeholder(item.Desc).
				Value(&values[i])
			if item.Required {
				inp = inp.Validate(requireValidator)
			}
			fields = append(fields, inp)
		case "multiline":
			txt := huh.NewText().
				Lines(2).
				Title(strings.ToTitle(item.Name)).
				Placeholder(item.Desc).
				Value(&values[i])
			if item.Required {
				txt = txt.Validate(requireValidator)
			}
			fields = append(fields, txt)
		default:
			log.Printf("unknown form type %q for item %q, skipping", item.Form, item.Name)
		}
	}

	items := msgConfig.Items
	extract := func() map[string]any {
		m := make(map[string]any, len(items))
		for i, item := range items {
			m[item.Name] = values[i]
		}
		return m
	}

	return huh.NewForm(huh.NewGroup(fields...)), extract, msgConfig.Template, nil
}
