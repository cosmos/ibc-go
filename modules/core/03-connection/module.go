package connection

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/ibc-go/v10/modules/core/03-connection/client/cli"
	"github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
)

// Name returns the IBC connection ICS name.
func Name() string {
	return types.SubModuleName
}

// GetQueryCmd returns the root query command for the IBC connections.
func GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}
