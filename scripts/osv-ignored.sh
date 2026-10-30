#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Prints the advisories accepted in the root osv-scanner.toml as GHSA identifiers,
# comma-separated, the format actions/dependency-review-action's allow-ghsas input wants.
#
#   ./scripts/osv-ignored.sh
#
# GHSA entries are passed through. Go advisories are keyed differently -- GO-ID in the Go
# database, GHSA in GitHub's -- so they are mapped through the Go database's alias list. An
# entry with no GHSA alias exists only in the Go database, which means the action cannot
# report it and there is nothing to allow; that is reported on stderr and skipped.
#
# A lookup that fails is an error rather than a skip: silently dropping an entry would fail
# the dependency review on an advisory this repository has already accepted.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

GO_VULN_DB="https://vuln.go.dev"

die() {
  echo "osv-ignored: $*" >&2
  exit 2
}

# The header comment is the help text. Printing it up to the first non-comment line, rather
# than to a hardcoded line number, keeps the two from drifting apart as the comment is
# edited -- a stale range silently spills the script's own code into `--help`.
usage() {
  sed -n '3,${/^#/!q; s|^# \{0,1\}||; p;}' "${BASH_SOURCE[0]}"
}

while (($# > 0)); do
  case "$1" in
    -h | --help) usage; exit 0 ;;
    *) die "unknown argument '$1'" ;;
  esac
done

CONFIG="${REPO_ROOT}/osv-scanner.toml"
[[ -f "${CONFIG}" ]] || die "'${CONFIG}' does not exist"

# Parsed with Python's standard TOML parser (3.11+); an entry without an id is an error.
ids="$(python3 -c '
import sys, tomllib
with open(sys.argv[1], "rb") as f:
    print("\n".join(sorted({v["id"] for v in tomllib.load(f).get("IgnoredVulns", [])})))
' "${CONFIG}")" || die "could not read the ids in ${CONFIG}"

command -v jq >/dev/null || die "jq is required"

ghsas=""

add() {
  case ", ${ghsas}, " in
    *", $1, "*) ;;
    *) ghsas="${ghsas:+${ghsas}, }$1" ;;
  esac
}

for id in ${ids}; do
  if [[ "${id}" =~ ^GHSA-[0-9a-z]{4}-[0-9a-z]{4}-[0-9a-z]{4}$ ]]; then
    add "${id}"
    continue
  fi

  # Checked before it reaches a URL: a typo should be a clear error, not an odd request.
  [[ "${id}" =~ ^GO-[0-9]{4}-[0-9]+$ ]] \
    || die "'${id}' in an osv-scanner.toml is neither a GHSA nor a Go advisory ID"

  json="$(curl -fsS --retry 3 --retry-delay 2 "${GO_VULN_DB}/ID/${id}.json")" \
    || die "could not read ${id} from ${GO_VULN_DB}"

  mapped="$(printf '%s' "${json}" | jq -r '(.aliases // [])[] | select(startswith("GHSA-"))')"

  if [[ -z "${mapped}" ]]; then
    echo "${id}: no GHSA alias, so GitHub's advisory database cannot report it" >&2
    continue
  fi

  for ghsa in ${mapped}; do
    add "${ghsa}"
  done
done

printf '%s\n' "${ghsas}"
