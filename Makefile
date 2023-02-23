all: fmt lint snapshot_windows

deps:
	go install fyne.io/fyne/v2/cmd/fyne@latest
	go install github.com/nicksnyder/go-i18n/v2/goi18n@latest
	# go install github.com/goreleaser/goreleaser@latest
	# for pro install frpm: https://github.com/goreleaser/goreleaser-pro/releases

extract: fonts translations

fmt:
	gofmt -s -w .

fonts:
	fyne bundle --pkg ui    -o ./ui/embed_font.go ./ui/resources/JetBrainsMono/fonts/ttf/JetBrainsMono-Regular.ttf
	fyne bundle --pkg ui -a -o ./ui/embed_font.go ./ui/resources/JetBrainsMono/fonts/ttf/JetBrainsMono-Bold.ttf
	fyne bundle --pkg ui -a -o ./ui/embed_font.go ./ui/resources/JetBrainsMono/fonts/ttf/JetBrainsMono-Italic.ttf
	fyne bundle --pkg ui -a -o ./ui/embed_font.go ./ui/resources/JetBrainsMono/fonts/ttf/JetBrainsMono-BoldItalic.ttf
	fyne bundle --pkg ui    -o ./ui/embed_img.go ./ui/resources/default_avatar.jpg
	fyne bundle --pkg ui -a -o ./ui/embed_img.go ./ui/resources/Icon.png
	fyne bundle --pkg ui -a -o ./ui/embed_img.go ./ui/resources/tf2.png

lint: lint_deps
	golangci-lint run --timeout 3m --verbose
	go vet -tags ci ./...
	test -z $(goimports -e -d . | tee /dev/stderr)
	gocyclo -over 35 .
	golint -set_exit_status $(go list -tags ci ./...)
	staticcheck -go 1.19 ./...

lint_deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.51.2
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	go install golang.org/x/lint/golint@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest

build: build_linux build_windows

build_windows:
	fyne-cross windows -pull -arch=amd64 -name=bd.exe

build_linux:
	fyne-cross linux -pull -arch=amd64 -name=bd

release_local:
	goreleaser release --nightly --clean --snapshot

snapshot_windows:
	goreleaser build --single-target --snapshot --clean --id windows

snapshot_linux:
	goreleaser build --snapshot --rm-dist --id unix

test:
	go test .\...

translations:
	goi18n extract -outdir translations/ -format yaml

update:
	go get -u ./...
