.PHONY: frontend

all: fmt check test

deps_release:
	go install github.com/nicksnyder/go-i18n/v2/goi18n@latest

bump_deps:
	go get -u ./...
	cd frontend && yarn upgrade-interactive --latest

check: lint_golangci static lint_ts

lint_golangci:
	@golangci-lint run --timeout 3m

lint_ts:
	cd frontend && yarn run eslint:check && yarn prettier src/ --check

frontend:
	cd frontend && yarn run build

static:
	@staticcheck -go 1.20 ./...

build_deps:
	go install github.com/daixiang0/gci@latest
	go install mvdan.cc/gofumpt@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.54.2
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/goreleaser/goreleaser@latest

test:
	go test ./...

fmt:
	gci write . --skip-generated -s standard -s default
	gofumpt -l -w .
	cd frontend && yarn prettier src/ --write

watch:
	cd frontend && yarn run watch

deps:
	cd frontend && yarn install
	go mod download

snapshot:
	goreleaser build --snapshot --clean
