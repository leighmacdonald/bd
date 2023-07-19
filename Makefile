all: fmt check
	go build

deps:
	go install github.com/nicksnyder/go-i18n/v2/goi18n@latest
	# go install github.com/goreleaser/goreleaser@latest
	# for pro install from: https://github.com/goreleaser/goreleaser-pro/releases

extract: tr_extract

bump_deps:
	go get -u ./...
	cd widget && yarn upgrade-interactive --latest

check: lint_golangci static

lint_golangci:
	@golangci-lint run --timeout 3m

static:
	@staticcheck -go 1.20 ./...

check_deps:
	go install github.com/daixiang0/gci@latest
	go install mvdan.cc/gofumpt@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.53.3
	go install honnef.co/go/tools/cmd/staticcheck@latest

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
	cd widget && yarn upgrade-interactive --latest

fmt:
	gci write . --skip-generated -s standard -s default
	gofumpt -l -w .

watch:
	cd widget && yarn run watch
