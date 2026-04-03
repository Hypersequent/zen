#!/bin/bash
#
# Type-checks and runtime-tests golden files inside Docker (zod v3 + v4).
#
# Golden files must contain these metadata comments to be included:
#   // @typecheck             — present by default; files without it are skipped
#   // @zod-version: v3|v4   — (optional) restrict to one zod major; omit for both

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "========================================"
echo "Golden File Type-Check (Docker)"
echo "========================================"
echo ""

docker run --rm \
    -v "${PROJECT_DIR}/testdata:/golden:ro" \
    -v "${PROJECT_DIR}/tests:/tests:ro" \
    node:22-alpine \
    sh -c '
set -e

mkdir -p /test/zod3 /test/zod4 /test/golden

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

    # Also copy to /test/golden/ for runtime tests (all versions)
    prepare_ts "/test/golden/${ts_name}"
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
echo "Type checks passed!"
echo "========================================"
echo ""

# --- Phase 2: Runtime tests ---

echo "========================================"
echo "Runtime Tests (vitest)"
echo "========================================"
echo ""

for dir in zod3 zod4; do
    label="zod@${dir#zod}"
    version="${dir#zod}"  # "3" or "4"
    runtime_dir="/test/runtime-${dir}"

    mkdir -p "${runtime_dir}"

    # Copy test files
    cp /tests/cases.ts "${runtime_dir}/"
    cp /tests/golden.test.ts "${runtime_dir}/"

    zod_dep="^${version}"

    cat > "${runtime_dir}/package.json" <<PKG
{
  "name": "zen-runtime-tests-${dir}",
  "private": true,
  "type": "module",
  "dependencies": { "zod": "${zod_dep}", "vitest": "^3" }
}
PKG

    cat > "${runtime_dir}/tsconfig.json" <<TSCONFIG
{
  "compilerOptions": {
    "strict": true,
    "noEmit": true,
    "moduleResolution": "bundler",
    "esModuleInterop": true,
    "target": "ES2022",
    "module": "ES2022",
    "skipLibCheck": true
  },
  "include": ["*.ts"]
}
TSCONFIG

    echo "Running runtime tests with ${label}..."
    echo "----------------------------------------"

    cd "${runtime_dir}"
    npm install --silent 2>&1
    ZOD_VERSION="v${version}" npx vitest run --reporter=verbose

    echo ""
    echo "✓ ${label} runtime: PASSED"
    echo ""
done

echo "========================================"
echo "All checks passed!"
echo "========================================"
'
