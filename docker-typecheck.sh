#!/bin/bash
#
# Type-checks golden files against the correct zod version inside Docker.
#
# Golden files must contain these metadata comments to be included:
#   // @typecheck             — present by default; files without it are skipped
#   // @zod-version: v3|v4   — (optional) restrict to one zod major; omit for both
#
# Usage:
#   ./typecheck/docker-typecheck.sh

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "========================================"
echo "Golden File Type-Check (Docker)"
echo "========================================"
echo ""

docker run --rm \
    -v "${PROJECT_DIR}/testdata:/golden:ro" \
    node:22-alpine \
    sh -c '
set -e

mkdir -p /test/zod3 /test/zod4

zod3_count=0
zod4_count=0

for file in $(find /golden -name "*.golden" -type f); do
    # Only process files with @typecheck metadata
    head -5 "$file" | grep -q "^// @typecheck" || continue

    # Extract zod version from metadata (empty = both)
    version=$(sed -n "s|^// @zod-version: ||p" "$file" | head -1)

    # Build unique .ts filename from relative path
    relpath="${file#/golden/}"
    ts_name="$(echo "$relpath" | sed "s|/|__|g; s|\.golden$|.ts|")"

    prepare_ts() {
        printf "import { z } from \"zod\";\n" > "$1"
        sed "/^\/\/ @/d" "$file" >> "$1"
    }

    case "${version}" in
        v3)
            prepare_ts "/test/zod3/${ts_name}"
            zod3_count=$((zod3_count + 1))
            ;;
        v4)
            prepare_ts "/test/zod4/${ts_name}"
            zod4_count=$((zod4_count + 1))
            ;;
        *)
            prepare_ts "/test/zod3/${ts_name}"
            prepare_ts "/test/zod4/${ts_name}"
            zod3_count=$((zod3_count + 1))
            zod4_count=$((zod4_count + 1))
            ;;
    esac
done

echo "Found ${zod3_count} files for zod@3, ${zod4_count} files for zod@4"
echo ""

for dir in zod3 zod4; do
    cat > "/test/${dir}/tsconfig.json" <<TSCONFIG
{
  "compilerOptions": {
    "strict": true,
    "noEmit": true,
    "moduleResolution": "node",
    "esModuleInterop": true,
    "target": "ES2020",
    "module": "ES2020"
  },
  "include": ["*.ts"]
}
TSCONFIG
done

cat > /test/zod3/package.json <<PKG
{
  "name": "zen-typecheck-zod3",
  "private": true,
  "dependencies": { "zod": "^3", "typescript": "^5" }
}
PKG

cat > /test/zod4/package.json <<PKG
{
  "name": "zen-typecheck-zod4",
  "private": true,
  "dependencies": { "zod": "^4", "typescript": "^5" }
}
PKG

for dir in zod3 zod4; do
    label="zod@${dir#zod}"
    count_var="${dir}_count"
    count=$(eval echo "\$$count_var")

    if [ "$count" -eq 0 ]; then
        echo "No files to type-check for ${label}, skipping..."
        echo ""
        continue
    fi

    echo "Type-checking ${count} golden files with ${label}..."
    echo "----------------------------------------"

    cd "/test/${dir}"
    npm install --silent 2>&1
    npx tsc --noEmit

    echo ""
    echo "✓ ${label}: PASSED"
    echo ""
done

echo "========================================"
echo "All type checks passed!"
echo "========================================"
'
