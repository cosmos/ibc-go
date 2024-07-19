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

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
)

const (
	flagPacketTimeoutHeight    = "packet-timeout-height"
	flagPacketTimeoutTimestamp = "packet-timeout-timestamp"
	flagAbsoluteTimeouts       = "absolute-timeouts"
	flagMemo                   = "memo"
	flagForwarding             = "forwarding"
	flagUnwind                 = "unwind"
)

// defaultRelativePacketTimeoutTimestamp is the default packet timeout timestamp (in nanoseconds)
// relative to the current block timestamp of the counterparty chain provided by the client
// state. The timeout is disabled when set to 0. The default is currently set to a 10 minute
// timeout.
var defaultRelativePacketTimeoutTimestamp = uint64((time.Duration(10) * time.Minute).Nanoseconds())

// NewTransferTxCmd returns the command to create a NewMsgTransfer transaction
func NewTransferTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer [src-port] [src-channel] [receiver] [coins]",
		Short: "Transfer one or more fungible tokens through IBC",
		Long: strings.TrimSpace(`Transfer one or more fungible tokens through IBC. Multiple tokens can be transferred in a single
packet if the coins list is a comma-separated string (e.g. 100uatom,100uosmo). Timeouts can be specified as absolute using the {absolute-timeouts} flag. 
Timeout height can be set by passing in the height string in the form {revision}-{height} using the {packet-timeout-height} flag. 
Note, relative timeout height is not supported. Relative timeout timestamp is added to the value of the user's local system clock time 
using the {packet-timeout-timestamp} flag. If no timeout value is set then a default relative timeout value of 10 minutes is used. IBC tokens
can be automatically unwound to their native chain using the {unwind} flag. Please note that if the {unwind} flag is used, then the transfer should contain only
a single token. Tokens can also be automatically forwarded through multiple chains using the {fowarding} flag and specifying
a comma-separated list of source portID/channelID pairs for each intermediary chain. {unwind} and {forwarding} flags can be used together
to first unwind IBC tokens to their native chain and then forward them to the final destination.`),
		Example: fmt.Sprintf("%s tx ibc-transfer transfer [src-port] [src-channel] [receiver] [coins]", version.AppName),
		Args:    cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			sender := clientCtx.GetFromAddress().String()
			srcPort := args[0]
			srcChannel := args[1]
			receiver := args[2]

			coins, err := sdk.ParseCoinsNormalized(args[3])
			if err != nil {
				return err
			}

			for i, coin := range coins {
				if !strings.HasPrefix(coin.Denom, "ibc/") {
					denom := types.ExtractDenomFromPath(coin.Denom)
					coins[i].Denom = denom.IBCDenom()
				}
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

			forwarding, err := parseForwarding(cmd)
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
				srcPort, srcChannel, coins, sender, receiver, timeoutHeight, timeoutTimestamp, memo, forwarding,
			)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(flagPacketTimeoutHeight, "0-0", "Packet timeout block height in the format {revision}-{height}. The timeout is disabled when set to 0-0.")
	cmd.Flags().Uint64(flagPacketTimeoutTimestamp, defaultRelativePacketTimeoutTimestamp, "Packet timeout timestamp in nanoseconds from now. Default is 10 minutes. The timeout is disabled when set to 0.")
	cmd.Flags().Bool(flagAbsoluteTimeouts, false, "Timeout flags are used as absolute timeouts.")
	cmd.Flags().String(flagMemo, "", "Memo to be sent along with the packet.")
	cmd.Flags().String(flagForwarding, "", "Forwarding information in the form of a comma separated list of portID/channelID pairs.")
	cmd.Flags().Bool(flagUnwind, false, "Flag to indicate if the coin should be unwound to its native chain before forwarding.")

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// parseForwarding parses the forwarding flag into a Forwarding object or nil if the flag is not specified. If the flag cannot
// be parsed or the hops aren't in the portID/channelID format an error is returned.
func parseForwarding(cmd *cobra.Command) (*types.Forwarding, error) {
	var hops []types.Hop

	forwardingString, err := cmd.Flags().GetString(flagForwarding)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(forwardingString) == "" {
		return nil, nil
	}

	pairs := strings.Split(forwardingString, ",")
	for _, pair := range pairs {
		pairSplit := strings.Split(pair, "/")
		if len(pairSplit) != 2 {
			return nil, fmt.Errorf("expected a portID/channelID pair, found %s", pair)
		}

		hop := types.NewHop(pairSplit[0], pairSplit[1])
		hops = append(hops, hop)
	}

	unwind, err := cmd.Flags().GetBool(flagUnwind)
	if err != nil {
		return nil, err
	}

	return types.NewForwarding(unwind, hops...), nil
}
