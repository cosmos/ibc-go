#!/usr/bin/env bash

set -eo pipefail

echo "Generating gogo proto code"
cd proto

buf generate --template buf.gen.gogo.yaml $file

cd ..

# move proto files to the right places
cp -r github.com/cosmos/ibc-go/v*/modules/* modules/
cp -r github.com/cosmos/ibc-go/modules/* modules/
rm -rf github.com

# copy legacy denom trace to internal/types
mv modules/apps/transfer/types/denomtrace.pb.go modules/apps/transfer/internal/types/

go mod tidy
