module github.com/cosmos/ibc-go/e2e

go 1.18

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

// TODO: using version v1.0.0 causes a build failure. This is the previous version which compiles successfully.
replace (
	github.com/ChainSafe/go-schnorrkel => github.com/ChainSafe/go-schnorrkel v0.0.0-20200405005733-88cbf1b4c40d
	github.com/ChainSafe/go-schnorrkel/1 => github.com/ChainSafe/go-schnorrkel v1.0.0
	github.com/vedhavyas/go-subkey => github.com/strangelove-ventures/go-subkey v1.0.7
)

// uncomment to use the local version of ibc-go, you will need to run `go mod tidy` in e2e directory.
replace github.com/cosmos/ibc-go/v6 => ../
