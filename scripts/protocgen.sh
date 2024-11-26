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

echo "Generating prost proto code"
cd proto

buf generate --template buf.gen.prost.yaml $file

cd ..

mv prostgen/mod.rs prostgen/lib.rs
cp prostgen/* modules/light-clients/08-wasm-light-clients/packages/ibc-go-proto/src/
rm -rf prostgen
