#!/bin/sh

golangci_lint_version=2.15.0

step() {
	printf '\033[1;34m▸ %s\033[0m\n' "$1"
}

fail() {
	printf '\033[1;31m✗ %s\033[0m\n' "$1" >&2
	printf '  (git hooks from .githooks; see CONTRIBUTING.md)\n' >&2
	exit 1
}

find_golangci_lint() {
	lint=$(command -v golangci-lint 2>/dev/null || true)
	if [ -z "$lint" ] && [ -x "$(go env GOPATH)/bin/golangci-lint" ]; then
		lint="$(go env GOPATH)/bin/golangci-lint"
	fi
	if [ -z "$lint" ]; then
		printf '\033[1;31m✗ golangci-lint is required and was not found.\033[0m\n' >&2
		printf '  Install it with:\n    go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v%s\n' "$golangci_lint_version" >&2
		return 1
	fi
	installed=$("$lint" version --short 2>/dev/null || true)
	if [ "$installed" != "$golangci_lint_version" ]; then
		printf '\033[1;33m! golangci-lint %s found; CI uses %s.\033[0m\n' "${installed:-unknown}" "$golangci_lint_version" >&2
	fi
	echo "$lint"
}
