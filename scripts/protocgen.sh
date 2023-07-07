#!/usr/bin/env bash

set -eo pipefail

echo "Generating gogo proto code"
cd proto
proto_dirs=$(find ./ibc ./capability -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  for file in $(find "${dir}" -maxdepth 1 -name '*.proto'); do
    # this regex checks if a proto file has its go_package set to github.com/cosmos/ibc-go...
    # gogo proto files SHOULD ONLY be generated if this is false
    # we don't want gogo proto to run for proto files which are natively built for google.golang.org/protobuf
    if grep -q "option go_package" "$file" && grep -H -o -c 'option go_package.*github.com/cosmos/ibc-go' "$file" | grep -q ':1$'; then
      buf generate --template buf.gen.gogo.yaml $file
    fi
  done
done

cd ..

# move proto files to the right places
cp -r github.com/cosmos/ibc-go/v*/modules/* modules/
cp -r github.com/cosmos/ibc-go/modules/* modules/
rm -rf github.com

go mod tidy

./scripts/protocgen-pulsar.sh

# move pulsar files to the right places
cp -r api/github.com/cosmos/ibc-go/api/* api/
rm -rf api/github.com
