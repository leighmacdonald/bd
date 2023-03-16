package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/leighmacdonald/bd/internal/model"
	"image/color"
)

type contextMenuRichText struct {
	*widget.Button
	menu *fyne.Menu
}

func (b *contextMenuRichText) Tapped(e *fyne.PointEvent) {
	widget.ShowPopUpMenuAtPosition(b.menu, fyne.CurrentApp().Driver().CanvasForObject(b), e.AbsolutePosition)
}

func newContextMenuRichText(menu *fyne.Menu) *contextMenuRichText {
	return &contextMenuRichText{
		Button: widget.NewButtonWithIcon("", theme.AccountIcon(), func() {

		}),
		menu: menu,
	}
}

type contextMenuIcon struct {
	*widget.Icon
	menu *fyne.Menu
}

func (b *contextMenuIcon) Tapped(e *fyne.PointEvent) {
	widget.ShowPopUpMenuAtPosition(b.menu, fyne.CurrentApp().Driver().CanvasForObject(b), e.AbsolutePosition)
}

func newContextMenuIcon() *contextMenuIcon {
	return &contextMenuIcon{
		Icon: widget.NewIcon(theme.SettingsIcon()),
		menu: fyne.NewMenu(""),
	}
}

//	type clickableIcon struct {
//		*widget.Icon
//		onClicked func()
//	}
//
//	func (b *clickableIcon) Tapped(e *fyne.PointEvent) {
//		b.onClicked()
//	}
//
//	func newClickableIcon(icon fyne.Resource, clickHandler func()) *clickableIcon {
//		return &clickableIcon{
//			Icon:      widget.NewIcon(icon),
//			onClicked: clickHandler,
//		}
//	}
const (
	rowHeight  = 32
	sizeNumber = 64
	sizeName   = 300
	sizeIcon   = 64
)

type tableRowLayout struct {
}

func (l *tableRowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	w, h := float32(0), float32(0)
	for _, o := range objects {
		childSize := o.MinSize()
		w += childSize.Width
		h += childSize.Height
	}
	return fyne.NewSize(w, h)
}

type playerRow struct {
	widget.DisableableWidget

	Icon   fyne.Resource
	Player *model.Player

	hovered bool
	focused bool

	background fyne.CanvasObject

	tapAnim *fyne.Animation

	OnTapped func()
}

func newPlayerRow(icon fyne.Resource, player *model.Player) *playerRow {
	pr := &playerRow{
		Player: player,
		Icon:   icon,
	}
	pr.ExtendBaseWidget(pr)
	return pr
}

var colorRed = color.NRGBA{R: 255, G: 0, B: 0, A: 0xff}

func (r *playerRow) Tapped(e *fyne.PointEvent) {
	if r.OnTapped == nil {
		return
	}
	r.OnTapped()
}

func (r *playerRow) CreateRenderer() fyne.WidgetRenderer {
	r.ExtendBaseWidget(r)
	nameLabelSeg := &widget.TextSegment{Text: r.Player.Name, Style: widget.RichTextStyleStrong}
	nameLabelSeg.Style.Alignment = fyne.TextAlignLeading
	nameLabelText := widget.NewRichText(nameLabelSeg)

	KillsLabelSeg := &widget.TextSegment{Text: r.Player.Name, Style: widget.RichTextStyleStrong}
	KillsLabelSeg.Style.Alignment = fyne.TextAlignLeading
	killsLabelText := widget.NewRichText(KillsLabelSeg)

	objects := []fyne.CanvasObject{
		nameLabelText,
		killsLabelText,
		r.background,
	}
	renderer := rowRenderer{
		icon:           nil,
		nameLabel:      canvas.NewText(r.Player.Name, colorRed),
		killsLabel:     nil,
		deathsLabel:    nil,
		allTimeKDLabel: nil,
		background:     nil,
		layout:         layout.NewHBoxLayout(),
		objects:        objects,
	}

	return &renderer
}

type rowRenderer struct {
	icon           *canvas.Image
	nameLabel      *canvas.Text
	killsLabel     *canvas.Text
	deathsLabel    *canvas.Text
	allTimeKDLabel *canvas.Text
	matchLabel     *canvas.Text

	background *canvas.Rectangle

	objects []fyne.CanvasObject
	layout  fyne.Layout
}

func (r *rowRenderer) Layout(size fyne.Size) {
	//r.background.Resize()
}

func (r *rowRenderer) MinSize() fyne.Size {
	return fyne.NewSize(50, 50)
}

func (r *rowRenderer) Refresh() {

}
func (r *rowRenderer) BackgroundColor() color.Color {
	return color.Transparent
}

func (r *rowRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.background, r.icon, r.nameLabel, r.killsLabel, r.deathsLabel, r.allTimeKDLabel, r.matchLabel}
}

func (r *rowRenderer) Destroy() {

}
