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

const (
	flagSequences = "sequences"
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

// getCmdQueryChannels defines the command to query all the v2 channels that this chain maintains.
func getCmdQueryChannels() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "channels",
		Short:   "Query all channels",
		Long:    "Query all channels from a chain",
		Example: fmt.Sprintf("%s query %s %s channels", version.AppName, exported.ModuleName, types.SubModuleName),
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

			req := &types.QueryChannelsRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.Channels(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "channels")

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

// getCmdQueryUnreceivedPackets defines the command to query all the unreceived
// packets on the receiving chain
func getCmdQueryUnreceivedPackets() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unreceived-packets [channel-id]",
		Short: "Query a channel/v2 unreceived-packets",
		Long:  "Query a channel/v2 unreceived-packets by channel-id and sequences",
		Example: fmt.Sprintf(
			"%s query %s %s unreceived-packet [channel-id] --sequences=1,2,3", version.AppName, exported.ModuleName, types.SubModuleName,
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			channelID := args[0]
			seqSlice, err := cmd.Flags().GetInt64Slice(flagSequences)
			if err != nil {
				return err
			}

			seqs := make([]uint64, len(seqSlice))
			for i := range seqSlice {
				seqs[i] = uint64(seqSlice[i])
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.UnreceivedPackets(cmd.Context(), types.NewQueryUnreceivedPacketsRequest(channelID, seqs))
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Int64Slice(flagSequences, []int64{}, "comma separated list of packet sequence numbers")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// getCmdQueryUnreceivedAcks defines the command to query all the unreceived acks on the original sending chain
func getCmdQueryUnreceivedAcks() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unreceived-acks [channel-id]",
		Short: "Query all the unreceived acks associated with a channel",
		Long: `Given a list of acknowledgement sequences from counterparty, determine if an ack on the counterparty chain has been received on the executing chain.

The return value represents:
- Unreceived packet acknowledgement: packet commitment exists on original sending (executing) chain and ack exists on receiving chain.
`,
		Example: fmt.Sprintf("%s query %s %s unreceived-acks [channel-id] --sequences=1,2,3", version.AppName, exported.ModuleName, types.SubModuleName),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			seqSlice, err := cmd.Flags().GetInt64Slice(flagSequences)
			if err != nil {
				return err
			}

			seqs := make([]uint64, len(seqSlice))
			for i := range seqSlice {
				seqs[i] = uint64(seqSlice[i])
			}

			req := &types.QueryUnreceivedAcksRequest{
				ChannelId:          args[0],
				PacketAckSequences: seqs,
			}

			res, err := queryClient.UnreceivedAcks(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Int64Slice(flagSequences, []int64{}, "comma separated list of packet sequence numbers")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
