package cli

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/spf13/cobra"

	"github.com/cosmos/ibc-go/v3/modules/apps/27-interchain-accounts/host/types"
)

// GetCmdParams returns the command handler for the host submodule parameter querying.
func GetCmdParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "params",
		Short:   "Query the current interchain-accounts host submodule parameters",
		Long:    "Query the current interchain-accounts host submodule parameters",
		Args:    cobra.NoArgs,
		Example: fmt.Sprintf("%s query interchain-accounts host params", version.AppName),
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

// GetCmdPacketEvents returns the command handler for the host packet events querying.
func GetCmdPacketEvents() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "packet-events [channel-id] [sequence]",
		Short:   "Query the interchain-accounts host submodule packet events",
		Long:    "Query the interchain-accounts host submodule packet events",
		Args:    cobra.ExactArgs(2),
		Example: fmt.Sprintf("%s query interchain-accounts host packet-events channel-0 100", version.AppName),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			seq, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}

			req := &types.QueryPacketEventsRequest{
				ChannelId: args[0],
				Sequence:  seq,
			}

			res, err := queryClient.PacketEvents(cmd.Context(), req)
			if err != nil {
				return err
			}

			txResp, err := tx.QueryTx(clientCtx, string(msgRecvPacket.GetDataSignBytes()))
			if err != nil {
				return err
			}

			res := &types.QueryPacketEventsResponse{}
			return clientCtx.PrintProto(txResp.Events)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
