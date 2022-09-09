package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
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
<<<<<<< HEAD
		newRegisterAccountCmd(),
		newSubmitTxCmd(),
=======
		newRegisterInterchainAccountCmd(),
		newSendTxCmd(),
>>>>>>> a4be561 (chore: rename `SubmitTx` to `SendTx` (#2255))
	)

	return cmd
}
