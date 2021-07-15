package cli

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/cosmos/ibc-go/modules/apps/ibc-account/types"
)

func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the ibc account module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(GetIBCAccountCmd())

	return cmd
}

func GetIBCAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "address [address] [connection-id]",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			address := args[0]
			connectionId := args[1]

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.IBCAccount(context.Background(), &types.QueryIBCAccountRequest{Address: address, ConnectionId: connectionId})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
