package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/types"
)

// newProvideCounterpartyCmd defines the command to provide the counterparty to an IBC channel.
func newProvideCounterpartyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "provide-counterparty [channel-identifier] [counterparty-channel-identifier]",
		Args:    cobra.ExactArgs(2),
		Short:   "provide the counterparty channel id to an IBC channel end",
		Long:    `Provide the counterparty channel id to an IBC channel end specified by its channel ID.`,
		Example: fmt.Sprintf("%s tx %s %s provide-counterparty channel-0 channel-1", version.AppName, exported.ModuleName, types.SubModuleName),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			channelID := args[0]
			counterpartyChannelID := args[1]

			msg := types.MsgProvideCounterparty{
				ChannelId:             channelID,
				CounterpartyChannelId: counterpartyChannelID,
				Signer:                clientCtx.GetFromAddress().String(),
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
