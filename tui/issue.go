package tui

import (
	"errors"

	"github.com/charmbracelet/huh"
)

const (
	IssueActionNameStart = "issueStart"
	IssueActionNameList  = "issueList"
	IssueActionNameClose = "issueClose"
)

// IssueActionSelect presents the list of available issue actions.
func IssueActionSelect(action *string) *huh.Group {
	return huh.NewGroup(
		huh.NewSelect[string]().
			Title("Issue action:").
			Options(
				huh.NewOption("Start\n"+descStyle.Render(
					"Start working on an issue (create branch)"), IssueActionNameStart),
				huh.NewOption("List\n"+descStyle.Render("List open issues"), IssueActionNameList),
				huh.NewOption("Close\n"+descStyle.Render("Close an issue"), IssueActionNameClose),
			).
			Value(action),
	)
}

func IssueInput(issueID, title, branchType *string, allowedBranchTypes []string) *huh.Group {
	typeOpts := make([]huh.Option[string], 0, len(allowedBranchTypes))
	for _, allowed := range allowedBranchTypes {
		typeOpts = append(typeOpts, huh.NewOption(allowed, allowed))
	}

	group := huh.NewGroup(
		huh.NewInput().
			Title("Issue ID:").
			Placeholder("ABC-42").
			Validate(func(s string) error {
				if s == "" {
					return errors.New("required")
				}

				return nil
			}).
			Value(issueID),
		huh.NewInput().
			Title("Title:").
			Placeholder("Short description of the issue").
			Validate(func(s string) error {
				if s == "" {
					return errors.New("required")
				}

				return nil
			}).
			Value(title),
		huh.NewSelect[string]().
			Title("Type:").
			Options(typeOpts...).
			Value(branchType),
	)

	return group
}

func IssueConfirm(confirmTitle string, confirmed *bool) *huh.Group {
	return huh.NewGroup(
		huh.NewConfirm().
			Title(confirmTitle).
			Value(confirmed),
	)
}
