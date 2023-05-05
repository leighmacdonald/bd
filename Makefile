all: fmt check
	go build

deps:
	go install fyne.io/fyne/v2/cmd/fyne@latest
	go install github.com/nicksnyder/go-i18n/v2/goi18n@latest
	# go install github.com/goreleaser/goreleaser@latest
	# for pro install from: https://github.com/goreleaser/goreleaser-pro/releases

extract: fonts tr_extract

fmt:
	gofmt -s -w .

bump_deps:
	go get -u ./...
	cd widget && yarn upgrade-interactive --latest

fonts:
	fyne bundle --pkg ui    -o ./internal/ui/embed_img.go ./internal/ui/resources/default_avatar.jpg
	fyne bundle --pkg ui -a -o ./internal/ui/embed_img.go ./internal/ui/resources/Icon.png
	fyne bundle --pkg ui -a -o ./internal/ui/embed_img.go ./internal/ui/resources/tf2.png

check: lint_golangci lint_vet lint_imports lint_cyclo lint_golint static

lint_golangci:
	@golangci-lint run --timeout 3m

lint_vet:
	@go vet -tags ci ./...

lint_imports:
	@test -z $(goimports -e -d . | tee /dev/stderr)

lint_cyclo:
	@gocyclo -over 40 .

lint_golint:
	@golint -set_exit_status $(go list -tags ci ./...)

static:
	@staticcheck -go 1.20 ./...

check_deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.52.2
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
	goreleaser build --snapshot --clean --id linux

test:
	go test ./...

tr_extract:
	goi18n extract -outdir internal/tr/ -format yaml

tf_new_lang:
	goi18n merge internal/tr/active.en.toml translate.es.toml

tr_gen_translate:
	goi18n merge -format yaml -outdir internal/tr/ internal/tr/active.*.yaml

tr_merge:
	goi18n merge -format yaml -outdir internal/tr/ internal/tr/active.*.yaml internal/tr/translate.*.yaml

update:
	go get -u ./...
