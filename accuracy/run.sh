#!/usr/bin/env bash
# Accuracy gate: renders every fixture with proteus and with headless Chromium
# (fonts pinned by accuracy/fonts.conf) and fails when a page differs by more
# than MAX_MISMATCH percent of pixels (or MAX_NEAR percent once ink shifted by
# up to 1px is forgiven), or when the text differs (textdiff: fewer than
# MIN_MATCH percent of words found, or fewer than MIN_WITHIN percent within TOL
# points of Chromium's position). Needs node + playwright and poppler.
set -euo pipefail
cd "$(dirname "$0")/.."
MAX_MISMATCH="${MAX_MISMATCH:-3}"   # raw ink mismatch per page, includes anti-aliasing noise
MAX_NEAR="${MAX_NEAR:-0.5}"         # ink with nothing within 1px on the other render
MIN_MATCH="${MIN_MATCH:-99}"
MIN_WITHIN="${MIN_WITHIN:-98}"
TOL="${TOL:-2}"
OUT="${OUT:-accuracy/out}"
mkdir -p "$OUT"
export FONTCONFIG_FILE="$PWD/accuracy/fonts.conf"

go build -o "$OUT/proteus" ./cmd/proteus
go build -o "$OUT/pixdiff" ./internal/tools/pixdiff
go build -o "$OUT/mdhtml" ./internal/tools/mdhtml
go build -o "$OUT/textdiff" ./internal/tools/textdiff

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
  "$OUT/pixdiff" -max "$MAX_MISMATCH" -near 1 -max-near "$MAX_NEAR" -diff "$OUT/$name.diff.png" \
    "$OUT/$name.chromium.pdf" "$OUT/$name.proteus.pdf" || status=1
  "$OUT/textdiff" -tol "$TOL" -min-match "$MIN_MATCH" -min-within "$MIN_WITHIN" \
    "$OUT/$name.chromium.pdf" "$OUT/$name.proteus.pdf" || status=1
done
exit $status
