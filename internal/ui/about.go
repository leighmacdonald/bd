package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/internal/model"
	"github.com/leighmacdonald/bd/internal/translations"
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
		labelBuiltBy: widget.NewRichTextWithText(translations.One(translations.LabelAboutBuiltBy)),
		labelDate:    widget.NewRichTextWithText(translations.One(translations.LabelAboutBuildDate)),
		labelVersion: widget.NewRichTextWithText(translations.One(translations.LabelAboutVersion)),
		labelCommit:  widget.NewRichTextWithText(translations.One(translations.LabelAboutCommit)),
		labelGo: widget.NewRichText(
			&widget.TextSegment{Text: "Go ", Style: widget.RichTextStyleInline},
			&widget.TextSegment{Text: runtime.Version(), Style: widget.RichTextStyleStrong},
		),
	}
	about.Dialog = dialog.NewCustom(
		translations.One(translations.LabelAbout),
		translations.One(translations.LabelClose),
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
