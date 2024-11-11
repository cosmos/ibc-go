package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
)

// GetQueryCmd returns the query commands for the IBC channel/v2.
func GetQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.SubModuleName,
		Short:                      "IBC channel/v2 query subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		getCmdQueryChannel(),
		getCmdQueryNextSequenceSend(),
		getCmdQueryPacketCommitment(),
		getCmdQueryPacketCommitments(),
		getCmdQueryPacketAcknowledgement(),
		getCmdQueryPacketReceipt(),
	)

	return queryCmd
}

// NewTxCmd returns the command to submit transactions defined for IBC channel/v2.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.SubModuleName,
		Short:                      "IBC channel/v2 transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		newCreateChannelTxCmd(),
		newRegisterCounterpartyTxCmd(),
	)

	return txCmd
}
