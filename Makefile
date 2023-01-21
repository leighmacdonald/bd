all: deps build_windows

deps:
	go install fyne.io/fyne/v2/cmd/fyne@latest

build_windows:
	fyne package -os windows -icon Icon.png
