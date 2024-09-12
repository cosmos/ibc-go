package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/types"
)

// getCmdQueryClient defines the command to query the client information (creator and counterparty) for the given client ID.
func getCmdQueryClient() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "client [client-id]",
		Short:   "Query the information of a client.",
		Long:    "Query the client information (creator and counterparty) for the provided client ID.",
		Example: fmt.Sprintf("%s query %s %s client [client-id]", version.AppName, exported.ModuleName, types.SubModuleName),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			clientID := args[0]

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryClientRequest{ClientId: clientID}

			res, err := queryClient.Client(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
