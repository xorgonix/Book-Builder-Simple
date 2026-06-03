#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
output_file="$root_dir/allfiles.txt"

: > "$output_file"

find "$root_dir" \
  \( -type d \( -name ".git" -o -iname "*history*" \) -prune \) -o \
  \( -type f -name "*.go" ! -ipath "*history*" -print0 \) |
  sort -z |
  while IFS= read -r -d '' file; do
    {
      printf '===== %s =====\n' "$file"
      cat "$file"
      printf '\n\n'
    } >> "$output_file"
  done

printf 'Wrote %s\n' "$output_file"
