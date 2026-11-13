#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Verifies that every eligible file carries an `SPDX-License-Identifier` header matching
# the license CONTRIBUTING.md assigns to it: code under Apache-2.0 (see ../LICENSE) and
# documentation (Markdown and MDX) under CC-BY-4.0 (see ../docs/LICENSE).
#
#   ./scripts/check-license-headers.sh            # report violations, exit 1 if any
#   ./scripts/check-license-headers.sh --fix      # insert/correct headers in place
#   ./scripts/check-license-headers.sh --list     # print the set of files that are checked
#
# Only files tracked by git are considered. Generated code, vendored code, and components
# published under a deliberately different license are skipped -- see EXCLUDED_PATHS and
# PRESERVED_HEADERS below.

set -euo pipefail

CODE_LICENSE="Apache-2.0"
DOCS_LICENSE="CC-BY-4.0"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${REPO_ROOT}"

# Extensions that must carry a header, and the comment marker to use for each.
#   //  -> rust, go, solidity, protobuf
#   #   -> yaml, nix, shell, just
slash_exts="rs go sol proto"
hash_exts="yml yaml nix sh just"
#   <!-- --> or YAML front matter -> markdown, mdx (documentation, under ${DOCS_LICENSE})
doc_exts="md mdx"

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

  # Docusaurus snapshots of the documentation as it stood at each past release. They are
  # frozen copies, not edited in place, so they keep the terms they were published under.
  'docs/versioned_docs/**'
)

# Files whose existing SPDX header is intentionally different from the license chosen by
# license_for and must be left alone. Anything listed here is checked for *having* a
# header, but its value is not forced.
#
# Deliberately empty: every file this repository owns follows CONTRIBUTING.md. This
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
  for ext in ${slash_exts} ${hash_exts} ${doc_exts}; do
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

# Prints the license a file must carry: documentation under ${DOCS_LICENSE}, code under
# ${CODE_LICENSE}.
license_for() {
  case "$1" in
    *.md | *.mdx) printf '%s' "${DOCS_LICENSE}" ;;
    *) printf '%s' "${CODE_LICENSE}" ;;
  esac
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
  awk -v want="$(license_for "$f")" '
    !done && /SPDX-License-Identifier:/ {
      sub(/SPDX-License-Identifier:[[:space:]]*[^ *\/#]+/, "SPDX-License-Identifier: " want)
      done = 1
    }
    { print }
  ' "$f" >"$f.spdx.tmp" && write_back "$f" "$f.spdx.tmp"
}

# Directories Docusaurus renders (see ../docs/docusaurus.config.js). Docusaurus takes a
# page title from a leading `#` heading only if nothing comes before it, so a page here
# without front matter gets a front matter block for its header instead of an HTML comment.
DOCUSAURUS_DIRS="docs/docs docs/architecture docs/events docs/params"

is_docusaurus_page() {
  local d
  for d in ${DOCUSAURUS_DIRS}; do
    [[ "$1" == "$d"/* ]] && return 0
  done
  return 1
}

# Inserts a header at the top of a Markdown or MDX file. A file that opens with YAML front
# matter gets a YAML comment as its first front matter line, since the front matter must
# stay on line 1 for GitHub and Docusaurus to recognise it. A Docusaurus page without front
# matter gets a front matter block holding only that comment. Any other file gets an HTML
# comment on line 1.
insert_doc_header() {
  local f=$1
  local tag="SPDX-License-Identifier: $(license_for "$f")"
  local frontmatter=0
  is_docusaurus_page "$f" && frontmatter=1
  awk -v tag="${tag}" -v frontmatter="${frontmatter}" '
    { lines[NR] = $0 }
    END {
      if (NR >= 1 && lines[1] == "---") {
        print lines[1]
        print "# " tag
        for (i = 2; i <= NR; i++) print lines[i]
        exit
      }
      if (frontmatter) {
        print "---"
        print "# " tag
        print "---"
      } else {
        print "<!-- " tag " -->"
      }
      # Separate the header from the body with exactly one blank line.
      if (NR >= 1 && lines[1] != "") print ""
      for (i = 1; i <= NR; i++) print lines[i]
    }
  ' "$f" >"$f.spdx.tmp" && write_back "$f" "$f.spdx.tmp"
}

# Inserts a header at the top of a file, after a shebang if one is present.
insert_header() {
  local f=$1
  case "$f" in
    *.md | *.mdx)
      insert_doc_header "$f"
      return
      ;;
  esac
  local marker
  marker=$(comment_marker_for "$f")
  local header="${marker} SPDX-License-Identifier: $(license_for "$f")"

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

  want=$(license_for "${file}")
  if [[ "${found}" != "${want}" ]]; then
    if [[ "${MODE}" == fix ]]; then
      replace_header "${file}"
      fixed=$((fixed + 1))
    else
      mismatched+=("${file}  (${found}, want ${want})")
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
  echo "Missing 'SPDX-License-Identifier' header (${#missing[@]} file(s)):" >&2
  printf '  %s\n' "${missing[@]}" >&2
fi

if ((${#mismatched[@]} > 0)); then
  status=1
  echo "Unexpected SPDX license (${#mismatched[@]} file(s)):" >&2
  printf '  %s\n' "${mismatched[@]}" >&2
fi

if ((status != 0)); then
  echo >&2
  echo "Run './scripts/check-license-headers.sh --fix' to correct these." >&2
  echo "If a file is intentionally licensed differently, add it to PRESERVED_HEADERS in" >&2
  echo "scripts/check-license-headers.sh with a comment explaining why." >&2
else
  echo "check-license-headers: all files carry the '${CODE_LICENSE}' or '${DOCS_LICENSE}' header"
fi

exit "${status}"
