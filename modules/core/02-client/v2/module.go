package clientv2

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
)

// Name returns the IBC channel ICS name.
func Name() string {
	return types.SubModuleName
}

// GetTxCmd returns the root tx command for IBC channels.
func GetTxCmd() *cobra.Command {
	return nil // TODO
}

// GetQueryCmd returns the root query command for IBC channels.
func GetQueryCmd() *cobra.Command {
	return nil // TODO
}
