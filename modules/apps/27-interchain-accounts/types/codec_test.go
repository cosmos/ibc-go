package types_test

import (
	sdkmath "cosmossdk.io/math"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/gogoproto/proto"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	"github.com/cosmos/ibc-go/v7/testing/simapp"
	"github.com/cosmos/ibc-go/v7/testing/simapp/params"
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

func (suite *TypesTestSuite) TestSerializeAndDeserializeCosmosTx() {
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

		suite.Run(tc.name, func() {
			tempApp := simapp.NewSimApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.NewAppOptionsWithFlagHome(suite.chainA.TempDir()))
			encodingConfig := params.EncodingConfig{
				InterfaceRegistry: tempApp.InterfaceRegistry(),
				Codec:             tempApp.AppCodec(),
				TxConfig:          tempApp.TxConfig(),
				Amino:             tempApp.LegacyAmino(),
			}

			bz, err := types.SerializeCosmosTx(encodingConfig.Codec, tc.msgs)
			suite.Require().NoError(err, tc.name)

			msgs, err := types.DeserializeCosmosTx(encodingConfig.Codec, bz)
			if tc.expPass {
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().Error(err, tc.name)
			}

			for i, msg := range msgs {
				suite.Require().Equal(tc.msgs[i], msg)
			}
		})
	}

	tempApp := simapp.NewSimApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.NewAppOptionsWithFlagHome(suite.chainA.TempDir()))
	encodingConfig := params.EncodingConfig{
		InterfaceRegistry: tempApp.InterfaceRegistry(),
		Codec:             tempApp.AppCodec(),
		TxConfig:          tempApp.TxConfig(),
		Amino:             tempApp.LegacyAmino(),
	}

	// test serializing non sdk.Msg type
	bz, err := types.SerializeCosmosTx(encodingConfig.Codec, []proto.Message{&banktypes.MsgSendResponse{}})
	suite.Require().NoError(err)
	suite.Require().NotEmpty(bz)

	// test deserializing unknown bytes
	_, err = types.DeserializeCosmosTx(encodingConfig.Codec, bz)
	suite.Require().Error(err) // unregistered type

	// test deserializing unknown bytes
	msgs, err := types.DeserializeCosmosTx(encodingConfig.Codec, []byte("invalid"))
	suite.Require().Error(err)
	suite.Require().Empty(msgs)
}

// unregistered bytes causes amino to panic.
// test that DeserializeCosmosTx gracefully returns an error on
// unsupported amino codec.
func (suite *TypesTestSuite) TestDeserializeAndSerializeCosmosTxWithAmino() {
	cdc := codec.NewLegacyAmino()
	marshaler := codec.NewAminoCodec(cdc)

	msgs, err := types.SerializeCosmosTx(marshaler, []proto.Message{&banktypes.MsgSend{}})
	suite.Require().Error(err)
	suite.Require().Empty(msgs)

	bz, err := types.DeserializeCosmosTx(marshaler, []byte{0x10, 0})
	suite.Require().Error(err)
	suite.Require().Empty(bz)
}
