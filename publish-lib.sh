#!/usr/bin/env sh
set -eu

version="${1:-}"

if ! command -v git >/dev/null 2>&1; then
	echo "git is required." >&2
	exit 1
fi
if ! command -v go >/dev/null 2>&1; then
	echo "go is required." >&2
	exit 1
fi

if [ -z "$version" ]; then
	latest="$(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' | sort -V | tail -n 1 || true)"
	if [ -z "$latest" ]; then
		version="v0.0.1"
	else
		base="${latest#v}"
		major="${base%%.*}"
		rest="${base#*.}"
		minor="${rest%%.*}"
		patch="${rest##*.}"
		patch=$((patch + 1))
		version="v${major}.${minor}.${patch}"
	fi
fi

case "$version" in
	v*) ;;
	*) version="v$version" ;;
esac

echo "Publishing library version $version"

if ! git diff --quiet; then
	echo "Working tree has unstaged changes. Commit or stash them before publishing." >&2
	exit 1
fi
if ! git diff --cached --quiet; then
	echo "Working tree has staged changes. Commit or unstage them before publishing." >&2
	exit 1
fi
if git rev-parse "$version" >/dev/null 2>&1; then
	echo "Tag $version already exists." >&2
	exit 1
fi

./test-auto.sh --skip-integration
go test ./pkg/acmego

git tag -a "$version" -m "publish library $version"
git push origin "$version"

echo "Published $version to GitHub. Consumers can use: go get github.com/neko233-com/acme-go/pkg/acmego@$version"
