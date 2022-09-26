package cli

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	icatypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/types"
)

const (
	memoFlag string = "memo"
)

func generatePacketDataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-packet-data [message]",
		Short: "Generates ICA packet data.",
		Long: `generate-packet-data accepts a message string and serializes it
into packet data which is outputted to stdout. It can be used in conjunction with send-tx"
which submits pre-built packet data containing messages to be executed on the host chain.
`,
		Example: `<binary> tx interchain-accounts host generate-packet-data '{
    "@type":"/cosmos.bank.v1beta1.MsgSend",
    "from_address":"cosmos15ccshhmp0gsx29qpqq6g4zmltnnvgmyu9ueuadh9y2nc5zj0szls5gtddz",
    "to_address":"cosmos10h9stc5v6ntgeygf5xf945njqq5h32r53uquvw",
    "amount": [
        {
            "denom": "stake",
            "amount": "1000"
        }
    ]
}' --memo memo


<binary> tx interchain-accounts host generate-packet-data '[{
    "@type":"/cosmos.bank.v1beta1.MsgSend",
    "from_address":"cosmos15ccshhmp0gsx29qpqq6g4zmltnnvgmyu9ueuadh9y2nc5zj0szls5gtddz",
    "to_address":"cosmos10h9stc5v6ntgeygf5xf945njqq5h32r53uquvw",
    "amount": [
        {
            "denom": "stake",
            "amount": "1000"
        }
    ]
},
{
	"@type": "/cosmos.staking.v1beta1.MsgDelegate",
	"delegator_address": "cosmos15ccshhmp0gsx29qpqq6g4zmltnnvgmyu9ueuadh9y2nc5zj0szls5gtddz",
	"validator_address": "cosmosvaloper1qnk2n4nlkpw9xfqntladh74w6ujtulwnmxnh3k",
	"amount": {
		"denom": "stake",
		"amount": "1000"
	}
}]'`,
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
	sdkMessages, err := convertBytesIntoSdkMessages(cdc, msgBytes)
	if err != nil {
		return nil, err
	}

	return generateIcaPacketDataFromSdkMessages(cdc, sdkMessages, memo)
}

// convertBytesIntoSdkMessages returns a list of sdk messages from bytes. The bytes can be in the form of a single
// message, or a json array of messages.
func convertBytesIntoSdkMessages(cdc *codec.ProtoCodec, msgBytes []byte) ([]sdk.Msg, error) {
	var rawMessages []json.RawMessage
	if err := json.Unmarshal(msgBytes, &rawMessages); err != nil {
		// if we fail to unmarshal a list of messages, we assume we are just dealing with a single message.
		// in this case we return a list of a single item.
		var msg sdk.Msg
		if err := cdc.UnmarshalInterfaceJSON(msgBytes, &msg); err != nil {
			return nil, err
		}

		return []sdk.Msg{msg}, nil
	}

	sdkMessages := make([]sdk.Msg, len(rawMessages))
	for i, anyJSON := range rawMessages {
		var msg sdk.Msg
		if err := cdc.UnmarshalInterfaceJSON(anyJSON, &msg); err != nil {
			return nil, err
		}

		sdkMessages[i] = msg
	}

	return sdkMessages, nil
}

// generateIcaPacketDataFromSdkMessages generates ica packet data as bytes from a given set of sdk messages and a memo.
func generateIcaPacketDataFromSdkMessages(cdc *codec.ProtoCodec, sdkMessages []sdk.Msg, memo string) ([]byte, error) {
	icaPacketDataBytes, err := icatypes.SerializeCosmosTx(cdc, sdkMessages)
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
