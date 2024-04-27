module github.com/cosmos/ibc-go/api

go 1.21

// NOTE: This replace points the ics23 code deps used by 23-commitment to a branch using pulsar codegen (i.e. protov2 compatible encoding interfaces)
// This should be removed and reverted when depinject supports protov1 with gogoproto.
replace github.com/cosmos/ics23/go/api => github.com/cosmos/ics23/go/api v0.0.0-20240418174942-ccce00eba150

require (
	cosmossdk.io/api v0.7.2
	github.com/cosmos/cosmos-proto v1.0.0-beta.4
	github.com/cosmos/gogoproto v1.4.11
	github.com/cosmos/ics23/go/api v0.0.0
	google.golang.org/genproto/googleapis/api v0.0.0-20231012201019-e917dd12ba7a
	google.golang.org/protobuf v1.33.0
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	golang.org/x/exp v0.0.0-20231006140011-7918f672742d // indirect
	golang.org/x/net v0.19.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto v0.0.0-20231016165738-49dd2c1f3d0b // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231030173426-d783a09b4405 // indirect
	google.golang.org/grpc v1.59.0 // indirect
)
