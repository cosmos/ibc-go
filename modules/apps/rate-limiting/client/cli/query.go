package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

const (
	FlagDenom = "denom"
)

// GetCmdQueryRateLimit implements a command to query rate limits by channel-id or client-id and denom
func GetCmdQueryRateLimit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rate-limit [channel-or-client-id]",
		Short: "Query rate limits from a given channel-id/client-id and denom",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query rate limits from a given channel-id/client-id and denom.
If the denom flag is omitted, all rate limits for the given channel-id/client-id are returned.

Example:
  $ %s query %s rate-limit [channel-or-client-id]
  $ %s query %s rate-limit [channel-or-client-id] --denom=[denom]
`,
				version.AppName, types.ModuleName, version.AppName, types.ModuleName,
			),
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			channelOrClientID := args[0]
			denom, err := cmd.Flags().GetString(FlagDenom)
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			// Query all rate limits for the channel/client ID if denom is not specified.
			if denom == "" {
				req := &types.QueryRateLimitsByChannelOrClientIDRequest{
					ChannelOrClientId: channelOrClientID,
				}
				res, err := queryClient.RateLimitsByChannelOrClientID(context.Background(), req)
				if err != nil {
					return err
				}
				return clientCtx.PrintProto(res)
			}

			// Query specific rate limit if denom is provided
			req := &types.QueryRateLimitRequest{
				Denom:             denom,
				ChannelOrClientId: channelOrClientID,
			}
			res, err := queryClient.RateLimit(context.Background(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res.RateLimit)
		},
	}

	cmd.Flags().String(FlagDenom, "", "The denom identifying a specific rate limit")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdQueryAllRateLimits return all available rate limits.
func GetCmdQueryAllRateLimits() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-rate-limits",
		Short: "Query for all rate limits",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryAllRateLimitsRequest{}
			res, err := queryClient.AllRateLimits(context.Background(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdQueryRateLimitsByChainID return all rate limits that exist between this chain
// and the specified ChainId
func GetCmdQueryRateLimitsByChainID() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rate-limits-by-chain [chain-id]",
		Short: "Query for all rate limits associated with the channels/clients connecting to the given ChainID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			chainID := args[0]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryRateLimitsByChainIDRequest{
				ChainId: chainID,
			}
			res, err := queryClient.RateLimitsByChainID(context.Background(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdQueryAllBlacklistedDenoms returns the command to query all blacklisted denoms
func GetCmdQueryAllBlacklistedDenoms() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-blacklisted-denoms",
		Short: "Query for all blacklisted denoms",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryAllBlacklistedDenomsRequest{}
			res, err := queryClient.AllBlacklistedDenoms(context.Background(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryAllWhitelistedAddresses returns the command to query all whitelisted address pairs
func GetCmdQueryAllWhitelistedAddresses() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-whitelisted-addresses",
		Short: "Query for all whitelisted address pairs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryAllWhitelistedAddressesRequest{}
			res, err := queryClient.AllWhitelistedAddresses(context.Background(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
