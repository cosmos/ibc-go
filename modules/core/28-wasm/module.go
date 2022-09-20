package wasm

import (
	"github.com/cosmos/ibc-go/v5/modules/core/28-wasm/types"
	"github.com/gogo/protobuf/grpc"
)

// RegisterQueryService registers the gRPC query service for IBC channels.
func RegisterQueryService(server grpc.Server, queryServer types.QueryServer) {
	types.RegisterQueryServer(server, queryServer)
}

// TODO: genesis state import and export
// TODO: message server
