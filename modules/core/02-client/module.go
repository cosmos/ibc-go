package client

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/client/cli"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
)

// Name returns the IBC client name
func Name() string {
	return types.SubModuleName
}

// GetQueryCmd returns no root query command for the IBC client
func GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// GetTxCmd returns the root tx command for 02-client.
func GetTxCmd() *cobra.Command {
	return cli.NewTxCmd()
}
