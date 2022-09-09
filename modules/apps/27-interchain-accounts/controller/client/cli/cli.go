package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
)

// GetQueryCmd returns the query commands for the ICA controller submodule
func GetQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        "controller",
		Short:                      "interchain-accounts controller subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
	}

	queryCmd.AddCommand(
		GetCmdQueryInterchainAccount(),
		GetCmdParams(),
	)

	return queryCmd
}

// NewTxCmd creates and returns the tx command
func NewTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "controller",
		Short:                      "ica controller transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		newRegisterAccountCmd(),
		newSubmitTxCmd(),
	)

	return cmd
}
