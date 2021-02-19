go 1.15

module github.com/cosmos/ibc-go

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

require (
	github.com/confio/ics23/go v0.6.3
	github.com/cosmos/cosmos-sdk v0.41.3
	github.com/gogo/protobuf v1.3.3
	github.com/golang/protobuf v1.4.3
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/tendermint/tendermint v0.34.7
	google.golang.org/genproto v0.0.0-20210114201628-6edceaf6022f
	google.golang.org/grpc v1.35.0
)
