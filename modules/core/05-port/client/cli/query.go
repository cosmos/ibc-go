package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"

	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/modules/core/05-port/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
)

// GetCmdQueryPorts defines the command to query a port
func GetCmdQueryPort() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "port [port-id] [counterparty-port-id] [counterparty-channel-id] [counterparty-version]",
		Short:   "Query an IBC port",
		Long:    "Query an IBC port by providing it's port ID and associated counterparty port and channel identifiers",
		Example: fmt.Sprintf("%s query %s %s port", version.AppName, host.ModuleName, types.SubModuleName),
		Args:    cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryPortRequest{
				PortId: args[0],
				Counterparty: &channeltypes.Counterparty{
					PortId:    args[1],
					ChannelId: args[2],
				},
				CounterpartyVersion: args[3],
			}

			portRes, err := queryClient.Port(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(portRes)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
