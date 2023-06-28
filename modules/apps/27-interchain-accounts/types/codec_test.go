package types_test

import (
	"github.com/cosmos/gogoproto/proto"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	"github.com/cosmos/ibc-go/v7/testing/simapp"
)

// mockSdkMsg defines a mock struct, used for testing codec error scenarios
type mockSdkMsg struct{}

// Reset implements sdk.Msg
func (mockSdkMsg) Reset() {
}

// String implements sdk.Msg
func (mockSdkMsg) String() string {
	return ""
}

// ProtoMessage implements sdk.Msg
func (mockSdkMsg) ProtoMessage() {
}

// ValidateBasic implements sdk.Msg
func (mockSdkMsg) ValidateBasic() error {
	return nil
}

// GetSigners implements sdk.Msg
func (mockSdkMsg) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{}
}

func (s *TypesTestSuite) TestSerializeAndDeserializeCosmosTx() {
	testCases := []struct {
		name    string
		msgs    []proto.Message
		expPass bool
	}{
		{
			"single msg",
			[]proto.Message{
				&banktypes.MsgSend{
					FromAddress: TestOwnerAddress,
					ToAddress:   TestOwnerAddress,
					Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdkmath.NewInt(100))),
				},
			},
			true,
		},
		{
			"multiple msgs, same types",
			[]proto.Message{
				&banktypes.MsgSend{
					FromAddress: TestOwnerAddress,
					ToAddress:   TestOwnerAddress,
					Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdkmath.NewInt(100))),
				},
				&banktypes.MsgSend{
					FromAddress: TestOwnerAddress,
					ToAddress:   TestOwnerAddress,
					Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdkmath.NewInt(200))),
				},
			},
			true,
		},
		{
			"multiple msgs, different types",
			[]proto.Message{
				&banktypes.MsgSend{
					FromAddress: TestOwnerAddress,
					ToAddress:   TestOwnerAddress,
					Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdkmath.NewInt(100))),
				},
				&govtypes.MsgSubmitProposal{
					InitialDeposit: sdk.NewCoins(sdk.NewCoin("bananas", sdkmath.NewInt(100))),
					Proposer:       TestOwnerAddress,
				},
			},
			true,
		},
		{
			"unregistered msg type",
			[]proto.Message{
				&mockSdkMsg{},
			},
			false,
		},
		{
			"multiple unregistered msg types",
			[]proto.Message{
				&mockSdkMsg{},
				&mockSdkMsg{},
				&mockSdkMsg{},
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			bz, err := types.SerializeCosmosTx(simapp.MakeTestEncodingConfig().Codec, tc.msgs, types.EncodingProtobuf)
			s.Require().NoError(err, tc.name)

			msgs, err := types.DeserializeCosmosTx(simapp.MakeTestEncodingConfig().Codec, bz, types.EncodingProtobuf)
			if tc.expPass {
				s.Require().NoError(err, tc.name)
			} else {
				s.Require().Error(err, tc.name)
			}

			for i, msg := range msgs {
				s.Require().Equal(tc.msgs[i], msg)
			}
		})
	}

	// test serializing non sdk.Msg type
	bz, err := types.SerializeCosmosTx(simapp.MakeTestEncodingConfig().Codec, []proto.Message{&banktypes.MsgSendResponse{}}, types.EncodingProtobuf)
	s.Require().NoError(err)
	s.Require().NotEmpty(bz)

	// test deserializing unknown bytes
	_, err = types.DeserializeCosmosTx(simapp.MakeTestEncodingConfig().Codec, bz, types.EncodingProtobuf)
	s.Require().Error(err) // unregistered type

	// test deserializing unknown bytes
	msgs, err := types.DeserializeCosmosTx(simapp.MakeTestEncodingConfig().Codec, []byte("invalid"), types.EncodingProtobuf)
	s.Require().Error(err)
	s.Require().Empty(msgs)
}

// unregistered bytes causes amino to panic.
// test that DeserializeCosmosTx gracefully returns an error on
// unsupported amino codec.
func (s *TypesTestSuite) TestDeserializeAndSerializeCosmosTxWithAmino() {
	cdc := codec.NewLegacyAmino()
	marshaler := codec.NewAminoCodec(cdc)

	msgs, err := types.SerializeCosmosTx(marshaler, []proto.Message{&banktypes.MsgSend{}}, types.EncodingProtobuf)
	s.Require().Error(err)
	s.Require().Empty(msgs)

	bz, err := types.DeserializeCosmosTx(marshaler, []byte{0x10, 0}, types.EncodingProtobuf)
	s.Require().Error(err)
	s.Require().Empty(bz)
}
