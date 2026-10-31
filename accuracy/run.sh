#!/usr/bin/env bash
# Accuracy gate: renders every fixture with proteus and with headless Chromium
# (fonts pinned by accuracy/fonts.conf) and fails when a page differs by more
# than MAX_MISMATCH percent of pixels. Needs node + playwright and poppler.
set -euo pipefail
cd "$(dirname "$0")/.."
MAX_MISMATCH="${MAX_MISMATCH:-3}"
OUT="${OUT:-accuracy/out}"
mkdir -p "$OUT"
export FONTCONFIG_FILE="$PWD/accuracy/fonts.conf"

go build -o "$OUT/proteus" ./cmd/proteus
go build -o "$OUT/pixdiff" ./internal/tools/pixdiff
go build -o "$OUT/mdhtml" ./internal/tools/mdhtml

args=()
for f in testdata/*.html testdata/*.md; do
  name="$(basename "${f%.*}")"
  src="$f"
  if [[ "$f" == *.md ]]; then
    src="$OUT/$name.md.html"
    "$OUT/mdhtml" "$f" "$src"
  fi
  "$OUT/proteus" -q "$f" "$OUT/$name.proteus.pdf"
  args+=("$src" "$OUT/$name.chromium.pdf")
done
node accuracy/chromium.cjs "${args[@]}"

status=0
for f in testdata/*.html testdata/*.md; do
  name="$(basename "${f%.*}")"
  echo "== $name"
  "$OUT/pixdiff" -max "$MAX_MISMATCH" -diff "$OUT/$name.diff.png" \
    "$OUT/$name.chromium.pdf" "$OUT/$name.proteus.pdf" || status=1
done
exit $status
