package port

import (
	"github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v7/modules/core/client/cli"
	"github.com/spf13/cobra"
)

// Name returns the IBC port ICS name.
func Name() string {
	return types.SubModuleName
}

// GetQueryCmd returns the root query command for IBC ports.
func GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}
