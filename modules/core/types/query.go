package types

import (
	"github.com/cosmos/gogoproto/grpc"

	channel "github.com/cosmos/ibc-go/v8/modules/core/04-channel"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

// QueryServer defines the IBC interfaces that the gRPC query server must implement
type QueryServer interface {
	// connectiontypes.QueryServer
	channeltypes.QueryServer
}

// RegisterQueryService registers each individual IBC submodule query service
func RegisterQueryService(server grpc.Server, queryService QueryServer) {
	// connection.RegisterQueryService(server, queryService)
	channel.RegisterQueryService(server, queryService)
}
