all: fmt lint snapshot_windows

deps:
	go install fyne.io/fyne/v2/cmd/fyne@latest
	go install github.com/nicksnyder/go-i18n/v2/goi18n@latest
	go install github.com/goreleaser/goreleaser@latest
	go mod tidy

extract: fonts translations

fmt:
	gofmt -s -w .

fonts:
	fyne bundle --pkg ui    -o .\ui\font.go .\ui\resources\JetBrainsMono\fonts\ttf\JetBrainsMono-Regular.ttf
	fyne bundle --pkg ui -a -o .\ui\font.go .\ui\resources\JetBrainsMono\fonts\ttf\JetBrainsMono-Bold.ttf
	fyne bundle --pkg ui -a -o .\ui\font.go .\ui\resources\JetBrainsMono\fonts\ttf\JetBrainsMono-Italic.ttf
	fyne bundle --pkg ui -a -o .\ui\font.go .\ui\resources\JetBrainsMono\fonts\ttf\JetBrainsMono-BoldItalic.ttf
	fyne bundle --pkg ui    -o .\ui\img.go .\ui\resources\default_avatar.jpg
	fyne bundle --pkg ui -a -o .\ui\img.go .\ui\resources\Icon.png
	fyne bundle --pkg ui -a -o .\ui\img.go .\ui\resources\tf2_logo.svg

lint:
	# TODO remove `--disable unused` check once further along in development
	golangci-lint run --fix --disable unused

release_local:
	goreleaser release --snapshot --rm-dist

snapshot_windows:
	goreleaser build --single-target --snapshot --rm-dist --id windows

snapshot_linux:
	goreleaser build --snapshot --rm-dist --id unix

translations:
	goi18n extract -outdir translations/ -format yaml

update:
	go get -u ./...
