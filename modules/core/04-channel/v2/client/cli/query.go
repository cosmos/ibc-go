package cli

import (
	"fmt"

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
		Long:    "Query the channel information (creator and channel) for the provided channel ID.",
		Example: fmt.Sprintf("%s query %s %s channel [channel-id]", version.AppName, exported.ModuleName, types.SubModuleName),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			channelID := args[0]

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryChannelRequest{ChannelId: channelID}

			res, err := queryClient.Channel(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func getCmdQueryPacketCommitment() *cobra.Command {
	return nil
}
