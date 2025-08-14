#!/usr/bin/env bash
set -euo pipefail

# Check Go test filenames for banned terms and excessive length.
# Usage: scripts/check-test-filenames.sh [root-dir]

root=${1:-.}

# Configure rules
maxlen=${MAX_TEST_FILENAME_LEN:-44}
regex_banned='(^|_)(coverage|comprehensive|extra|additional|more|final|last|exact|fix)($|_)'

err=0
while IFS= read -r -d '' f; do
  base=$(basename "$f")
  # Check length (basename only)
  if (( ${#base} > maxlen )); then
    echo "ERROR: test filename too long (${#base} > ${maxlen}): $f" >&2
    err=1
  fi
  # Check banned terms
  if [[ "$base" =~ $regex_banned ]]; then
    echo "ERROR: test filename contains banned adjective: $f" >&2
    err=1
  fi
  # Check spaces or uppercase
  if [[ "$base" =~ [A-Z] ]] || [[ "$base" =~ [[:space:]] ]]; then
    echo "ERROR: test filename must be lowercase with underscores only: $f" >&2
    err=1
  fi
done < <(find "$root" -type f -name '*_test.go' -print0)

if (( err != 0 )); then
  echo "\nTest filename check failed." >&2
  exit 1
fi

echo "All test filenames look good."
exit 0

