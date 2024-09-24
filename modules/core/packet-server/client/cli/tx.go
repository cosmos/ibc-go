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

	commitmenttypesv2 "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/types"
)

// newProvideCounterpartyCmd defines the command to provide the counterparty to an IBC client.
func newProvideCounterpartyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provide-counterparty [client-identifier] [counterparty-client-identifier] [counterparty-merkle-path-prefix]",
		Args:  cobra.ExactArgs(3),
		Short: "provide the counterparty to an IBC client",
		Long: `Provide the counterparty to an IBC client specified by its client ID.
The [counterparty-merkle-path-prefix] is a comma-separated list of hex-encoded strings.`,
		Example: fmt.Sprintf("%s tx %s %s provide-counterparty 07-tendermint-0 07-tendermint-1 696263,657572656b61", version.AppName, exported.ModuleName, types.SubModuleName),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			clientIdentifier := args[0]
			counterpartyClientIdentifier := args[1]
			counterpartyMerklePathPrefix, err := parseMerklePathPrefix(args[2])
			if err != nil {
				return err
			}

			counterparty := types.NewCounterparty(counterpartyClientIdentifier, counterpartyMerklePathPrefix)
			msg := types.MsgProvideCounterparty{
				ChannelId:    clientIdentifier,
				Counterparty: counterparty,
				Signer:       clientCtx.GetFromAddress().String(),
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
