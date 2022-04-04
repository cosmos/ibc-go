package cli

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	"github.com/spf13/cobra"
)

// GetCmdTotalRecvFees returns the command handler for the Query/TotalRecvFees rpc.
func GetCmdTotalRecvFees() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "total-recv-fees [port-id] [channel-id] [sequence]",
		Short:   "Query the total receive fees for a packet",
		Long:    "Query the total receive fees for a packet",
		Args:    cobra.ExactArgs(3),
		Example: fmt.Sprintf("%s query ibc-fee total-recv-fees transfer channel-5 100", version.AppName),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			portID, channelID := args[0], args[1]
			seq, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}

			packetID := channeltypes.NewPacketId(channelID, portID, seq)

			if err := packetID.Validate(); err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryTotalRecvFeesRequest{
				PacketId: packetID,
			}

			res, err := queryClient.TotalRecvFees(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdTotalAckFees returns the command handler for the Query/TotalAckFees rpc.
func GetCmdTotalAckFees() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "total-ack-fees [port-id] [channel-id] [sequence]",
		Short:   "Query the total acknowledgement fees for a packet",
		Long:    "Query the total acknowledgement fees for a packet",
		Args:    cobra.ExactArgs(3),
		Example: fmt.Sprintf("%s query ibc-fee total-ack-fees transfer channel-5 100", version.AppName),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			portID, channelID := args[0], args[1]
			seq, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}

			packetID := channeltypes.NewPacketId(channelID, portID, seq)

			if err := packetID.Validate(); err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryTotalAckFeesRequest{
				PacketId: packetID,
			}

			res, err := queryClient.TotalAckFees(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdTotalTimeoutFees returns the command handler for the Query/TotalTimeoutFees rpc.
func GetCmdTotalTimeoutFees() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "total-timeout-fees [port-id] [channel-id] [sequence]",
		Short:   "Query the total timeout fees for a packet",
		Long:    "Query the total timeout fees for a packet",
		Args:    cobra.ExactArgs(3),
		Example: fmt.Sprintf("%s query ibc-fee total-timeout-fees transfer channel-5 100", version.AppName),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			portID, channelID := args[0], args[1]
			seq, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}

			packetID := channeltypes.NewPacketId(channelID, portID, seq)

			if err := packetID.Validate(); err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryTotalTimeoutFeesRequest{
				PacketId: packetID,
			}

			res, err := queryClient.TotalTimeoutFees(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdCounterpartyAddress returns the command handler for the Query/CounterpartyAddress rpc.
func GetCmdCounterpartyAddress() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "counterparty-address [channel-id] [address]",
		Short:   "Query the relayer counterparty address on a given channel",
		Long:    "Query the relayer counterparty address on a given channel",
		Args:    cobra.ExactArgs(2),
		Example: fmt.Sprintf("%s query ibc-fee counterparty-address channel-5 cosmos1layxcsmyye0dc0har9sdfzwckaz8sjwlfsj8zs", version.AppName),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			if _, err := sdk.AccAddressFromBech32(args[1]); err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryCounterpartyAddressRequest{
				ChannelId:      args[0],
				RelayerAddress: args[1],
			}

			res, err := queryClient.CounterpartyAddress(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
