package cli

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// newCreateChannelTxCmd defines the command to create an IBC channel/v2.
func newCreateChannelTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create-channel [client-identifier] [merkle-path-prefix]",
		Args:    cobra.ExactArgs(2),
		Short:   "create an IBC channel/v2",
		Long:    `Creates an IBC channel/v2 using the client identifier representing the counterparty chain and the hex-encoded merkle path prefix under which the counterparty stores packet flow information.`,
		Example: fmt.Sprintf("%s tx %s %s create-channel 07-tendermint-0 696263,657572656b61", version.AppName, exported.ModuleName, types.SubModuleName),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			clientID := args[0]
			merklePathPrefix, err := parseMerklePathPrefix(args[2])
			if err != nil {
				return err
			}

			msg := types.NewMsgCreateChannel(clientID, merklePathPrefix, clientCtx.GetFromAddress().String())

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// newRegisterCounterpartyCmd defines the command to provide the counterparty channel identifier to an IBC channel.
func newRegisterCounterpartyTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "register-counterparty [channel-identifier] [counterparty-channel-identifier]",
		Args:    cobra.ExactArgs(2),
		Short:   "Register the counterparty channel identifier for an IBC channel",
		Long:    `Register the counterparty channel identifier for an IBC channel specified by its channel ID.`,
		Example: fmt.Sprintf("%s tx %s %s register-counterparty channel-0 channel-1", version.AppName, exported.ModuleName, types.SubModuleName),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			channelID := args[0]
			counterpartyChannelID := args[1]

			msg := types.MsgRegisterCounterparty{
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

// parseMerklePathPrefix parses a comma-separated list of hex-encoded strings into a MerklePath.
func parseMerklePathPrefix(merklePathPrefixString string) (commitmenttypesv2.MerklePath, error) {
	var keyPath [][]byte
	hexPrefixes := strings.Split(merklePathPrefixString, ",")
	for _, hexPrefix := range hexPrefixes {
		prefix, err := hex.DecodeString(hexPrefix)
		if err != nil {
			return commitmenttypesv2.MerklePath{}, fmt.Errorf("invalid hex merkle path prefix: %w", err)
		}
		keyPath = append(keyPath, prefix)
	}

	return commitmenttypesv2.MerklePath{KeyPath: keyPath}, nil
}
