package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/bd/internal/tr"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"net/url"
	"runtime"
)

type aboutDialog struct {
	dialog.Dialog
	labelBuiltBy *widget.RichText
	labelDate    *widget.RichText
	labelVersion *widget.RichText
	labelCommit  *widget.RichText
	labelGo      *widget.RichText
}

func newAboutDialog(parent fyne.Window, version model.Version) *aboutDialog {
	u, _ := url.Parse(urlHome)
	about := aboutDialog{
		labelBuiltBy: widget.NewRichTextWithText(tr.Localizer.MustLocalize(
			&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "about_label_built_by", Other: "Built By "}})),
		labelDate: widget.NewRichTextWithText(tr.Localizer.MustLocalize(
			&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "about_label_build_date", Other: "Build Date "}})),
		labelVersion: widget.NewRichTextWithText(tr.Localizer.MustLocalize(
			&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "about_label_version", Other: "Version "}})),
		labelCommit: widget.NewRichTextWithText(tr.Localizer.MustLocalize(
			&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "about_label_commit", Other: "Commit "}})),
		labelGo: widget.NewRichText(
			&widget.TextSegment{Text: "Go ", Style: widget.RichTextStyleInline},
			&widget.TextSegment{Text: runtime.Version(), Style: widget.RichTextStyleStrong},
		),
	}

	about.Dialog = dialog.NewCustom(
		tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "about_title", Other: "About"}}),
		tr.Localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "about_button_close", Other: "Close"}}),
		container.NewVBox(
			about.labelVersion,
			about.labelCommit,
			about.labelDate,
			about.labelBuiltBy,
			about.labelGo,
			widget.NewHyperlink(urlHome, u),
		),
		parent,
	)
	about.labelVersion.Segments = append(about.labelVersion.Segments, &widget.TextSegment{
		Style: widget.RichTextStyleStrong,
		Text:  version.Version,
	})

	about.labelCommit.Segments = append(about.labelCommit.Segments, &widget.TextSegment{
		Style: widget.RichTextStyleStrong,
		Text:  version.Commit,
	})

	about.labelDate.Segments = append(about.labelDate.Segments, &widget.TextSegment{
		Style: widget.RichTextStyleStrong,
		Text:  version.Date,
	})

	about.labelBuiltBy.Segments = append(about.labelBuiltBy.Segments, &widget.TextSegment{
		Style: widget.RichTextStyleStrong,
		Text:  version.BuiltBy,
	})
	about.Refresh()
	return &about
}
