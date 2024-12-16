package client

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/client/cli"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
)

// Name returns the IBC channel/v2 name
func Name() string {
	return types.SubModuleName
}

// GetQueryCmd returns the root query command for IBC channels v2.
func GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// GetTxCmd returns the root tx command for IBC channels v2.
func GetTxCmd() *cobra.Command {
	return cli.NewTxCmd()
}
