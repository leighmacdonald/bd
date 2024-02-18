.PHONY: frontend

all: fmt check test

deps_release:
	go install github.com/nicksnyder/go-i18n/v2/goi18n@latest

bump_deps:
	go get -u ./...
	cd frontend && pnpm up --latest --interactive

check: lint_golangci static lint_ts

lint_golangci:
	@golangci-lint run --timeout 3m

lint_ts:
	cd frontend && pnpm run eslint:check && pnpm prettier src/ --check

frontend:
	cd frontend && pnpm run build

static:
	@staticcheck -go 1.22 ./...

build_deps:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/daixiang0/gci@latest
	go install mvdan.cc/gofumpt@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.56.1
	go install honnef.co/go/tools/cmd/staticcheck@v0.4.6
	go install github.com/goreleaser/goreleaser@latest

test:
	go test ./...

fmt:
	gci write . --skip-generated -s standard -s default
	gofumpt -l -w .
	cd frontend && pnpm prettier src/ --write

watch-go:
	@go install github.com/cosmtrek/air@latest
	@air

watch-ts:
	cd frontend && pnpm watch

deps:
	cd frontend && pnpm install --frozen-lockfile
	go mod download

snapshot:
	goreleaser build --snapshot --clean

generate:
	sqlc generate