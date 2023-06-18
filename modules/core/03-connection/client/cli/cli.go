package cli

import (
	"github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	"github.com/spf13/cobra"
)

// GetQueryCmd returns the query commands for IBC connections
func GetQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.SubModuleName,
		Short:                      "IBC connection query subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
	}

	queryCmd.AddCommand(
		GetCmdQueryConnections(),
		GetCmdQueryConnection(),
		GetCmdQueryClientConnections(),
		GetCmdConnectionParams(),
	)

	return queryCmd
}
