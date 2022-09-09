package cli

import (
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
<<<<<<< HEAD
=======

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
		newRegisterInterchainAccountCmd(),
		newSubmitTxCmd(),
	)

	return cmd
}
>>>>>>> f8f226d (chore: rename `RegisterAccount` rpc and msgs to `RegisterInterchainAccount` (#2253))
