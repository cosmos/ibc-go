package cli

import (
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/cosmos/ibc-go/v4/modules/apps/31-ibc-query/client/utils"
	"github.com/cosmos/ibc-go/v4/modules/apps/31-ibc-query/types"
	clienttypes "github.com/cosmos/ibc-go/v4/modules/core/02-client/types"
)

const (
	flagPacketTimeoutHeight    = "packet-timeout-height"
	flagPacketTimeoutTimestamp = "packet-timeout-timestamp"
	flagAbsoluteTimeouts       = "absolute-timeouts"
)

func NewMsgCrossChainQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cross-chain-query [client-id] [query-path]",
		Short:   "Request ibc query on a given channel.",
		Long:    strings.TrimSpace(`Register a payee address on a given channel.`),
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			creator := clientCtx.GetFromAddress().String()
			queryId := utils.GetQueryIdentifier()

			clientId := args[0]
			path := args[1]
			
			//TODO
			// Get chain height from queried chain
			temporaryQueryHeight := uint64(123) 

			timeoutHeightStr, err := cmd.Flags().GetString(flagPacketTimeoutHeight)
			if err != nil {
				return err
			}
			timeoutHeight, err := clienttypes.ParseHeight(timeoutHeightStr)
			if err != nil {
				return err
			}

			timeoutTimestamp, err := cmd.Flags().GetUint64(flagPacketTimeoutTimestamp)
			if err != nil {
				return err
			}
			
			msg := types.NewMsgSubmitCrossChainQuery(queryId, path, timeoutHeight.RevisionHeight, timeoutTimestamp, temporaryQueryHeight, clientId, creator)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
