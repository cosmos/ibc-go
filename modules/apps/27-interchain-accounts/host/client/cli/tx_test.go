package cli

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	icatypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
)

const msgDelegateMessage = `{
  "@type": "/cosmos.staking.v1beta1.MsgDelegate",
  "delegator_address": "cosmos15ccshhmp0gsx29qpqq6g4zmltnnvgmyu9ueuadh9y2nc5zj0szls5gtddz",
  "validator_address": "cosmosvaloper1qnk2n4nlkpw9xfqntladh74w6ujtulwnmxnh3k",
  "amount": {
    "denom": "stake",
    "amount": "1000"
  }
}`

const bankSendMessage = `{
    "@type":"/cosmos.bank.v1beta1.MsgSend",
    "from_address":"cosmos15ccshhmp0gsx29qpqq6g4zmltnnvgmyu9ueuadh9y2nc5zj0szls5gtddz",
    "to_address":"cosmos10h9stc5v6ntgeygf5xf945njqq5h32r53uquvw",
    "amount": [
        {
            "denom": "stake",
            "amount": "1000"
        }
    ]
}`

func TestGeneratePacketData(t *testing.T) {
	tests := []struct {
		name                string
		memo                string
		expectedPass        bool
		message             string
		registerInterfaceFn func(registry codectypes.InterfaceRegistry)
		assertionFn         func(msg sdk.Msg)
	}{
		{
			name:                "packet data generation succeeds (MsgDelegate)",
			memo:                "non-empty-memo",
			expectedPass:        true,
			message:             msgDelegateMessage,
			registerInterfaceFn: stakingtypes.RegisterInterfaces,
			assertionFn: func(msg sdk.Msg) {
				msgDelegate := msg.(*stakingtypes.MsgDelegate)
				require.Equal(t, "cosmos15ccshhmp0gsx29qpqq6g4zmltnnvgmyu9ueuadh9y2nc5zj0szls5gtddz", msgDelegate.DelegatorAddress)
				require.Equal(t, "cosmosvaloper1qnk2n4nlkpw9xfqntladh74w6ujtulwnmxnh3k", msgDelegate.ValidatorAddress)
				require.Equal(t, "stake", msgDelegate.Amount.Denom)
				require.Equal(t, uint64(1000), msgDelegate.Amount.Amount.Uint64())
			},
		},
		{
			name:                "packet data generation succeeds (MsgSend)",
			memo:                "non-empty-memo",
			expectedPass:        true,
			message:             bankSendMessage,
			registerInterfaceFn: banktypes.RegisterInterfaces,
			assertionFn: func(msg sdk.Msg) {
				bankSendMsg := msg.(*banktypes.MsgSend)
				require.Equal(t, "cosmos15ccshhmp0gsx29qpqq6g4zmltnnvgmyu9ueuadh9y2nc5zj0szls5gtddz", bankSendMsg.FromAddress)
				require.Equal(t, "cosmos10h9stc5v6ntgeygf5xf945njqq5h32r53uquvw", bankSendMsg.ToAddress)
				require.Equal(t, "stake", bankSendMsg.Amount.GetDenomByIndex(0))
				require.Equal(t, uint64(1000), bankSendMsg.Amount[0].Amount.Uint64())
			},
		},
		{
			name:                "empty memo is valid",
			memo:                "",
			expectedPass:        true,
			message:             msgDelegateMessage,
			registerInterfaceFn: stakingtypes.RegisterInterfaces,
			assertionFn: func(msg sdk.Msg) {
				msgDelegate := msg.(*stakingtypes.MsgDelegate)
				require.Equal(t, "cosmos15ccshhmp0gsx29qpqq6g4zmltnnvgmyu9ueuadh9y2nc5zj0szls5gtddz", msgDelegate.DelegatorAddress)
				require.Equal(t, "cosmosvaloper1qnk2n4nlkpw9xfqntladh74w6ujtulwnmxnh3k", msgDelegate.ValidatorAddress)
				require.Equal(t, "stake", msgDelegate.Amount.Denom)
				require.Equal(t, uint64(1000), msgDelegate.Amount.Amount.Uint64())
			},
		},
		{
			name:                "invalid message string",
			memo:                "",
			expectedPass:        false,
			message:             "{}",
			registerInterfaceFn: func(codectypes.InterfaceRegistry) {},
			assertionFn:         func(sdk.Msg) {},
		},
	}

	for _, tt := range tests {
		tt := tt
		ir := codectypes.NewInterfaceRegistry()
		tt.registerInterfaceFn(ir)

		cdc := codec.NewProtoCodec(ir)
		t.Run(tt.name, func(t *testing.T) {
			bz, err := generatePacketData(cdc, []byte(tt.message), tt.memo)
			if !tt.expectedPass {
				require.Error(t, err)
				require.Nil(t, bz)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, bz)

			packetData := icatypes.InterchainAccountPacketData{}
			err = cdc.UnmarshalJSON(bz, &packetData)
			require.NoError(t, err)

			require.NoError(t, err)
			require.Equal(t, icatypes.EXECUTE_TX, packetData.Type)
			require.Equal(t, tt.memo, packetData.Memo)

			data := packetData.Data
			messages, err := icatypes.DeserializeCosmosTx(cdc, data)

			require.NoError(t, err)
			require.NotNil(t, messages)

			tt.assertionFn(messages[0])
		})
	}
}
