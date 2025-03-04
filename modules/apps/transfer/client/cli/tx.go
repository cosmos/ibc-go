package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
)

const (
	flagPacketTimeoutHeight    = "packet-timeout-height"
	flagPacketTimeoutTimestamp = "packet-timeout-timestamp"
	flagAbsoluteTimeouts       = "absolute-timeouts"
	flagMemo                   = "memo"
)

// defaultRelativePacketTimeoutTimestamp is the default packet timeout timestamp (in nanoseconds)
// relative to the current block timestamp of the counterparty chain provided by the client
// state. For IBC v1 protocol, either timeout timestamp or timeout height must be set.
// If you are sending with IBC v2 protocol, timeout timestamp must be set.
// The default is currently set to a 10 minute timeout.
var defaultRelativePacketTimeoutTimestamp = uint64((time.Duration(10) * time.Minute).Nanoseconds())

// NewTransferTxCmd returns the command to create a NewMsgTransfer transaction
func NewTransferTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer [src-port] [src-channel] [receiver] [coin]",
		Short: "Transfer a fungible tokens through IBC",
		Long: strings.TrimSpace(`Transfer one fungible tokens through IBC. Timeouts can be specified as absolute using the {absolute-timeouts} flag. 
Timeout height can be set by passing in the height string in the form {revision}-{height} using the {packet-timeout-height} flag. 
Note, relative timeout height is not supported. Relative timeout timestamp is added to the value of the user's local system clock time 
using the {packet-timeout-timestamp} flag. If no timeout value is set then a default relative timeout value of 10 minutes is used.`),
		Example: fmt.Sprintf("%s tx ibc-transfer transfer [src-port] [src-channel] [receiver] [coin]", version.AppName),
		Args:    cobra.RangeArgs(2, 4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			sender := clientCtx.GetFromAddress().String()

			srcPort := args[0]
			srcChannel := args[1]
			receiver := args[2]

			coin, err := sdk.ParseCoinNormalized(args[3])
			if err != nil {
				return err
			}

			if !strings.HasPrefix(coin.Denom, "ibc/") {
				denom := types.ExtractDenomFromPath(coin.Denom)
				coin.Denom = denom.IBCDenom()
			}

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

			memo, err := cmd.Flags().GetString(flagMemo)
			if err != nil {
				return err
			}

			// NOTE: relative timeouts using block height are not supported.
			// if the timeouts are not absolute, CLI users rely solely on local clock time in order to calculate relative timestamps.
			if !absoluteTimeouts {
				if !timeoutHeight.IsZero() {
					return errors.New("relative timeouts using block height is not supported")
				}

				if timeoutTimestamp == 0 {
					return errors.New("relative timeouts must provide a non zero value timestamp")
				}

				// use local clock time as reference time for calculating timeout timestamp.
				now := time.Now().UnixNano()
				if now <= 0 {
					return errors.New("local clock time is not greater than Jan 1st, 1970 12:00 AM")
				}

				timeoutTimestamp = uint64(now) + timeoutTimestamp
			}

			msg := types.NewMsgTransfer(
				srcPort, srcChannel, coin, sender, receiver, timeoutHeight, timeoutTimestamp, memo,
			)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(flagPacketTimeoutHeight, "0-0", "Packet timeout block height in the format {revision}-{height}.")
	cmd.Flags().Uint64(flagPacketTimeoutTimestamp, defaultRelativePacketTimeoutTimestamp, "Packet timeout timestamp in nanoseconds from now. Default is 10 minutes. On IBC v1 protocol, either timeout timestamp or timeout height must be set. On IBC v2 protocol timeout timestamp must be set.")
	cmd.Flags().Bool(flagAbsoluteTimeouts, false, "Timeout flags are used as absolute timeouts.")
	cmd.Flags().String(flagMemo, "", "Memo to be sent along with the packet.")

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
