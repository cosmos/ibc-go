package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// getCmdQueryChannel defines the command to query the channel information (creator and channel) for the given channel ID.
func getCmdQueryChannel() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "channel [channel-id]",
		Short:   "Query the information of a channel.",
		Long:    "Query the channel information for the provided channel ID.",
		Example: fmt.Sprintf("%s query %s %s channel [channel-id]", version.AppName, exported.ModuleName, types.SubModuleName),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			channelID := args[0]

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Channel(cmd.Context(), types.NewQueryChannelRequest(channelID))
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// getCmdQueryNextSequenceSend defines the command to query a next send sequence for a given channel
func getCmdQueryNextSequenceSend() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "next-sequence-send [channel-id]",
		Short: "Query a next send sequence",
		Long:  "Query the next sequence send for a given channel",
		Example: fmt.Sprintf(
			"%s query %s %s next-sequence-send [channel-id]", version.AppName, exported.ModuleName, types.SubModuleName,
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			channelID := args[0]
			prove, err := cmd.Flags().GetBool(flags.FlagProve)
			if err != nil {
				return err
			}

			if prove {
				res, err := queryNextSequenceSendABCI(clientCtx, channelID)
				if err != nil {
					return err
				}

				return clientCtx.PrintProto(res)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.NextSequenceSend(cmd.Context(), types.NewQueryNextSequenceSendRequest(channelID))
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Bool(flags.FlagProve, true, "show proofs for the query results")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func getCmdQueryPacketCommitment() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "packet-commitment [channel-id] [sequence]",
		Short: "Query a channel/v2 packet commitment",
		Long:  "Query a channel/v2 packet commitment by channel-id and sequence",
		Example: fmt.Sprintf(
			"%s query %s %s packet-commitment [channel-id] [sequence]", version.AppName, exported.ModuleName, types.SubModuleName,
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			channelID := args[0]
			seq, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}

			prove, err := cmd.Flags().GetBool(flags.FlagProve)
			if err != nil {
				return err
			}

			if prove {
				res, err := queryPacketCommitmentABCI(clientCtx, channelID, seq)
				if err != nil {
					return err
				}

				return clientCtx.PrintProto(res)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.PacketCommitment(cmd.Context(), types.NewQueryPacketCommitmentRequest(channelID, seq))
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Bool(flags.FlagProve, true, "show proofs for the query results")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func getCmdQueryPacketCommitments() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "packet-commitments [channel-id]",
		Short:   "Query all packet commitments associated with a channel",
		Long:    "Query all packet commitments associated with a channel",
		Example: fmt.Sprintf("%s query %s %s packet-commitments [channel-id]", version.AppName, exported.ModuleName, types.SubModuleName),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			req := &types.QueryPacketCommitmentsRequest{
				ChannelId:  args[0],
				Pagination: pageReq,
			}

			res, err := queryClient.PacketCommitments(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "packet commitments associated with a channel")

	return cmd
}

func getCmdQueryPacketAcknowledgement() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "packet-acknowledgement [channel-id] [sequence]",
		Short: "Query a channel/v2 packet acknowledgement",
		Long:  "Query a channel/v2 packet acknowledgement by channel-id and sequence",
		Example: fmt.Sprintf(
			"%s query %s %s packet-acknowledgement [channel-id] [sequence]", version.AppName, exported.ModuleName, types.SubModuleName,
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			channelID := args[0]
			seq, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}

			prove, err := cmd.Flags().GetBool(flags.FlagProve)
			if err != nil {
				return err
			}

			if prove {
				res, err := queryPacketAcknowledgementABCI(clientCtx, channelID, seq)
				if err != nil {
					return err
				}

				return clientCtx.PrintProto(res)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.PacketAcknowledgement(cmd.Context(), types.NewQueryPacketAcknowledgementRequest(channelID, seq))
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Bool(flags.FlagProve, true, "show proofs for the query results")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func getCmdQueryPacketReceipt() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "packet-receipt [channel-id] [sequence]",
		Short: "Query a channel/v2 packet receipt",
		Long:  "Query a channel/v2 packet receipt by channel-id and sequence",
		Example: fmt.Sprintf(
			"%s query %s %s packet-receipt [channel-id] [sequence]", version.AppName, exported.ModuleName, types.SubModuleName,
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			channelID := args[0]
			seq, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}

			prove, err := cmd.Flags().GetBool(flags.FlagProve)
			if err != nil {
				return err
			}

			if prove {
				res, err := queryPacketReceiptABCI(clientCtx, channelID, seq)
				if err != nil {
					return err
				}

				return clientCtx.PrintProto(res)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.PacketReceipt(cmd.Context(), types.NewQueryPacketReceiptRequest(channelID, seq))
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Bool(flags.FlagProve, true, "show proofs for the query results")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
