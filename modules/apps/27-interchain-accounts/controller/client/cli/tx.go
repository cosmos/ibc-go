package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/spf13/cobra"

	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
)

const (
	// The controller chain channel version
	flagVersion                = "version"
	flagPacketTimeoutHeight    = "packet-timeout-height"
	flagPacketTimeoutTimestamp = "packet-timeout-timestamp"
)

// NewTxCmd creates and returns the tx command
func NewTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "controller",
		Short:                      "ica controller transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		newRegisterAccountCmd(),
		newSubmitTxCmd(),
	)

	return cmd
}

func newRegisterAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register [connection-id]",
		Short: "Register an interchain account on the provided connection.",
		Long: strings.TrimSpace(`Register an account on the counterparty chain via the 
connection id from the source chain. Connection identifier should be for the source chain 
and the interchain account will be created on the counterparty chain. Callers are expected to 
provide the appropriate application version string via {version} flag. Generates a new 
port identifier using the provided owner string, binds to the port identifier and claims 
the associated capability.`),
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

			msg := types.NewMsgRegisterAccount(connectionID, owner, version)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(flagVersion, "", "Controller chain channel version")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func newSubmitTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit [connection-id] [path/to/packet_msg.json]",
		Short: "Submit an interchain account txn on the provided connection.",
		Long: strings.TrimSpace(`Submits pre-built packet data containing messages to be executed on the host chain 
from an authentication module and attempts to send the packet. Packet data is provided as json, file or string. An 
appropriate absolute timeoutTimestamp must be provided with flag {packet-timeout-timestamp}, along with a timeoutHeight
via {packet-timeout-timestamp}`),
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
			if err := cdc.UnmarshalInterfaceJSON([]byte(msgContentOrFileName), &icaMsgData); err != nil {

				// check for file path if JSON input is not provided
				contents, err := os.ReadFile(msgContentOrFileName)
				if err != nil {
					return fmt.Errorf("neither JSON input nor path to .json file for client state were provided: %w", err)
				}

				if err := cdc.UnmarshalInterfaceJSON(contents, &icaMsgData); err != nil {
					return fmt.Errorf("error unmarshalling client state file: %w", err)
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

			msg := types.NewMsgSubmitTx(owner, connectionID, timeoutHeight, timeoutTimestamp, icaMsgData)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(flagPacketTimeoutHeight, icatypes.DefaultRelativePacketTimeoutHeight, "Packet timeout block height. The timeout is disabled when set to 0-0.")
	cmd.Flags().Uint64(flagPacketTimeoutTimestamp, icatypes.DefaultRelativePacketTimeoutTimestamp, "Packet timeout timestamp in nanoseconds from now. Default is 10 minutes. The timeout is disabled when set to 0.")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
