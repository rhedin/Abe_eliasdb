#!/usr/bin/env bash
set -euo pipefail

# Usage: ./strip-understand-logs.sh <directory>
DIR="${1:-.}"

find "$DIR" -name '*.go' -type f | while read -r file; do
    awk '
    BEGIN { in_block = 0 }

    # Start of multi-line block
    /<understand>/ { in_block = 1; next }

    # End of multi-line block
    /<\/understand>/ { in_block = 0; next }

    # Self-closing single-line tag: remove the entire line if it contains /*<understand/>*/
    /<understand\/>/ { next }

    # Normal line: print only if not inside a block
    {
        if (!in_block) print
    }
    ' "$file" > "$file.tmp" && mv "$file.tmp" "$file"
done
