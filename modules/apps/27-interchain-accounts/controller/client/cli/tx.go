package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
)

const (
	// The controller chain channel version
	flagVersion = "version"
	// The channel ordering
	flagOrdering               = "ordering"
	flagPacketTimeoutTimestamp = "packet-timeout-timestamp"
)

// defaultRelativePacketTimeoutTimestamp is the default packet timeout timestamp (in nanoseconds)
// relative to the current block timestamp of the counterparty chain provided by the client
// state. The timeout is disabled when set to 0. The default is currently set to a 10 minute
// timeout.
var defaultRelativePacketTimeoutTimestamp = uint64((time.Duration(10) * time.Minute).Nanoseconds())

func newRegisterInterchainAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register [connection-id]",
		Short: "Register an interchain account on the provided connection.",
		Long: strings.TrimSpace(`Register an account on the counterparty chain via the 
connection id from the source chain. Connection identifier should be for the source chain 
and the interchain account will be created on the counterparty chain. Callers are expected to 
provide the appropriate application version string via {version} flag and the desired ordering
via the {ordering} flag. Generates a new port identifier using the provided owner string.`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			connectionID := args[0]
			owner := clientCtx.GetFromAddress().String()
			version, err := cmd.Flags().GetString(flagVersion)
			if err != nil {
				return err
			}

			order, err := parseOrdering(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgRegisterInterchainAccount(connectionID, owner, version, order)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(flagVersion, "", "Controller chain channel version")
	cmd.Flags().String(flagOrdering, channeltypes.UNORDERED.String(), fmt.Sprintf("Channel ordering, can be one of: %s", strings.Join(connectiontypes.SupportedOrderings, ", ")))
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func newSendTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-tx [connection-id] [path/to/packet_msg.json]",
		Short: "Send an interchain account tx on the provided connection.",
		Long: strings.TrimSpace(`Submits pre-built packet data containing messages to be executed on the host chain and attempts to send the packet. 
Packet data is provided as json, file or string. A relative timeout timestamp can be provided using the flag {packet-timeout-timestamp}. 
An absolute timeout is calculated on chain using the current block time. If no timeout value is set then a default relative timeout value of 10 minutes is used.`),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			cdc := codec.NewProtoCodec(clientCtx.InterfaceRegistry)

			connectionID := args[0]
			owner := clientCtx.GetFromAddress().String()

			// attempt to unmarshal ica msg data argument
			var icaMsgData icatypes.InterchainAccountPacketData
			msgContentOrFileName := args[1]
			if err := cdc.UnmarshalJSON([]byte(msgContentOrFileName), &icaMsgData); err != nil {
				// check for file path if JSON input is not provided
				contents, err := os.ReadFile(msgContentOrFileName)
				if err != nil {
					return fmt.Errorf("neither JSON input nor path to .json file for packet data with messages were provided: %w", err)
				}

				if err := cdc.UnmarshalJSON(contents, &icaMsgData); err != nil {
					return fmt.Errorf("error unmarshalling packet data with messages file: %w", err)
				}
			}

			timeoutTimestamp, err := cmd.Flags().GetUint64(flagPacketTimeoutTimestamp)
			if err != nil {
				return err
			}

			msg := types.NewMsgSendTx(owner, connectionID, timeoutTimestamp, icaMsgData)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().Uint64(flagPacketTimeoutTimestamp, defaultRelativePacketTimeoutTimestamp, "Packet timeout timestamp in nanoseconds from now. Default is 10 minutes.")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// parseOrdering gets the channel ordering from the flags.
func parseOrdering(cmd *cobra.Command) (channeltypes.Order, error) {
	orderString, err := cmd.Flags().GetString(flagOrdering)
	if err != nil {
		return channeltypes.NONE, err
	}

	order, found := channeltypes.Order_value[strings.ToUpper(orderString)]
	if !found {
		return channeltypes.NONE, fmt.Errorf("invalid channel ordering: %s", orderString)
	}

	return channeltypes.Order(order), nil
}
