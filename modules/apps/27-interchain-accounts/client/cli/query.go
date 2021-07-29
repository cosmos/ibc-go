package cli

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/cosmos/ibc-go/modules/apps/27-interchain-accounts/types"
)

func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "interchain-accounts",
		Short:                      "Querying commands for the interchain accounts module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(GetInterchainAccountCmd())

	return cmd
}

func GetInterchainAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "address [address] [connection-id]",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			ownerAddress := args[0]
			connectionId := args[1]

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.InterchainAccountAddress(context.Background(), &types.QueryInterchainAccountAddressRequest{OwnerAddress: ownerAddress, ConnectionId: connectionId})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
