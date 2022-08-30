package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/ibc-go/v5/modules/apps/icq/types"
	transfertypes "github.com/cosmos/ibc-go/v5/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
	channelutils "github.com/cosmos/ibc-go/v5/modules/core/04-channel/client/utils"
)

const (
	flagPacketTimeoutHeight    = "packet-timeout-height"
	flagPacketTimeoutTimestamp = "packet-timeout-timestamp"
	flagAbsoluteTimeouts       = "absolute-timeouts"
)

// NewQueryTxCmd returns the command to create a NewMsgQuery transaction
func NewQueryTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		// TODO: handle > 1 request input
		Use:   "query [src-port] [src-channel] [request]",
		Short: "Issue an interchain query through IBC",
		Long: strings.TrimSpace(`Issue an interchain query through IBC. Timeouts can be specified
as absolute or relative using the "absolute-timeouts" flag. Timeout height can be set by passing in the height string
in the form {revision}-{height} using the "packet-timeout-height" flag. Relative timeout height is added to the block
height queried from the latest consensus state corresponding to the counterparty channel. Relative timeout timestamp 
is added to the greater value of the local clock time and the block timestamp queried from the latest consensus state 
corresponding to the counterparty channel. Any timeout set to 0 is disabled.`),
		Example: fmt.Sprintf("%s tx ibc-icq query [src-port] [src-channel] [request]", version.AppName),
		Args:    cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			signer := clientCtx.GetFromAddress().String()
			srcPort := args[0]
			srcChannel := args[1]
			requestData := args[2]

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

			absoluteTimeouts, err := cmd.Flags().GetBool(flagAbsoluteTimeouts)
			if err != nil {
				return err
			}

			// if the timeouts are not absolute, retrieve latest block height and block timestamp
			// for the consensus state connected to the destination port/channel
			if !absoluteTimeouts {
				consensusState, height, _, err := channelutils.QueryLatestConsensusState(clientCtx, srcPort, srcChannel)
				if err != nil {
					return err
				}

				if !timeoutHeight.IsZero() {
					absoluteHeight := height
					absoluteHeight.RevisionNumber += timeoutHeight.RevisionNumber
					absoluteHeight.RevisionHeight += timeoutHeight.RevisionHeight
					timeoutHeight = absoluteHeight
				}

				if timeoutTimestamp != 0 {
					// use local clock time as reference time if it is later than the
					// consensus state timestamp of the counter party chain, otherwise
					// still use consensus state timestamp as reference
					now := time.Now().UnixNano()
					consensusStateTimestamp := consensusState.GetTimestamp()
					if now > 0 {
						now := uint64(now)
						if now > consensusStateTimestamp {
							timeoutTimestamp = now + timeoutTimestamp
						} else {
							timeoutTimestamp = consensusStateTimestamp + timeoutTimestamp
						}
					} else {
						return errors.New("local clock time is not greater than Jan 1st, 1970 12:00 AM")
					}
				}
			}

			var request abci.RequestQuery
			if err := json.Unmarshal([]byte(requestData), &request); err != nil {
				return err
			}
			msg := types.NewMsgQuery(
				srcPort, srcChannel, []abci.RequestQuery{request}, timeoutHeight, timeoutTimestamp, signer,
			)
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(flagPacketTimeoutHeight, transfertypes.DefaultRelativePacketTimeoutHeight, "Packet timeout block height. The timeout is disabled when set to 0-0.")
	cmd.Flags().Uint64(flagPacketTimeoutTimestamp, transfertypes.DefaultRelativePacketTimeoutTimestamp, "Packet timeout timestamp in nanoseconds from now. Default is 10 minutes. The timeout is disabled when set to 0.")
	cmd.Flags().Bool(flagAbsoluteTimeouts, false, "Timeout flags are used as absolute timeouts.")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
