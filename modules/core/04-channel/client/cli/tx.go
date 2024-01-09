package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// NewPruneAcknowledgementsTxCmd returns the command to create a new MsgPruneAcknowledgements transaction
func NewPruneAcknowledgementsTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prune-acknowledgements [port] [channel] [limit]",
		Short: "Prune expired packet acknowledgements stored in IBC state",
		Long: `Prune expired packet acknowledgements and receipts stored in IBC state. Packet ackwnowledgements and 
		receipts are considered expired if a channel has been upgraded.`,
		Example: fmt.Sprintf("%s tx %s %s prune-acknowledgements transfer channel-0 1000", version.AppName, exported.ModuleName, types.SubModuleName),
		Args:    cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			portID, channelID := args[0], args[1]
			limit, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}

			signer := clientCtx.GetFromAddress().String()
			msg := types.NewMsgPruneAcknowledgements(portID, channelID, limit, signer)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
