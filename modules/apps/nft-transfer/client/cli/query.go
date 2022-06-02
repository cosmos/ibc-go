package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"

	"github.com/cosmos/ibc-go/v3/modules/apps/nft-transfer/types"
)

// GetCmdQueryClassTrace defines the command to query a a class trace from a given trace hash or ibc class.
func GetCmdQueryClassTrace() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "class-trace [hash/class]",
		Short:   "Query the class trace info from a given trace hash or ibc class",
		Long:    "Query the class trace info from a given trace hash or ibc class",
		Example: fmt.Sprintf("%s query nft-transfer class-trace 27A6394C3F9FF9C9DCF5DFFADF9BB5FE9A37C7E92B006199894CF1824DF9AC7C", version.AppName),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryClassTraceRequest{
				Hash: args[0],
			}

			res, err := queryClient.ClassTrace(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryClassTraces defines the command to query all the class trace infos
// that this chain mantains.
func GetCmdQueryClassTraces() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "class-traces",
		Short:   "Query the trace info for all the class",
		Long:    "Query the trace info for all the class",
		Example: fmt.Sprintf("%s query nft-transfer class-traces", version.AppName),
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			req := &types.QueryClassTracesRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.ClassTraces(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "class trace")

	return cmd
}

// GetCmdQueryEscrowAddress returns the command handler for nft-transfer escrow-address querying.
func GetCmdQueryEscrowAddress() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "escrow-address",
		Short:   "Get the escrow address for a channel",
		Long:    "Get the escrow address for a channel",
		Args:    cobra.ExactArgs(2),
		Example: fmt.Sprintf("%s query nft-transfer escrow-address [port] [channel-id]", version.AppName),
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

// GetCmdQueryClassHash defines the command to query a class hash from a given trace.
func GetCmdQueryClassHash() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "class-hash [trace]",
		Short:   "Query the class hash info from a given class trace",
		Long:    "Query the class hash info from a given class trace",
		Example: fmt.Sprintf("%s query nft-transfer class-hash transfer/channel-0/class-id", version.AppName),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryClassHashRequest{
				Trace: args[0],
			}

			res, err := queryClient.ClassHash(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
