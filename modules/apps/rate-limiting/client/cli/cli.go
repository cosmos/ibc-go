package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
)

// GetQueryCmd returns the cli query commands for this module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "ratelimiting",
		Short:                      "IBC ratelimiting querying subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetCmdQueryRateLimit(),
		GetCmdQueryAllRateLimits(),
		GetCmdQueryRateLimitsByChainID(),
		GetCmdQueryAllBlacklistedDenoms(),
		GetCmdQueryAllWhitelistedAddresses(),
	)
	return cmd
}
