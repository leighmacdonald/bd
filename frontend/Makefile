build:
	pnpm run build

deps:
	pnpm install --frozen-lockfile

check-eslint:
	pnpm run eslint:check

check-prettier:
	pnpm prettier src/ --check

check: check-eslint check-prettier

watch:
	pnpm watch

serve:
	pnpm serve
