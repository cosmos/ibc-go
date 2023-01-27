package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	wasm "github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm"
	"github.com/spf13/cobra"
)

// GetQueryCmd returns the query commands for IBC channels
func GetQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        wasm.SubModuleName,
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
		Use:                        wasm.SubModuleName,
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
