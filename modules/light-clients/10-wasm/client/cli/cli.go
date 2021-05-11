package cli

import (
	"github.com/cosmos/ibc-go/modules/light-clients/10-wasm/types"
	"github.com/spf13/cobra"

)

// NewTxCmd returns a root CLI command handler for all x/ibc/light-clients/07-tendermint transaction commands.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.SubModuleName,
		Short:                      "WASM client transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
	}

	txCmd.AddCommand(
		NewCreateClientCmd(),
		NewUpdateClientCmd(),
		NewSubmitMisbehaviourCmd(),
	)

	return txCmd
}
