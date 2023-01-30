all: fmt lint binary_local

build_windows:
	fyne package -os windows -icon ./Icon.png

deps:
	go install fyne.io/fyne/v2/cmd/fyne@latest
	go install github.com/nicksnyder/go-i18n/v2/goi18n@latest
	go install github.com/goreleaser/goreleaser@latest

lint:
	golangci-lint run

fmt:
	go mod tidy
	gofmt -s -w .

extract:
	goi18n extract -outdir translations/ -format yaml

release_local:
	goreleaser release --snapshot --rm-dist

binary_local:
	goreleaser build --single-target --snapshot --rm-dist
