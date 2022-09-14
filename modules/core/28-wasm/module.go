package wasm

import (
	"github.com/cosmos/ibc-go/v5/modules/core/28-wasm/cli"
	"github.com/cosmos/ibc-go/v5/modules/core/28-wasm/types"
	"github.com/gogo/protobuf/grpc"
	"github.com/spf13/cobra"
)

// Name returns the IBC channel ICS name.
func Name() string {
	return "wasm"
}

// GetTxCmd returns the root tx command for IBC channels.
func GetTxCmd() *cobra.Command {
	return cli.NewTxCmd()
}

// GetQueryCmd returns the root query command for IBC channels.
func GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// RegisterQueryService registers the gRPC query service for IBC channels.
func RegisterQueryService(server grpc.Server, queryServer types.QueryServer) {
	types.RegisterQueryServer(server, queryServer)
}
