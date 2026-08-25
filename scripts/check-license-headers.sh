#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Verifies that every eligible source file carries an `SPDX-License-Identifier` header
# matching the license this repository is published under (see ../LICENSE.md).
#
#   ./scripts/check-license-headers.sh            # report violations, exit 1 if any
#   ./scripts/check-license-headers.sh --fix      # insert/correct headers in place
#   ./scripts/check-license-headers.sh --list     # print the set of files that are checked
#
# Only files tracked by git are considered. Generated code, vendored code, and components
# published under a deliberately different license are skipped -- see EXCLUDED_PATHS and
# PRESERVED_HEADERS below.

set -euo pipefail

LICENSE="Apache-2.0"
SPDX_TAG="SPDX-License-Identifier: ${LICENSE}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${REPO_ROOT}"

# Extensions that must carry a header, and the comment marker to use for each.
#   //  -> rust, go, solidity, protobuf
#   #   -> yaml, nix, shell, just
slash_exts="rs go sol proto"
hash_exts="yml yaml nix sh just"

# Paths (git pathspec globs) that are never checked. Each entry is paired with the reason
# it is excluded so this list stays auditable.
EXCLUDED_PATHS=(
  # GPL-3.0-or-later code derived from Open Ethereum / Parity Technologies. Stamping an
  # Apache-2.0 header here would misstate the license. Pending a legal decision.
  'packages/ethereum/trie-db/**'

  # Any `generated/` directory holds build output -- e.g. the Anchor IDL bindings that
  # solana-ibc-sdk/build.rs rewrites on every `cargo build`. Headers written there would be
  # silently discarded on the next regeneration.
  '**/generated/**'

  # Third-party dependencies and build output that is not ours to license.
  '**/node_modules/**'
  '**/target/**'
  '**/out/**'
  '**/lib/forge-std/**'
  'ibc-solidity/lib/**'
)

# Files whose existing SPDX header is intentionally different from ${LICENSE} and must be
# left alone. Anything listed here is checked for *having* a header, but its value is not
# forced to ${LICENSE}.
#
# Deliberately empty: every file this repository owns is published under ${LICENSE}. This
# stays as an escape hatch for future third-party code that cannot be relicensed -- add the
# path together with a comment saying why, rather than weakening the check.
PRESERVED_HEADERS=()

# Markers that identify machine-generated files, matched case-insensitively against the
# first 10 lines. Headers added to such files would be lost on the next regeneration.
#
# The line must look like a comment and contain one of the banner phrases below. Bare
# "auto-generated" is deliberately NOT a marker: it shows up in ordinary prose, e.g.
# packages/go-anchor/ics07_tendermint_patches/instructions.go, whose doc comment explains
# that it sits outside "the auto-generated packages" on purpose and is hand-written.
GENERATED_MARKERS='^[[:space:]]*(//|#|/\*|\*|--).*(do[[:space:]]+not[[:space:]]+edit|code[[:space:]]+generated|generated[[:space:]]+by|@generated|auto-generated[[:space:]]+file)'

is_preserved() {
  local f=$1 p
  # Expanding an empty array under `set -u` is an error on bash 3.2 (still the system bash
  # on macOS), so bail out before the loop.
  ((${#PRESERVED_HEADERS[@]} == 0)) && return 1
  for p in "${PRESERVED_HEADERS[@]}"; do
    [[ "$f" == "$p" ]] && return 0
  done
  return 1
}

is_generated() {
  head -n 10 -- "$1" 2>/dev/null | grep -qiE "${GENERATED_MARKERS}"
}

# Emits the list of tracked, non-generated, non-excluded files that require a header.
collect_files() {
  local pathspecs=() ext
  for ext in ${slash_exts} ${hash_exts}; do
    pathspecs+=("*.${ext}")
  done
  # `justfile` and `*.just` are both just-language files; the former has no extension.
  pathspecs+=('justfile' '*/justfile')

  local excludes=() p
  for p in "${EXCLUDED_PATHS[@]}"; do
    excludes+=(":(exclude,glob)${p}")
  done

  git ls-files -z -- "${pathspecs[@]}" "${excludes[@]}" \
    | while IFS= read -r -d '' f; do
        [[ -f "$f" ]] || continue
        # Skip symlinks (e.g. ibc-solana/programs/test-access-manager/src/*.rs, which point
        # into access-manager/src/). Writing through one would replace the link with a
        # regular file; the target is stamped under its own path instead.
        [[ -L "$f" ]] && continue
        is_generated "$f" && continue
        printf '%s\0' "$f"
      done
}

comment_marker_for() {
  case "$1" in
    justfile | */justfile) printf '#' ;;
    *.*)
      local ext=${1##*.} e
      for e in ${hash_exts}; do
        [[ "$ext" == "$e" ]] && { printf '#'; return; }
      done
      printf '//'
      ;;
    *) printf '#' ;;
  esac
}

# Prints the SPDX value already present in the first 10 lines, or nothing. A file with no
# header is the normal case rather than an error, so the non-matching `grep` is swallowed
# to keep `set -o pipefail` from aborting the run.
existing_spdx() {
  head -n 10 -- "$1" 2>/dev/null \
    | grep -m1 -oE 'SPDX-License-Identifier:[[:space:]]*[^ */#]+' \
    | sed -E 's/SPDX-License-Identifier:[[:space:]]*//' \
    || true
}

# Overwrites a file's contents from a temp file without replacing the file itself, so the
# mode bits (notably the executable bit on shell scripts) and ownership survive. A plain
# `mv` would drop them.
write_back() {
  local f=$1 tmp=$2
  cat "${tmp}" >"$f"
  rm -f "${tmp}"
}

# Rewrites an existing SPDX value in place, preserving the surrounding comment syntax.
replace_header() {
  local f=$1
  awk -v want="${LICENSE}" '
    !done && /SPDX-License-Identifier:/ {
      sub(/SPDX-License-Identifier:[[:space:]]*[^ *\/#]+/, "SPDX-License-Identifier: " want)
      done = 1
    }
    { print }
  ' "$f" >"$f.spdx.tmp" && write_back "$f" "$f.spdx.tmp"
}

# Inserts a header at the top of a file, after a shebang if one is present.
insert_header() {
  local f=$1
  local marker
  marker=$(comment_marker_for "$f")
  local header="${marker} ${SPDX_TAG}"

  # The shebang pattern requires the `/` of an interpreter path so that Rust inner
  # attributes (`#![deny(...)]`, `#![doc = ...]`) are not mistaken for one.
  awk -v header="${header}" '
    { lines[NR] = $0 }
    END {
      start = 1
      if (NR >= 1 && lines[1] ~ /^#![ \t]*\//) { print lines[1]; start = 2 }
      print header
      # Separate the header from the body with exactly one blank line.
      if (start <= NR && lines[start] != "") print ""
      for (i = start; i <= NR; i++) print lines[i]
    }
  ' "$f" >"$f.spdx.tmp" && write_back "$f" "$f.spdx.tmp"
}

MODE=check
case "${1:-}" in
  --fix) MODE=fix ;;
  --list) MODE=list ;;
  --check | '') MODE=check ;;
  -h | --help)
    sed -n '2,14p' "${BASH_SOURCE[0]}"
    exit 0
    ;;
  *)
    echo "unknown argument: $1" >&2
    exit 2
    ;;
esac

missing=()
mismatched=()
fixed=0

while IFS= read -r -d '' file; do
  if [[ "${MODE}" == list ]]; then
    printf '%s\n' "${file}"
    continue
  fi

  found=$(existing_spdx "${file}")

  if [[ -z "${found}" ]]; then
    if [[ "${MODE}" == fix ]]; then
      insert_header "${file}"
      fixed=$((fixed + 1))
    else
      missing+=("${file}")
    fi
    continue
  fi

  # A header exists. Leave deliberately-different licenses alone.
  is_preserved "${file}" && continue

  if [[ "${found}" != "${LICENSE}" ]]; then
    if [[ "${MODE}" == fix ]]; then
      replace_header "${file}"
      fixed=$((fixed + 1))
    else
      mismatched+=("${file}  (${found})")
    fi
  fi
done < <(collect_files)

if [[ "${MODE}" == list ]]; then
  exit 0
fi

if [[ "${MODE}" == fix ]]; then
  echo "check-license-headers: updated ${fixed} file(s)"
  exit 0
fi

status=0

if ((${#missing[@]} > 0)); then
  status=1
  echo "Missing '${SPDX_TAG}' header (${#missing[@]} file(s)):" >&2
  printf '  %s\n' "${missing[@]}" >&2
fi

if ((${#mismatched[@]} > 0)); then
  status=1
  echo "Unexpected SPDX license, want '${LICENSE}' (${#mismatched[@]} file(s)):" >&2
  printf '  %s\n' "${mismatched[@]}" >&2
fi

if ((status != 0)); then
  echo >&2
  echo "Run 'just lint-license-fix' (or ./scripts/check-license-headers.sh --fix) to correct these." >&2
  echo "If a file is intentionally licensed differently, add it to PRESERVED_HEADERS in" >&2
  echo "scripts/check-license-headers.sh with a comment explaining why." >&2
else
  echo "check-license-headers: all files carry the '${LICENSE}' header"
fi

exit "${status}"
