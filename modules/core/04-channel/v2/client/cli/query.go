package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

const (
	flagSequences = "sequences"
)

// getCmdQueryNextSequenceSend defines the command to query a next send sequence for a given client
func getCmdQueryNextSequenceSend() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "next-sequence-send [client-id]",
		Short: "Query a next send sequence",
		Long:  "Query the next sequence send for a given client",
		Example: fmt.Sprintf(
			"%s query %s %s next-sequence-send [client-id]", version.AppName, exported.ModuleName, types.SubModuleName,
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			clientID := args[0]
			prove, err := cmd.Flags().GetBool(flags.FlagProve)
			if err != nil {
				return err
			}

			if prove {
				res, err := queryNextSequenceSendABCI(clientCtx, clientID)
				if err != nil {
					return err
				}

				return clientCtx.PrintProto(res)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.NextSequenceSend(cmd.Context(), types.NewQueryNextSequenceSendRequest(clientID))
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
		Use:   "packet-commitment [client-id] [sequence]",
		Short: "Query a channel/v2 packet commitment",
		Long:  "Query a channel/v2 packet commitment by client-id and sequence",
		Example: fmt.Sprintf(
			"%s query %s %s packet-commitment [client-id] [sequence]", version.AppName, exported.ModuleName, types.SubModuleName,
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			clientID := args[0]
			seq, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}

			prove, err := cmd.Flags().GetBool(flags.FlagProve)
			if err != nil {
				return err
			}

			if prove {
				res, err := queryPacketCommitmentABCI(clientCtx, clientID, seq)
				if err != nil {
					return err
				}

				return clientCtx.PrintProto(res)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.PacketCommitment(cmd.Context(), types.NewQueryPacketCommitmentRequest(clientID, seq))
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
		Use:     "packet-commitments [client-id]",
		Short:   "Query all packet commitments associated with a client",
		Long:    "Query all packet commitments associated with a client",
		Example: fmt.Sprintf("%s query %s %s packet-commitments [client-id]", version.AppName, exported.ModuleName, types.SubModuleName),
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
				ClientId:   args[0],
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
	flags.AddPaginationFlagsToCmd(cmd, "packet commitments associated with a client")

	return cmd
}

func getCmdQueryPacketAcknowledgement() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "packet-acknowledgement [client-id] [sequence]",
		Short: "Query a channel/v2 packet acknowledgement",
		Long:  "Query a channel/v2 packet acknowledgement by client-id and sequence",
		Example: fmt.Sprintf(
			"%s query %s %s packet-acknowledgement [client-id] [sequence]", version.AppName, exported.ModuleName, types.SubModuleName,
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			clientID := args[0]
			seq, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}

			prove, err := cmd.Flags().GetBool(flags.FlagProve)
			if err != nil {
				return err
			}

			if prove {
				res, err := queryPacketAcknowledgementABCI(clientCtx, clientID, seq)
				if err != nil {
					return err
				}

				return clientCtx.PrintProto(res)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.PacketAcknowledgement(cmd.Context(), types.NewQueryPacketAcknowledgementRequest(clientID, seq))
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
		Use:   "packet-receipt [client-id] [sequence]",
		Short: "Query a channel/v2 packet receipt",
		Long:  "Query a channel/v2 packet receipt by client-id and sequence",
		Example: fmt.Sprintf(
			"%s query %s %s packet-receipt [client-id] [sequence]", version.AppName, exported.ModuleName, types.SubModuleName,
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			clientID := args[0]
			seq, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}

			prove, err := cmd.Flags().GetBool(flags.FlagProve)
			if err != nil {
				return err
			}

			if prove {
				res, err := queryPacketReceiptABCI(clientCtx, clientID, seq)
				if err != nil {
					return err
				}

				return clientCtx.PrintProto(res)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.PacketReceipt(cmd.Context(), types.NewQueryPacketReceiptRequest(clientID, seq))
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
		Use:   "unreceived-packets [client-id]",
		Short: "Query a channel/v2 unreceived-packets",
		Long:  "Query a channel/v2 unreceived-packets by client-id and sequences",
		Example: fmt.Sprintf(
			"%s query %s %s unreceived-packet [client-id] --sequences=1,2,3", version.AppName, exported.ModuleName, types.SubModuleName,
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			clientID := args[0]
			seqSlice, err := cmd.Flags().GetInt64Slice(flagSequences)
			if err != nil {
				return err
			}

			seqs := make([]uint64, len(seqSlice))
			for i := range seqSlice {
				seqs[i] = uint64(seqSlice[i])
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.UnreceivedPackets(cmd.Context(), types.NewQueryUnreceivedPacketsRequest(clientID, seqs))
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
		Use:   "unreceived-acks [client-id]",
		Short: "Query all the unreceived acks associated with a client",
		Long: `Given a list of acknowledgement sequences from counterparty, determine if an ack on the counterparty chain has been received on the executing chain.

The return value represents:
- Unreceived packet acknowledgement: packet commitment exists on original sending (executing) chain and ack exists on receiving chain.
`,
		Example: fmt.Sprintf("%s query %s %s unreceived-acks [client-id] --sequences=1,2,3", version.AppName, exported.ModuleName, types.SubModuleName),
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
				ClientId:           args[0],
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
