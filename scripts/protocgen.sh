#!/usr/bin/env bash

set -eo pipefail

echo "Generating gogo proto code"
cd proto

buf generate --template buf.gen.gogo.yaml $file

cd ..

# move proto files to the right places
cp -r github.com/cosmos/ibc-go/v*/modules/* modules/
cp -r github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v*/* modules/light-clients/08-wasm/
# If other modules are added later with protos, they need to be added above here 👆
rm -rf github.com

# copy legacy denom trace to internal/types
mv modules/apps/transfer/types/denomtrace.pb.go modules/apps/transfer/internal/types/

go mod tidy
