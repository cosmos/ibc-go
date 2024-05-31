package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
)

// GetCmdQueryDenom defines the command to query a denomination from a given hash or ibc denom.
func GetCmdQueryDenom() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "denom [hash/denom]",
		Short:   "Query the denom trace info from a given hash or ibc denom",
		Long:    "Query the denom trace info from a given hash or ibc denom",
		Example: fmt.Sprintf("%s query ibc-transfer denom 27A6394C3F9FF9C9DCF5DFFADF9BB5FE9A37C7E92B006199894CF1824DF9AC7C", version.AppName),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryV2Client(clientCtx)

			req := &types.QueryDenomRequest{
				Hash: args[0],
			}

			res, err := queryClient.Denom(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryDenoms defines the command to query all the denominations that this chain maintains.
func GetCmdQueryDenoms() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "denoms",
		Short:   "Query for all token denominations",
		Long:    "Query for all token denominations",
		Example: fmt.Sprintf("%s query ibc-transfer denoms", version.AppName),
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryV2Client(clientCtx)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			req := &types.QueryDenomsRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.Denoms(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "denominations")

	return cmd
}

// GetCmdParams returns the command handler for ibc-transfer parameter querying.
func GetCmdParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "params",
		Short:   "Query the current ibc-transfer parameters",
		Long:    "Query the current ibc-transfer parameters",
		Args:    cobra.NoArgs,
		Example: fmt.Sprintf("%s query ibc-transfer params", version.AppName),
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res.Params)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdQueryEscrowAddress returns the command handler for ibc-transfer parameter querying.
func GetCmdQueryEscrowAddress() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "escrow-address",
		Short:   "Get the escrow address for a channel",
		Long:    "Get the escrow address for a channel",
		Args:    cobra.ExactArgs(2),
		Example: fmt.Sprintf("%s query ibc-transfer escrow-address [port] [channel-id]", version.AppName),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			port := args[0]
			channel := args[1]
			addr := types.GetEscrowAddress(port, channel)
			return clientCtx.PrintString(fmt.Sprintf("%s\n", addr.String()))
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdQueryDenomHash defines the command to query a denomination hash from a given trace.
func GetCmdQueryDenomHash() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "denom-hash [trace]",
		Short:   "Query the denom hash info from a given denom trace",
		Long:    "Query the denom hash info from a given denom trace",
		Example: fmt.Sprintf("%s query ibc-transfer denom-hash transfer/channel-0/uatom", version.AppName),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryDenomHashRequest{
				Trace: args[0],
			}

			res, err := queryClient.DenomHash(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryTotalEscrowForDenom defines the command to query the total amount of tokens in escrow for a denom
func GetCmdQueryTotalEscrowForDenom() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "total-escrow [denom]",
		Short:   "Query the total amount of tokens in escrow for a denom",
		Long:    "Query the total amount of tokens in escrow for a denom",
		Example: fmt.Sprintf("%s query ibc-transfer total-escrow uosmo", version.AppName),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryTotalEscrowForDenomRequest{
				Denom: args[0],
			}

			res, err := queryClient.TotalEscrowForDenom(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
