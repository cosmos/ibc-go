package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
)

// GetQueryCmd returns the query commands for IBC connections
//func GetQueryCmd() *cobra.Command {
//	queryCmd := &cobra.Command{
//		Use:                        "ibc-icq",
//		Short:                      "IBC interchain query query subcommands",
//		DisableFlagParsing:         true,
//		SuggestionsMinimumDistance: 2,
//	}
//
//	queryCmd.AddCommand(
//		GetCmdQueryDenomTrace(),
//		GetCmdQueryDenomTraces(),
//		GetCmdParams(),
//		GetCmdQueryEscrowAddress(),
//		GetCmdQueryDenomHash(),
//	)
//
//	return queryCmd
//}

// NewTxCmd returns the transaction commands for IBC fungible token transfer
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        "ibc-icq",
		Short:                      "IBC interchain query transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewQueryTxCmd(),
	)

	return txCmd
}
