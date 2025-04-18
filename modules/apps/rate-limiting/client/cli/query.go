package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

// GetQueryCmd returns the CLI query commands for the rate-limiting module
func GetQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the rate-limiting module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	// Add query commands here when defined
	// Example:
	// queryCmd.AddCommand(
	//     GetCmdQueryRateLimit(),
	//     GetCmdQueryParams(),
	// )

	return queryCmd
}

// Example query command structure to be implemented
/*
// GetCmdQueryRateLimit returns the command to query a rate limit
func GetCmdQueryRateLimit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rate-limit [channel-id] [denom]",
		Short: "Query a rate limit for a specific channel and denomination",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			channelID := args[0]
			denom := args[1]

			req := &types.QueryRateLimitRequest{
				ChannelId: channelID,
				Denom:     denom,
			}

			res, err := queryClient.RateLimit(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryParams returns the command to query the module parameters
func GetCmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the current rate-limiting module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
*/
