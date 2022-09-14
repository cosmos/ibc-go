package cli

import (
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	icatypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
)

func generatePacketData() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-packet-data [message]",
		Short: "Generate ICA packet data.",
		Long:  strings.TrimSpace(`Generate ICA packet data.`), // TODO write more here
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			cdc := codec.NewProtoCodec(clientCtx.InterfaceRegistry)

			var msg sdk.Msg
			if err := cdc.UnmarshalInterfaceJSON([]byte(args[0]), &msg); err != nil {
				return err
			}

			icaPacketDataBytes, err := icatypes.SerializeCosmosTx(cdc, []sdk.Msg{msg})
			if err != nil {
				return err
			}

			memo, err := cmd.Flags().GetString("memo")
			if err != nil {
				return err
			}

			icaPacketData := icatypes.InterchainAccountPacketData{
				Type: icatypes.EXECUTE_TX,
				Data: icaPacketDataBytes,
				Memo: memo,
			}

			if err := icaPacketData.ValidateBasic(); err != nil {
				return err
			}

			jsonBytes := cdc.MustMarshalJSON(&icaPacketData)
			cmd.Println(string(jsonBytes))
			return nil
		},
	}

	cmd.Flags().String("memo", "", "")
	return cmd
}
