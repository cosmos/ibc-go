package types_test

import (
	"github.com/cosmos/gogoproto/proto"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

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

// TestSerializeAndDeserializeCosmosTx tests the SerializeCosmosTx and DeserializeCosmosTx functions
// for all supported encoding types.
//
// expPass set to false means that:
// - the test case is expected to fail on deserialization for protobuf encoding.
// - the test case is expected to fail on serialization for proto3 json encoding.
func (s *TypesTestSuite) TestSerializeAndDeserializeCosmosTx() {
	testedEncodings := []string{types.EncodingProtobuf, types.EncodingProto3JSON}
	// each test case will have a corresponding expected errors in case of failures:
	expSerializeErrorStrings := make([]string, len(testedEncodings))
	expDeserializeErrorStrings := make([]string, len(testedEncodings))

	var msgs []proto.Message
	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"single msg",
			func() {
				msgs = []proto.Message{
					&banktypes.MsgSend{
						FromAddress: TestOwnerAddress,
						ToAddress:   TestOwnerAddress,
						Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdkmath.NewInt(100))),
					},
				}
			},
			true,
		},
		{
			"multiple msgs, same types",
			func() {
				msgs = []proto.Message{
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
				}
			},
			true,
		},
		{
			"success: multiple msgs, different types",
			func() {
				msgs = []proto.Message{
					&banktypes.MsgSend{
						FromAddress: TestOwnerAddress,
						ToAddress:   TestOwnerAddress,
						Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdkmath.NewInt(100))),
					},
					&stakingtypes.MsgDelegate{
						DelegatorAddress: TestOwnerAddress,
						ValidatorAddress: TestOwnerAddress,
						Amount:           sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000)),
					},
				}
			},
			true,
		},
		{
			"success: msg with nested any",
			func() {
				testProposal := &govtypes.TextProposal{
					Title:       "IBC Gov Proposal",
					Description: "tokens for all!",
				}
				content, err := codectypes.NewAnyWithValue(testProposal)
				s.Require().NoError(err)

				msgs = []proto.Message{
					&govtypes.MsgSubmitProposal{
						Content:        content,
						InitialDeposit: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000))),
						Proposer:       TestOwnerAddress,
					},
				}
			},
			true,
		},
		{
			"success: msg with nested array of any",
			func() {
				sendMsg := &banktypes.MsgSend{
					FromAddress: TestOwnerAddress,
					ToAddress:   TestOwnerAddress,
					Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdkmath.NewInt(100))),
				}
				sendAny, err := codectypes.NewAnyWithValue(sendMsg)
				s.Require().NoError(err)

				testProposal := &govtypes.TextProposal{
					Title:       "IBC Gov Proposal",
					Description: "tokens for all!",
				}
				content, err := codectypes.NewAnyWithValue(testProposal)
				s.Require().NoError(err)
				legacyPropMsg := &govtypes.MsgSubmitProposal{
					Content:        content,
					InitialDeposit: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000))),
					Proposer:       TestOwnerAddress,
				}
				legacyPropAny, err := codectypes.NewAnyWithValue(legacyPropMsg)
				s.Require().NoError(err)

				delegateMsg := &stakingtypes.MsgDelegate{
					DelegatorAddress: TestOwnerAddress,
					ValidatorAddress: TestOwnerAddress,
					Amount:           sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000)),
				}
				delegateAny, err := codectypes.NewAnyWithValue(delegateMsg)
				s.Require().NoError(err)

				messages := []*codectypes.Any{sendAny, legacyPropAny, delegateAny}

				propMsg := &govtypesv1.MsgSubmitProposal{
					Messages:       messages,
					InitialDeposit: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000))),
					Proposer:       TestOwnerAddress,
					Metadata:       "",
					Title:          "New IBC Gov Proposal",
					Summary:        "more tokens for all!",
				}

				msgs = []proto.Message{propMsg}
			},
			true,
		},
		{
			"success: empty messages",
			func() {
				msgs = []proto.Message{}
			},
			true,
		},
		{
			"failure: unregistered msg type",
			func() {
				msgs = []proto.Message{
					&mockSdkMsg{},
				}

				expSerializeErrorStrings = []string{"NO_ERROR_EXPECTED", "cannot marshal CosmosTx with proto3 json"}
				expDeserializeErrorStrings = []string{"cannot unmarshal CosmosTx with protobuf", "cannot unmarshal CosmosTx with proto3 json"}
			},
			false,
		},
		{
			"failure: multiple unregistered msg types",
			func() {
				msgs = []proto.Message{
					&mockSdkMsg{},
					&mockSdkMsg{},
					&mockSdkMsg{},
				}

				expSerializeErrorStrings = []string{"NO_ERROR_EXPECTED", "cannot marshal CosmosTx with proto3 json"}
				expDeserializeErrorStrings = []string{"cannot unmarshal CosmosTx with protobuf", "cannot unmarshal CosmosTx with proto3 json"}
			},
			false,
		},
		{
			"failure: nested unregistered msg",
			func() {
				mockMsg := &mockSdkMsg{}
				mockAny, err := codectypes.NewAnyWithValue(mockMsg)
				s.Require().NoError(err)

				msgs = []proto.Message{
					&govtypes.MsgSubmitProposal{
						Content:        mockAny,
						InitialDeposit: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000))),
						Proposer:       TestOwnerAddress,
					},
				}

				expSerializeErrorStrings = []string{"NO_ERROR_EXPECTED", "cannot marshal CosmosTx with proto3 json"}
				expDeserializeErrorStrings = []string{"cannot unmarshal CosmosTx with protobuf", "cannot unmarshal CosmosTx with proto3 json"}
			},
			false,
		},
		{
			"failure: nested array of unregistered msg",
			func() {
				mockMsg := &mockSdkMsg{}
				mockAny, err := codectypes.NewAnyWithValue(mockMsg)
				s.Require().NoError(err)

				messages := []*codectypes.Any{mockAny, mockAny, mockAny}

				propMsg := &govtypesv1.MsgSubmitProposal{
					Messages:       messages,
					InitialDeposit: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000))),
					Proposer:       TestOwnerAddress,
					Metadata:       "",
					Title:          "New IBC Gov Proposal",
					Summary:        "more tokens for all!",
				}

				msgs = []proto.Message{propMsg}

				expSerializeErrorStrings = []string{"NO_ERROR_EXPECTED", "cannot marshal CosmosTx with proto3 json"}
				expDeserializeErrorStrings = []string{"cannot unmarshal CosmosTx with protobuf", "cannot unmarshal CosmosTx with proto3 json"}
			},
			false,
		},
	}

	for i, encoding := range testedEncodings {
		for _, tc := range testCases {
			tc := tc

			s.Run(tc.name, func() {
				tc.malleate()

				bz, err := types.SerializeCosmosTx(simapp.MakeTestEncodingConfig().Codec, msgs, encoding)
				if encoding == types.EncodingProto3JSON && !tc.expPass {
					s.Require().Error(err, tc.name)
					s.Require().Contains(err.Error(), expSerializeErrorStrings[1], tc.name)
				} else {
					s.Require().NoError(err, tc.name)
				}

				deserializedMsgs, err := types.DeserializeCosmosTx(simapp.MakeTestEncodingConfig().Codec, bz, encoding)
				if tc.expPass {
					s.Require().NoError(err, tc.name)
				} else {
					s.Require().Error(err, tc.name)
					s.Require().Contains(err.Error(), expDeserializeErrorStrings[i], tc.name)
				}

				if tc.expPass {
					for i, msg := range msgs {
						// We're using proto.CompactTextString() for comparison instead of suite.Require().Equal() or proto.Equal()
						// for two main reasons:
						//
						// 1. When deserializing from JSON, the `Any` type has private fields and cached values
						//    that do not match the original message, causing equality checks to fail.
						//
						// 2. proto.Equal() does not have built-in support for comparing sdk's math.Int types.
						//
						// Using proto.CompactTextString() mitigates these issues by focusing on serialized string representation,
						// rather than internal details of the types.
						s.Require().Equal(proto.CompactTextString(msg), proto.CompactTextString(deserializedMsgs[i]))
					}
				}
			})
		}

		// test serializing non sdk.Msg type
		bz, err := types.SerializeCosmosTx(simapp.MakeTestEncodingConfig().Codec, []proto.Message{&banktypes.MsgSendResponse{}}, encoding)
		s.Require().NoError(err)
		s.Require().NotEmpty(bz)

		// test deserializing unknown bytes
		msgs, err := types.DeserializeCosmosTx(simapp.MakeTestEncodingConfig().Codec, bz, encoding)
		s.Require().Error(err) // unregistered type
		s.Require().Contains(err.Error(), expDeserializeErrorStrings[i])
		s.Require().Empty(msgs)

		// test deserializing unknown bytes
		msgs, err = types.DeserializeCosmosTx(simapp.MakeTestEncodingConfig().Codec, []byte("invalid"), encoding)
		s.Require().Error(err)
		s.Require().Contains(err.Error(), expDeserializeErrorStrings[i])
		s.Require().Empty(msgs)
	}
}

// unregistered bytes causes amino to panic.
// test that DeserializeCosmosTx gracefully returns an error on
// unsupported amino codec.
func (suite *TypesTestSuite) TestProtoDeserializeAndSerializeCosmosTxWithAmino() {
	cdc := codec.NewLegacyAmino()
	marshaler := codec.NewAminoCodec(cdc)

	msgs, err := types.SerializeCosmosTx(marshaler, []proto.Message{&banktypes.MsgSend{}}, types.EncodingProtobuf)
	suite.Require().ErrorIs(err, types.ErrInvalidCodec)
	suite.Require().Empty(msgs)

	bz, err := types.DeserializeCosmosTx(marshaler, []byte{0x10, 0}, types.EncodingProtobuf)
	suite.Require().ErrorIs(err, types.ErrInvalidCodec)
	suite.Require().Empty(bz)
}

func (suite *TypesTestSuite) TestJSONDeserializeCosmosTx() {
	testCases := []struct {
		name      string
		jsonBytes []byte
		expMsgs   []proto.Message
		expError  error
	}{
		{
			"success: single msg",
			[]byte(`{
				"messages": [
					{
						"@type": "/cosmos.bank.v1beta1.MsgSend",
						"from_address": "` + TestOwnerAddress + `",
						"to_address": "` + TestOwnerAddress + `",
						"amount": [{ "denom": "bananas", "amount": "100" }]
					}
				]
			}`),
			[]proto.Message{
				&banktypes.MsgSend{
					FromAddress: TestOwnerAddress,
					ToAddress:   TestOwnerAddress,
					Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdkmath.NewInt(100))),
				},
			},
			nil,
		},
		{
			"success: multiple msgs, same types",
			[]byte(`{
				"messages": [
					{
						"@type": "/cosmos.bank.v1beta1.MsgSend",
						"from_address": "` + TestOwnerAddress + `",
						"to_address": "` + TestOwnerAddress + `",
						"amount": [{ "denom": "bananas", "amount": "100" }]
					},
					{
						"@type": "/cosmos.bank.v1beta1.MsgSend",
						"from_address": "` + TestOwnerAddress + `",
						"to_address": "` + TestOwnerAddress + `",
						"amount": [{ "denom": "bananas", "amount": "100" }]
					}
				]
			}`),
			[]proto.Message{
				&banktypes.MsgSend{
					FromAddress: TestOwnerAddress,
					ToAddress:   TestOwnerAddress,
					Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdkmath.NewInt(100))),
				},
				&banktypes.MsgSend{
					FromAddress: TestOwnerAddress,
					ToAddress:   TestOwnerAddress,
					Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdkmath.NewInt(100))),
				},
			},
			nil,
		},
		{
			"success: multiple msgs, different types",
			[]byte(`{
				"messages": [
					{
						"@type": "/cosmos.bank.v1beta1.MsgSend",
						"from_address": "` + TestOwnerAddress + `",
						"to_address": "` + TestOwnerAddress + `",
						"amount": [{ "denom": "bananas", "amount": "100" }]
					},
					{
						"@type": "/cosmos.staking.v1beta1.MsgDelegate",
						"delegator_address": "` + TestOwnerAddress + `",
						"validator_address": "` + TestOwnerAddress + `",
						"amount": { "denom": "stake", "amount": "5000" }
					}
				]
			}`),
			[]proto.Message{
				&banktypes.MsgSend{
					FromAddress: TestOwnerAddress,
					ToAddress:   TestOwnerAddress,
					Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdkmath.NewInt(100))),
				},
				&stakingtypes.MsgDelegate{
					DelegatorAddress: TestOwnerAddress,
					ValidatorAddress: TestOwnerAddress,
					Amount:           sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000)),
				},
			},
			nil,
		},
		{
			"failure: unregistered msg type",
			[]byte(`{"messages":[{}]}`),
			[]proto.Message{
				&mockSdkMsg{},
			},
			types.ErrUnknownDataType,
		},
		{
			"failure: multiple unregistered msg types",
			[]byte(`{"messages":[{},{},{}]}`),
			[]proto.Message{
				&mockSdkMsg{},
				&mockSdkMsg{},
				&mockSdkMsg{},
			},
			types.ErrUnknownDataType,
		},
		{
			"failure: empty bytes",
			[]byte{},
			nil,
			types.ErrUnknownDataType,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			msgs, errDeserialize := types.DeserializeCosmosTx(simapp.MakeTestEncodingConfig().Codec, tc.jsonBytes, types.EncodingProto3JSON)
			if tc.expError == nil {
				suite.Require().NoError(errDeserialize, tc.name)
				for i, msg := range msgs {
					suite.Require().Equal(tc.expMsgs[i], msg)
				}
			} else {
				suite.Require().ErrorIs(errDeserialize, tc.expError, tc.name)
			}
		})
	}
}

func (suite *TypesTestSuite) TestUnsupportedEncodingType() {
	msgs := []proto.Message{
		&banktypes.MsgSend{
			FromAddress: TestOwnerAddress,
			ToAddress:   TestOwnerAddress,
			Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdkmath.NewInt(100))),
		},
	}

	bz, err := types.SerializeCosmosTx(simapp.MakeTestEncodingConfig().Codec, msgs, "unsupported")
	suite.Require().ErrorIs(err, types.ErrInvalidCodec)
	suite.Require().Nil(bz)

	data, err := types.SerializeCosmosTx(simapp.MakeTestEncodingConfig().Codec, msgs, types.EncodingProtobuf)
	suite.Require().NoError(err)

	_, err = types.DeserializeCosmosTx(simapp.MakeTestEncodingConfig().Codec, data, "unsupported")
	suite.Require().ErrorIs(err, types.ErrInvalidCodec)

	// verify that protobuf encoding still works otherwise:
	_, err = types.DeserializeCosmosTx(simapp.MakeTestEncodingConfig().Codec, data, types.EncodingProtobuf)
	suite.Require().NoError(err)
}
