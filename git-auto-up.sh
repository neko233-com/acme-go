#!/usr/bin/env sh
set -eu

commit_message="${1:-auto up}"

./test-auto.sh

git add .
if git diff --cached --quiet; then
	echo "No staged changes. Nothing to push."
	exit 0
fi

git commit -m "$commit_message"
git push

echo "Done."