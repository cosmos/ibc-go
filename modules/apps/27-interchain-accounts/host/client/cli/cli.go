package cli

import (
	"github.com/spf13/cobra"
)

// GetQueryCmd returns the query commands for the ICA host submodule
func GetQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        "host",
		Short:                      "IBC interchain accounts host query subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
	}

	queryCmd.AddCommand(
		GetCmdParams(),
		GetCmdPacketEvents(),
	)

	return queryCmd
}
