.PHONY: frontend

all: fmt check test

bump_deps:
	go get -u ./...
	cd frontend && pnpm up --latest --interactive

check: lint_golangci static
	make -C frontend check

lint_golangci:
	@golangci-lint run --timeout 3m

static:
	@staticcheck -go 1.22 ./...

deps: deps-go
	make -C frontend deps

frontend:
	make -C frontend build

deps-go:
	go install github.com/cosmtrek/air@v1.51.0
	go install github.com/nicksnyder/go-i18n/v2/goi18n@v2.4.0
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.26.0
	go install github.com/daixiang0/gci@v0.13.4
	go install mvdan.cc/gofumpt@v0.6.0
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.57.2
	go install honnef.co/go/tools/cmd/staticcheck@v0.4.6
	go install github.com/goreleaser/goreleaser@v1.25.1

test:
	go test ./...

fmt:
	gci write . --skip-generated -s standard -s default
	gofumpt -l -w .
	cd frontend && pnpm prettier src/ --write

watch-go:
	@air

serve-ts:
	make -C frontend serve

local-build: frontend
	go build -o bd

snapshot:
	goreleaser build --snapshot --clean

generate:
	sqlc generate