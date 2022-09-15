package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"

	icatypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
)

const (
	memoFlag string = "memo"
)

func generatePacketDataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-packet-data [message]",
		Short: "Generates ICA packet data.",
		Long: fmt.Sprintf(`generate-packet-data accepts a message string and serializes it
into packet data which is outputted to stdout. It can be used in conjuction with "%s ica controller send-tx"
which submits pre-built packet data containing messages to be executed on the host chain.
`, version.AppName),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			cdc := codec.NewProtoCodec(clientCtx.InterfaceRegistry)
			memo, err := cmd.Flags().GetString(memoFlag)
			if err != nil {
				return err
			}
			packetDataBytes, err := generatePacketData(cdc, []byte(args[0]), memo)
			if err != nil {
				return err
			}
			cmd.Println(string(packetDataBytes))
			return nil
		},
	}

	cmd.Flags().String(memoFlag, "", "an optional memo to be included in the interchain account packet data")
	return cmd
}

// generatePacketData takes in message bytes and a memo and serializes the message into an
// instance of InterchainAccountPacketData which is returned as bytes.
func generatePacketData(cdc *codec.ProtoCodec, msgBytes []byte, memo string) ([]byte, error) {
	var msg sdk.Msg
	if err := cdc.UnmarshalInterfaceJSON(msgBytes, &msg); err != nil {
		return nil, err
	}

	icaPacketDataBytes, err := icatypes.SerializeCosmosTx(cdc, []sdk.Msg{msg})
	if err != nil {
		return nil, err
	}

	icaPacketData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: icaPacketDataBytes,
		Memo: memo,
	}

	if err := icaPacketData.ValidateBasic(); err != nil {
		return nil, err
	}

	return cdc.MarshalJSON(&icaPacketData)
}
