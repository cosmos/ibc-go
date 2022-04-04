package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
)

// GetQueryCmd returns the query commands for 29-fee
func GetQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        "ibc-fee",
		Short:                      "Query subcommand for IBC relayer incentivization",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
	}

	queryCmd.AddCommand(
		GetCmdTotalRecvFees(),
		GetCmdTotalAckFees(),
		GetCmdTotalTimeoutFees(),
		GetCmdCounterpartyAddress(),
	)

	return queryCmd
}

// NewTxCmd returns the transaction commands for 29-fee
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        "ibc-fee",
		Short:                      "Transaction subcommand for IBC relayer incentivization",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewPayPacketFeeAsyncTxCmd(),
		NewRegisterCounterpartyAddress(),
	)

	return txCmd
}
