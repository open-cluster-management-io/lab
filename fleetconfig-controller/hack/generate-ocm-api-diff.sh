#!/usr/bin/env bash
# Generate raw OCM API diffs between two open-cluster-management.io/api releases.
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

usage() {
  cat <<EOF
Usage: $(basename "$0") <old-version> <new-version>

Compare upstream OCM API releases and write unified diffs under:
  hack/ocm-api-diff-v<old>-to-v<new>/

Versions may be passed with or without a leading "v" (e.g. 1.0.0 or v1.3.0).
Each version must be semver-like: MAJOR.MINOR.PATCH with an optional pre-release suffix.

Example:
  $(basename "$0") 1.0.0 1.3.0
EOF
}

normalize_ver() {
  echo "${1#v}"
}

# Release tags are semver-like (e.g. 1.0.0, 0.16.2). Reject path metacharacters.
validate_ver() {
  local ver="$1"
  local label="$2"
  if [[ -z "$ver" ]]; then
    echo "error: ${label} version must not be empty" >&2
    exit 1
  fi
  if [[ ! "$ver" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
    echo "error: invalid ${label} version '${ver}' (expected semver, e.g. 1.0.0)" >&2
    exit 1
  fi
}

assert_output_dir_safe() {
  local dir="$1"
  if [[ "$dir" != "$script_dir/"* ]]; then
    echo "error: output directory escapes script directory: '${dir}'" >&2
    exit 1
  fi
  local rel="${dir#"$script_dir/"}"
  if [[ ! "$rel" =~ ^ocm-api-diff-v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?-to-v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
    echo "error: unsafe output directory: '${dir}'" >&2
    exit 1
  fi
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -ne 2 ]]; then
  usage >&2
  exit 1
fi

old_ver="$(normalize_ver "$1")"
new_ver="$(normalize_ver "$2")"
validate_ver "$old_ver" "old"
validate_ver "$new_ver" "new"

cache_dir="${TMPDIR:-/tmp}/ocm-api-compare"
old_dir="$cache_dir/api-${old_ver}"
new_dir="$cache_dir/api-${new_ver}"
out_dir="$script_dir/ocm-api-diff-v${old_ver}-to-v${new_ver}"
assert_output_dir_safe "$out_dir"

fetch() {
  local ver="$1"
  local dir="$cache_dir/api-${ver}"
  if [[ -d "$dir" ]]; then
    return
  fi
  mkdir -p "$cache_dir"
  curl -fsSL "https://github.com/open-cluster-management-io/api/archive/refs/tags/v${ver}.tar.gz" \
    | tar -xz -C "$cache_dir"
}

find_api_go_files() {
  local root="$1"
  find "$root"/addon "$root"/cluster "$root"/work "$root"/operator \
    -name '*.go' ! -name 'doc.go' ! -path '*/zz_generated*' -print0 2>/dev/null
}

fetch "$old_ver"
fetch "$new_ver"

rm -rf "$out_dir"
mkdir -p "$out_dir"/{new-api-version,changed-types,feature-gates}

cat >"$out_dir/README.md" <<EOF
# OCM API raw diffs: v${old_ver} → v${new_ver}

Upstream: https://github.com/open-cluster-management-io/api

## Files

- \`new-api-version/\` — API version packages present only in v${new_ver}
- \`changed-types/*.patch\` — per-file unified diffs for API Go sources (types, funcs, etc.)
- \`changed-types/NEW__*.patch\` — files added in v${new_ver}
- \`changed-types/REMOVED__*.patch\` — files removed in v${new_ver}
- \`feature-gates/feature.go.patch\` — feature gate diff
- \`all-changed-types.patch\` — concatenated type diffs

Regenerate:

\`\`\`bash
hack/generate-ocm-api-diff.sh ${old_ver} ${new_ver}
\`\`\`
EOF

# API version packages that exist only in the newer release (e.g. addon/v1beta1).
empty_tree="$(mktemp -d)"
trap 'rm -rf "$empty_tree"' EXIT

for group in addon cluster work operator; do
  [[ -d "$new_dir/$group" ]] || continue
  for ver_path in "$new_dir/$group"/*/; do
    [[ -d "$ver_path" ]] || continue
    ver_name="$(basename "$ver_path")"
    if [[ -d "$old_dir/$group/$ver_name" ]]; then
      continue
    fi
    rel="$group/$ver_name"
    mkdir -p "$out_dir/new-api-version/$(dirname "$rel")"
    cp -R "$ver_path" "$out_dir/new-api-version/$rel"
    mkdir -p "$empty_tree/$rel"
    diff -ruN "$empty_tree/$rel" "$new_dir/$rel" \
      >"$out_dir/new-api-version/${group}-${ver_name}.full.patch" || true
  done
done

# Changed and removed API Go sources
while IFS= read -r -d '' oldf; do
  rel="${oldf#"$old_dir"/}"
  newf="$new_dir/$rel"
  safe="${rel//\//__}"
  if [[ ! -f "$newf" ]]; then
    diff -u "$oldf" /dev/null >"$out_dir/changed-types/REMOVED__${safe}.patch" || true
    continue
  fi
  if ! diff -q "$oldf" "$newf" >/dev/null 2>&1; then
    diff -u "$oldf" "$newf" >"$out_dir/changed-types/${safe}.patch" || true
  fi
done < <(find_api_go_files "$old_dir")

# New API Go files not in the older release
while IFS= read -r -d '' newf; do
  rel="${newf#"$new_dir"/}"
  oldf="$old_dir/$rel"
  [[ -f "$oldf" ]] && continue
  safe="NEW__${rel//\//__}"
  diff -u /dev/null "$newf" >"$out_dir/changed-types/${safe}.patch" || true
done < <(find_api_go_files "$new_dir")

if [[ -f "$old_dir/feature/feature.go" && -f "$new_dir/feature/feature.go" ]]; then
  diff -u "$old_dir/feature/feature.go" "$new_dir/feature/feature.go" \
    >"$out_dir/feature-gates/feature.go.patch" || true
fi

# Indexes and combined
find "$out_dir/changed-types" -name '*.patch' -print0 | xargs -0 -n1 basename 2>/dev/null | sort >"$out_dir/changed-types/INDEX.txt" || true

: >"$out_dir/all-changed-types.patch"
while IFS= read -r f; do
  printf '\n\n======== %s ========\n\n' "$(basename "$f")" >>"$out_dir/all-changed-types.patch"
  cat "$f" >>"$out_dir/all-changed-types.patch"
done < <(find "$out_dir/changed-types" -name '*.patch' | sort)

echo "Wrote $out_dir"
find "$out_dir" -name '*.patch' | wc -l
du -sh "$out_dir"
