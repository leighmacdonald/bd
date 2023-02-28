package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/translations"
	"net/url"
	"runtime"
)

type aboutDialog struct {
	dialog       dialog.Dialog
	labelBuiltBy *widget.RichText
	labelDate    *widget.RichText
	labelVersion *widget.RichText
	labelCommit  *widget.RichText
	labelGo      *widget.RichText
}

func (aboutDialog *aboutDialog) SetBuildInfo(version string, commit string, date string, builtBy string) {
	if len(aboutDialog.labelVersion.Segments) == 1 {
		aboutDialog.labelVersion.Segments = append(aboutDialog.labelVersion.Segments, &widget.TextSegment{
			Style: widget.RichTextStyleStrong,
			Text:  version,
		})
		aboutDialog.labelVersion.Refresh()
	}
	if len(aboutDialog.labelCommit.Segments) == 1 {
		aboutDialog.labelCommit.Segments = append(aboutDialog.labelCommit.Segments, &widget.TextSegment{
			Style: widget.RichTextStyleStrong,
			Text:  commit,
		})
		aboutDialog.labelCommit.Refresh()
	}
	if len(aboutDialog.labelDate.Segments) == 1 {
		aboutDialog.labelDate.Segments = append(aboutDialog.labelDate.Segments, &widget.TextSegment{
			Style: widget.RichTextStyleStrong,
			Text:  date,
		})
		aboutDialog.labelDate.Refresh()
	}
	if len(aboutDialog.labelBuiltBy.Segments) == 1 {
		aboutDialog.labelBuiltBy.Segments = append(aboutDialog.labelBuiltBy.Segments, &widget.TextSegment{
			Style: widget.RichTextStyleStrong,
			Text:  builtBy,
		})
		aboutDialog.labelBuiltBy.Refresh()
	}
}

func newAboutDialog(parent fyne.Window) *aboutDialog {
	u, _ := url.Parse(urlHome)
	about := aboutDialog{
		labelBuiltBy: widget.NewRichTextWithText("Built By: "),
		labelDate:    widget.NewRichTextWithText("Build Date: "),
		labelVersion: widget.NewRichTextWithText("Version: "),
		labelCommit:  widget.NewRichTextWithText("Commit: "),
		labelGo: widget.NewRichText(
			&widget.TextSegment{Text: "Go ", Style: widget.RichTextStyleInline},
			&widget.TextSegment{Text: runtime.Version(), Style: widget.RichTextStyleStrong},
		),
	}
	about.dialog = dialog.NewCustom(
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
	return &about
}
