#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

for modfile in $(find . -name go.mod); do
 echo "Updating $modfile"
 DIR=$(dirname $modfile)
 (cd $DIR; go mod tidy)
done
