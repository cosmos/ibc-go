package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/ibc-go/v5/modules/core/28-wasm/types"
	"github.com/spf13/cobra"
)

// GetQueryCmd returns the query commands for IBC channels
func GetQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.SubModuleName,
		Short:                      "IBC wasm manager module query subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		GetCmdCode(),
	)

	return queryCmd
}

// NewTxCmd returns a CLI command handler for all x/ibc channel transaction commands.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.SubModuleName,
		Short:                      "IBC wasm manager module transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewPushNewWasmCodeCmd(),
	)

	return txCmd
}
