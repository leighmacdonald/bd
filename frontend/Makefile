build:
	pnpm run build

deps:
	pnpm install --frozen-lockfile

check-eslint:
	pnpm run eslint:check --ignore-pattern dist/

check-prettier:
	pnpm prettier src/ --check

check: check-eslint check-prettier

watch:
	pnpm watch

serve:
	pnpm serve

bump_deps:
	pnpm up --latest --interactive

fmt:
	pnpm prettier src/ --write

clean:
	rm -rf dist/
