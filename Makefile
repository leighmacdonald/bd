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
	fyne bundle --pkg ui    -o ./ui/embed_font.go ./ui/resources/JetBrainsMono/fonts/ttf/JetBrainsMono-Regular.ttf
	fyne bundle --pkg ui -a -o ./ui/embed_font.go ./ui/resources/JetBrainsMono/fonts/ttf/JetBrainsMono-Bold.ttf
	fyne bundle --pkg ui -a -o ./ui/embed_font.go ./ui/resources/JetBrainsMono/fonts/ttf/JetBrainsMono-Italic.ttf
	fyne bundle --pkg ui -a -o ./ui/embed_font.go ./ui/resources/JetBrainsMono/fonts/ttf/JetBrainsMono-BoldItalic.ttf
	fyne bundle --pkg ui    -o ./ui/embed_img.go ./ui/resources/default_avatar.jpg
	fyne bundle --pkg ui -a -o ./ui/embed_img.go ./ui/resources/Icon.png
	fyne bundle --pkg ui -a -o ./ui/embed_img.go ./ui/resources/tf2.png

lint: lint_deps
	# TODO remove `--disable unused` check once further along in development
	golangci-lint run --disable unused --timeout 3m --verbose
	go vet -tags ci ./...
	test -z $(goimports -e -d . | tee /dev/stderr)
	gocyclo -over 35 .
	golint -set_exit_status $(go list -tags ci ./...)
	#staticcheck -go 1.19 ./...

lint_deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.51.2
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	go install golang.org/x/lint/golint@latest
	#go install honnef.co/go/tools/cmd/staticcheck@latest

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
