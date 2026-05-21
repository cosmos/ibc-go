#!/usr/bin/env bash
set -e

mapfile -t PKGS < <(go list ./... | grep -v '/simapp')

echo "mode: atomic" > coverage.txt
for pkg in "${PKGS[@]}"; do
    go test -v -timeout 30m -race -coverprofile=profile.out -covermode=atomic -tags='ledger test_ledger_mock' "$pkg"
    if [ -f profile.out ]; then
        tail -n +2 profile.out >> coverage.txt;
        rm profile.out
    fi
done
