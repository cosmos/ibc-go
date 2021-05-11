package wasm

import (
	"github.com/cosmos/ibc-go/modules/light-clients/10-wasm/client/cli"
	"github.com/cosmos/ibc-go/modules/light-clients/10-wasm/types"
	"github.com/spf13/cobra"
)

// Name returns the IBC client name
func Name() string {
	return types.SubModuleName
}

// GetTxCmd returns the root tx command for the IBC client
func GetTxCmd() *cobra.Command {
	return cli.NewTxCmd()
}
