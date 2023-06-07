package types_test

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/gogoproto/proto"

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

func (suite *TypesTestSuite) TestCosmwasmDeserializeCosmosTx() {
	var cwBytes []byte
	var protoMessages []proto.Message
	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success: single msg from cosmwasm",
			func() {
				cwBytes = []byte{123, 34, 109, 101, 115, 115, 97, 103, 101, 115, 34, 58, 91, 123, 34, 116, 121, 112, 101, 95, 117, 114, 108, 34, 58, 34, 47, 99, 111, 115, 109, 111, 115, 46, 98, 97, 110, 107, 46, 118, 49, 98, 101, 116, 97, 49, 46, 77, 115, 103, 83, 101, 110, 100, 34, 44, 34, 118, 97, 108, 117, 101, 34, 58, 91, 49, 50, 51, 44, 51, 52, 44, 49, 48, 50, 44, 49, 49, 52, 44, 49, 49, 49, 44, 49, 48, 57, 44, 57, 53, 44, 57, 55, 44, 49, 48, 48, 44, 49, 48, 48, 44, 49, 49, 52, 44, 49, 48, 49, 44, 49, 49, 53, 44, 49, 49, 53, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 57, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 52, 57, 44, 53, 53, 44, 49, 48, 48, 44, 49, 49, 54, 44, 49, 48, 56, 44, 52, 56, 44, 49, 48, 57, 44, 49, 48, 54, 44, 49, 49, 54, 44, 53, 49, 44, 49, 49, 54, 44, 53, 53, 44, 53, 53, 44, 49, 48, 55, 44, 49, 49, 50, 44, 49, 49, 55, 44, 49, 48, 52, 44, 49, 48, 51, 44, 53, 48, 44, 49, 48, 49, 44, 49, 48, 48, 44, 49, 49, 51, 44, 49, 50, 50, 44, 49, 48, 54, 44, 49, 49, 50, 44, 49, 49, 53, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 56, 44, 49, 49, 57, 44, 49, 48, 52, 44, 49, 48, 51, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 54, 44, 53, 55, 44, 49, 48, 56, 44, 49, 48, 54, 44, 49, 49, 53, 44, 51, 52, 44, 52, 52, 44, 51, 52, 44, 49, 49, 54, 44, 49, 49, 49, 44, 57, 53, 44, 57, 55, 44, 49, 48, 48, 44, 49, 48, 48, 44, 49, 49, 52, 44, 49, 48, 49, 44, 49, 49, 53, 44, 49, 49, 53, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 57, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 52, 57, 44, 53, 53, 44, 49, 48, 48, 44, 49, 49, 54, 44, 49, 48, 56, 44, 52, 56, 44, 49, 48, 57, 44, 49, 48, 54, 44, 49, 49, 54, 44, 53, 49, 44, 49, 49, 54, 44, 53, 53, 44, 53, 53, 44, 49, 48, 55, 44, 49, 49, 50, 44, 49, 49, 55, 44, 49, 48, 52, 44, 49, 48, 51, 44, 53, 48, 44, 49, 48, 49, 44, 49, 48, 48, 44, 49, 49, 51, 44, 49, 50, 50, 44, 49, 48, 54, 44, 49, 49, 50, 44, 49, 49, 53, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 56, 44, 49, 49, 57, 44, 49, 48, 52, 44, 49, 48, 51, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 54, 44, 53, 55, 44, 49, 48, 56, 44, 49, 48, 54, 44, 49, 49, 53, 44, 51, 52, 44, 52, 52, 44, 51, 52, 44, 57, 55, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 55, 44, 49, 49, 48, 44, 49, 49, 54, 44, 51, 52, 44, 53, 56, 44, 57, 49, 44, 49, 50, 51, 44, 51, 52, 44, 49, 48, 48, 44, 49, 48, 49, 44, 49, 49, 48, 44, 49, 49, 49, 44, 49, 48, 57, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 57, 56, 44, 57, 55, 44, 49, 49, 48, 44, 57, 55, 44, 49, 49, 48, 44, 57, 55, 44, 49, 49, 53, 44, 51, 52, 44, 52, 52, 44, 51, 52, 44, 57, 55, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 55, 44, 49, 49, 48, 44, 49, 49, 54, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 52, 57, 44, 52, 56, 44, 52, 56, 44, 51, 52, 44, 49, 50, 53, 44, 57, 51, 44, 49, 50, 53, 93, 125, 93, 125}
				protoMessages = []proto.Message{
					&banktypes.MsgSend{
						FromAddress: TestOwnerAddress,
						ToAddress:   TestOwnerAddress,
						Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdk.NewInt(100))),
					},
				}
			},
			true,
		},
		{
			"success: multiple msgs, same types from cosmwasm",
			func() {
				cwBytes = []byte{123, 34, 109, 101, 115, 115, 97, 103, 101, 115, 34, 58, 91, 123, 34, 116, 121, 112, 101, 95, 117, 114, 108, 34, 58, 34, 47, 99, 111, 115, 109, 111, 115, 46, 98, 97, 110, 107, 46, 118, 49, 98, 101, 116, 97, 49, 46, 77, 115, 103, 83, 101, 110, 100, 34, 44, 34, 118, 97, 108, 117, 101, 34, 58, 91, 49, 50, 51, 44, 51, 52, 44, 49, 48, 50, 44, 49, 49, 52, 44, 49, 49, 49, 44, 49, 48, 57, 44, 57, 53, 44, 57, 55, 44, 49, 48, 48, 44, 49, 48, 48, 44, 49, 49, 52, 44, 49, 48, 49, 44, 49, 49, 53, 44, 49, 49, 53, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 57, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 52, 57, 44, 53, 53, 44, 49, 48, 48, 44, 49, 49, 54, 44, 49, 48, 56, 44, 52, 56, 44, 49, 48, 57, 44, 49, 48, 54, 44, 49, 49, 54, 44, 53, 49, 44, 49, 49, 54, 44, 53, 53, 44, 53, 53, 44, 49, 48, 55, 44, 49, 49, 50, 44, 49, 49, 55, 44, 49, 48, 52, 44, 49, 48, 51, 44, 53, 48, 44, 49, 48, 49, 44, 49, 48, 48, 44, 49, 49, 51, 44, 49, 50, 50, 44, 49, 48, 54, 44, 49, 49, 50, 44, 49, 49, 53, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 56, 44, 49, 49, 57, 44, 49, 48, 52, 44, 49, 48, 51, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 54, 44, 53, 55, 44, 49, 48, 56, 44, 49, 48, 54, 44, 49, 49, 53, 44, 51, 52, 44, 52, 52, 44, 51, 52, 44, 49, 49, 54, 44, 49, 49, 49, 44, 57, 53, 44, 57, 55, 44, 49, 48, 48, 44, 49, 48, 48, 44, 49, 49, 52, 44, 49, 48, 49, 44, 49, 49, 53, 44, 49, 49, 53, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 57, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 52, 57, 44, 53, 53, 44, 49, 48, 48, 44, 49, 49, 54, 44, 49, 48, 56, 44, 52, 56, 44, 49, 48, 57, 44, 49, 48, 54, 44, 49, 49, 54, 44, 53, 49, 44, 49, 49, 54, 44, 53, 53, 44, 53, 53, 44, 49, 48, 55, 44, 49, 49, 50, 44, 49, 49, 55, 44, 49, 48, 52, 44, 49, 48, 51, 44, 53, 48, 44, 49, 48, 49, 44, 49, 48, 48, 44, 49, 49, 51, 44, 49, 50, 50, 44, 49, 48, 54, 44, 49, 49, 50, 44, 49, 49, 53, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 56, 44, 49, 49, 57, 44, 49, 48, 52, 44, 49, 48, 51, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 54, 44, 53, 55, 44, 49, 48, 56, 44, 49, 48, 54, 44, 49, 49, 53, 44, 51, 52, 44, 52, 52, 44, 51, 52, 44, 57, 55, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 55, 44, 49, 49, 48, 44, 49, 49, 54, 44, 51, 52, 44, 53, 56, 44, 57, 49, 44, 49, 50, 51, 44, 51, 52, 44, 49, 48, 48, 44, 49, 48, 49, 44, 49, 49, 48, 44, 49, 49, 49, 44, 49, 48, 57, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 57, 56, 44, 57, 55, 44, 49, 49, 48, 44, 57, 55, 44, 49, 49, 48, 44, 57, 55, 44, 49, 49, 53, 44, 51, 52, 44, 52, 52, 44, 51, 52, 44, 57, 55, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 55, 44, 49, 49, 48, 44, 49, 49, 54, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 52, 57, 44, 52, 56, 44, 52, 56, 44, 51, 52, 44, 49, 50, 53, 44, 57, 51, 44, 49, 50, 53, 93, 125, 44, 123, 34, 116, 121, 112, 101, 95, 117, 114, 108, 34, 58, 34, 47, 99, 111, 115, 109, 111, 115, 46, 98, 97, 110, 107, 46, 118, 49, 98, 101, 116, 97, 49, 46, 77, 115, 103, 83, 101, 110, 100, 34, 44, 34, 118, 97, 108, 117, 101, 34, 58, 91, 49, 50, 51, 44, 51, 52, 44, 49, 48, 50, 44, 49, 49, 52, 44, 49, 49, 49, 44, 49, 48, 57, 44, 57, 53, 44, 57, 55, 44, 49, 48, 48, 44, 49, 48, 48, 44, 49, 49, 52, 44, 49, 48, 49, 44, 49, 49, 53, 44, 49, 49, 53, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 57, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 52, 57, 44, 53, 53, 44, 49, 48, 48, 44, 49, 49, 54, 44, 49, 48, 56, 44, 52, 56, 44, 49, 48, 57, 44, 49, 48, 54, 44, 49, 49, 54, 44, 53, 49, 44, 49, 49, 54, 44, 53, 53, 44, 53, 53, 44, 49, 48, 55, 44, 49, 49, 50, 44, 49, 49, 55, 44, 49, 48, 52, 44, 49, 48, 51, 44, 53, 48, 44, 49, 48, 49, 44, 49, 48, 48, 44, 49, 49, 51, 44, 49, 50, 50, 44, 49, 48, 54, 44, 49, 49, 50, 44, 49, 49, 53, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 56, 44, 49, 49, 57, 44, 49, 48, 52, 44, 49, 48, 51, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 54, 44, 53, 55, 44, 49, 48, 56, 44, 49, 48, 54, 44, 49, 49, 53, 44, 51, 52, 44, 52, 52, 44, 51, 52, 44, 49, 49, 54, 44, 49, 49, 49, 44, 57, 53, 44, 57, 55, 44, 49, 48, 48, 44, 49, 48, 48, 44, 49, 49, 52, 44, 49, 48, 49, 44, 49, 49, 53, 44, 49, 49, 53, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 57, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 52, 57, 44, 53, 53, 44, 49, 48, 48, 44, 49, 49, 54, 44, 49, 48, 56, 44, 52, 56, 44, 49, 48, 57, 44, 49, 48, 54, 44, 49, 49, 54, 44, 53, 49, 44, 49, 49, 54, 44, 53, 53, 44, 53, 53, 44, 49, 48, 55, 44, 49, 49, 50, 44, 49, 49, 55, 44, 49, 48, 52, 44, 49, 48, 51, 44, 53, 48, 44, 49, 48, 49, 44, 49, 48, 48, 44, 49, 49, 51, 44, 49, 50, 50, 44, 49, 48, 54, 44, 49, 49, 50, 44, 49, 49, 53, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 56, 44, 49, 49, 57, 44, 49, 48, 52, 44, 49, 48, 51, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 54, 44, 53, 55, 44, 49, 48, 56, 44, 49, 48, 54, 44, 49, 49, 53, 44, 51, 52, 44, 52, 52, 44, 51, 52, 44, 57, 55, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 55, 44, 49, 49, 48, 44, 49, 49, 54, 44, 51, 52, 44, 53, 56, 44, 57, 49, 44, 49, 50, 51, 44, 51, 52, 44, 49, 48, 48, 44, 49, 48, 49, 44, 49, 49, 48, 44, 49, 49, 49, 44, 49, 48, 57, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 57, 56, 44, 57, 55, 44, 49, 49, 48, 44, 57, 55, 44, 49, 49, 48, 44, 57, 55, 44, 49, 49, 53, 44, 51, 52, 44, 52, 52, 44, 51, 52, 44, 57, 55, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 55, 44, 49, 49, 48, 44, 49, 49, 54, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 53, 48, 44, 52, 56, 44, 52, 56, 44, 51, 52, 44, 49, 50, 53, 44, 57, 51, 44, 49, 50, 53, 93, 125, 93, 125}
				protoMessages = []proto.Message{
					&banktypes.MsgSend{
						FromAddress: TestOwnerAddress,
						ToAddress:   TestOwnerAddress,
						Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdk.NewInt(100))),
					},
					&banktypes.MsgSend{
						FromAddress: TestOwnerAddress,
						ToAddress:   TestOwnerAddress,
						Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdk.NewInt(200))),
					},
				}
			},
			true,
		},
		{
			"success: multiple msgs, different types from cosmwasm",
			func() {
				cwBytes = []byte{123, 34, 109, 101, 115, 115, 97, 103, 101, 115, 34, 58, 91, 123, 34, 116, 121, 112, 101, 95, 117, 114, 108, 34, 58, 34, 47, 99, 111, 115, 109, 111, 115, 46, 98, 97, 110, 107, 46, 118, 49, 98, 101, 116, 97, 49, 46, 77, 115, 103, 83, 101, 110, 100, 34, 44, 34, 118, 97, 108, 117, 101, 34, 58, 91, 49, 50, 51, 44, 51, 52, 44, 49, 48, 50, 44, 49, 49, 52, 44, 49, 49, 49, 44, 49, 48, 57, 44, 57, 53, 44, 57, 55, 44, 49, 48, 48, 44, 49, 48, 48, 44, 49, 49, 52, 44, 49, 48, 49, 44, 49, 49, 53, 44, 49, 49, 53, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 57, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 52, 57, 44, 53, 53, 44, 49, 48, 48, 44, 49, 49, 54, 44, 49, 48, 56, 44, 52, 56, 44, 49, 48, 57, 44, 49, 48, 54, 44, 49, 49, 54, 44, 53, 49, 44, 49, 49, 54, 44, 53, 53, 44, 53, 53, 44, 49, 48, 55, 44, 49, 49, 50, 44, 49, 49, 55, 44, 49, 48, 52, 44, 49, 48, 51, 44, 53, 48, 44, 49, 48, 49, 44, 49, 48, 48, 44, 49, 49, 51, 44, 49, 50, 50, 44, 49, 48, 54, 44, 49, 49, 50, 44, 49, 49, 53, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 56, 44, 49, 49, 57, 44, 49, 48, 52, 44, 49, 48, 51, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 54, 44, 53, 55, 44, 49, 48, 56, 44, 49, 48, 54, 44, 49, 49, 53, 44, 51, 52, 44, 52, 52, 44, 51, 52, 44, 49, 49, 54, 44, 49, 49, 49, 44, 57, 53, 44, 57, 55, 44, 49, 48, 48, 44, 49, 48, 48, 44, 49, 49, 52, 44, 49, 48, 49, 44, 49, 49, 53, 44, 49, 49, 53, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 57, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 52, 57, 44, 53, 53, 44, 49, 48, 48, 44, 49, 49, 54, 44, 49, 48, 56, 44, 52, 56, 44, 49, 48, 57, 44, 49, 48, 54, 44, 49, 49, 54, 44, 53, 49, 44, 49, 49, 54, 44, 53, 53, 44, 53, 53, 44, 49, 48, 55, 44, 49, 49, 50, 44, 49, 49, 55, 44, 49, 48, 52, 44, 49, 48, 51, 44, 53, 48, 44, 49, 48, 49, 44, 49, 48, 48, 44, 49, 49, 51, 44, 49, 50, 50, 44, 49, 48, 54, 44, 49, 49, 50, 44, 49, 49, 53, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 56, 44, 49, 49, 57, 44, 49, 48, 52, 44, 49, 48, 51, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 54, 44, 53, 55, 44, 49, 48, 56, 44, 49, 48, 54, 44, 49, 49, 53, 44, 51, 52, 44, 52, 52, 44, 51, 52, 44, 57, 55, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 55, 44, 49, 49, 48, 44, 49, 49, 54, 44, 51, 52, 44, 53, 56, 44, 57, 49, 44, 49, 50, 51, 44, 51, 52, 44, 49, 48, 48, 44, 49, 48, 49, 44, 49, 49, 48, 44, 49, 49, 49, 44, 49, 48, 57, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 57, 56, 44, 57, 55, 44, 49, 49, 48, 44, 57, 55, 44, 49, 49, 48, 44, 57, 55, 44, 49, 49, 53, 44, 51, 52, 44, 52, 52, 44, 51, 52, 44, 57, 55, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 55, 44, 49, 49, 48, 44, 49, 49, 54, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 52, 57, 44, 52, 56, 44, 52, 56, 44, 51, 52, 44, 49, 50, 53, 44, 57, 51, 44, 49, 50, 53, 93, 125, 44, 123, 34, 116, 121, 112, 101, 95, 117, 114, 108, 34, 58, 34, 47, 99, 111, 115, 109, 111, 115, 46, 115, 116, 97, 107, 105, 110, 103, 46, 118, 49, 98, 101, 116, 97, 49, 46, 77, 115, 103, 68, 101, 108, 101, 103, 97, 116, 101, 34, 44, 34, 118, 97, 108, 117, 101, 34, 58, 91, 49, 50, 51, 44, 51, 52, 44, 49, 48, 48, 44, 49, 48, 49, 44, 49, 48, 56, 44, 49, 48, 49, 44, 49, 48, 51, 44, 57, 55, 44, 49, 49, 54, 44, 49, 49, 49, 44, 49, 49, 52, 44, 57, 53, 44, 57, 55, 44, 49, 48, 48, 44, 49, 48, 48, 44, 49, 49, 52, 44, 49, 48, 49, 44, 49, 49, 53, 44, 49, 49, 53, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 57, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 52, 57, 44, 53, 53, 44, 49, 48, 48, 44, 49, 49, 54, 44, 49, 48, 56, 44, 52, 56, 44, 49, 48, 57, 44, 49, 48, 54, 44, 49, 49, 54, 44, 53, 49, 44, 49, 49, 54, 44, 53, 53, 44, 53, 53, 44, 49, 48, 55, 44, 49, 49, 50, 44, 49, 49, 55, 44, 49, 48, 52, 44, 49, 48, 51, 44, 53, 48, 44, 49, 48, 49, 44, 49, 48, 48, 44, 49, 49, 51, 44, 49, 50, 50, 44, 49, 48, 54, 44, 49, 49, 50, 44, 49, 49, 53, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 56, 44, 49, 49, 57, 44, 49, 48, 52, 44, 49, 48, 51, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 54, 44, 53, 55, 44, 49, 48, 56, 44, 49, 48, 54, 44, 49, 49, 53, 44, 51, 52, 44, 52, 52, 44, 51, 52, 44, 49, 49, 56, 44, 57, 55, 44, 49, 48, 56, 44, 49, 48, 53, 44, 49, 48, 48, 44, 57, 55, 44, 49, 49, 54, 44, 49, 49, 49, 44, 49, 49, 52, 44, 57, 53, 44, 57, 55, 44, 49, 48, 48, 44, 49, 48, 48, 44, 49, 49, 52, 44, 49, 48, 49, 44, 49, 49, 53, 44, 49, 49, 53, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 57, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 52, 57, 44, 53, 53, 44, 49, 48, 48, 44, 49, 49, 54, 44, 49, 48, 56, 44, 52, 56, 44, 49, 48, 57, 44, 49, 48, 54, 44, 49, 49, 54, 44, 53, 49, 44, 49, 49, 54, 44, 53, 53, 44, 53, 53, 44, 49, 48, 55, 44, 49, 49, 50, 44, 49, 49, 55, 44, 49, 48, 52, 44, 49, 48, 51, 44, 53, 48, 44, 49, 48, 49, 44, 49, 48, 48, 44, 49, 49, 51, 44, 49, 50, 50, 44, 49, 48, 54, 44, 49, 49, 50, 44, 49, 49, 53, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 56, 44, 49, 49, 57, 44, 49, 48, 52, 44, 49, 48, 51, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 54, 44, 53, 55, 44, 49, 48, 56, 44, 49, 48, 54, 44, 49, 49, 53, 44, 51, 52, 44, 52, 52, 44, 51, 52, 44, 57, 55, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 55, 44, 49, 49, 48, 44, 49, 49, 54, 44, 51, 52, 44, 53, 56, 44, 49, 50, 51, 44, 51, 52, 44, 49, 48, 48, 44, 49, 48, 49, 44, 49, 49, 48, 44, 49, 49, 49, 44, 49, 48, 57, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 49, 49, 53, 44, 49, 49, 54, 44, 57, 55, 44, 49, 48, 55, 44, 49, 48, 49, 44, 51, 52, 44, 52, 52, 44, 51, 52, 44, 57, 55, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 55, 44, 49, 49, 48, 44, 49, 49, 54, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 53, 51, 44, 52, 56, 44, 52, 56, 44, 52, 56, 44, 51, 52, 44, 49, 50, 53, 44, 49, 50, 53, 93, 125, 93, 125}
				protoMessages = []proto.Message{
					&banktypes.MsgSend{
						FromAddress: TestOwnerAddress,
						ToAddress:   TestOwnerAddress,
						Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdk.NewInt(100))),
					},
					&stakingtypes.MsgDelegate{
						DelegatorAddress: TestOwnerAddress,
						ValidatorAddress: TestOwnerAddress,
						Amount:           sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(5000)),
					},
				}
			},
			true,
		},
		{
			"failure: unregistered msg type from cosmwasm",
			func() {
				cwBytes = []byte{123, 34, 109, 101, 115, 115, 97, 103, 101, 115, 34, 58, 91, 123, 34, 116, 121, 112, 101, 95, 117, 114, 108, 34, 58, 34, 47, 109, 111, 99, 107, 46, 77, 115, 103, 77, 111, 99, 107, 34, 44, 34, 118, 97, 108, 117, 101, 34, 58, 91, 49, 50, 51, 44, 49, 50, 53, 93, 125, 93, 125}
				protoMessages = []proto.Message{
					&mockSdkMsg{},
				}
			},
			false,
		},
		{
			"failure: multiple unregistered msg types from cosmwasm",
			func() {
				cwBytes = []byte{123, 34, 109, 101, 115, 115, 97, 103, 101, 115, 34, 58, 91, 123, 34, 116, 121, 112, 101, 95, 117, 114, 108, 34, 58, 34, 47, 109, 111, 99, 107, 46, 77, 115, 103, 77, 111, 99, 107, 34, 44, 34, 118, 97, 108, 117, 101, 34, 58, 91, 49, 50, 51, 44, 49, 50, 53, 93, 125, 44, 123, 34, 116, 121, 112, 101, 95, 117, 114, 108, 34, 58, 34, 47, 109, 111, 99, 107, 46, 77, 115, 103, 77, 111, 99, 107, 34, 44, 34, 118, 97, 108, 117, 101, 34, 58, 91, 49, 50, 51, 44, 49, 50, 53, 93, 125, 44, 123, 34, 116, 121, 112, 101, 95, 117, 114, 108, 34, 58, 34, 47, 109, 111, 99, 107, 46, 77, 115, 103, 77, 111, 99, 107, 34, 44, 34, 118, 97, 108, 117, 101, 34, 58, 91, 49, 50, 51, 44, 49, 50, 53, 93, 125, 93, 125}
				protoMessages = []proto.Message{
					&mockSdkMsg{},
					&mockSdkMsg{},
					&mockSdkMsg{},
				}
			},
			false,
		},
		{
			"failure: empty bytes",
			func() {
				cwBytes = []byte{}
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			tc.malleate()
			msgs, errDeserialize := types.DeserializeCosmosTx(simapp.MakeTestEncodingConfig().Marshaler, cwBytes, types.EncodingJSON)
			if tc.expPass {
				suite.Require().NoError(errDeserialize, tc.name)
				for i, msg := range msgs {
					suite.Require().Equal(protoMessages[i], msg)
				}
			} else {
				suite.Require().Error(errDeserialize, tc.name)
			}
		})
	}
}

func (suite *TypesTestSuite) TestSerializeAndDeserializeCosmosTx() {
	testedEncodings := []string{types.EncodingProtobuf, types.EncodingJSON}
	var msgs []proto.Message
	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success: single msg",
			func() {
				msgs = []proto.Message{
					&banktypes.MsgSend{
						FromAddress: TestOwnerAddress,
						ToAddress:   TestOwnerAddress,
						Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdk.NewInt(100))),
					},
				}
			},
			true,
		},
		{
			"success: multiple msgs, same types",
			func() {
				msgs = []proto.Message{
					&banktypes.MsgSend{
						FromAddress: TestOwnerAddress,
						ToAddress:   TestOwnerAddress,
						Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdk.NewInt(100))),
					},
					&banktypes.MsgSend{
						FromAddress: TestOwnerAddress,
						ToAddress:   TestOwnerAddress,
						Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdk.NewInt(200))),
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
						Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdk.NewInt(100))),
					},
					&stakingtypes.MsgDelegate{
						DelegatorAddress: TestOwnerAddress,
						ValidatorAddress: TestOwnerAddress,
						Amount:           sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(5000)),
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
				suite.Require().NoError(err)

				msgs = []proto.Message{
					&govtypes.MsgSubmitProposal{
						Content:        content,
						InitialDeposit: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(5000))),
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
					Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdk.NewInt(100))),
				}
				sendAny, err := codectypes.NewAnyWithValue(sendMsg)
				suite.Require().NoError(err)

				testProposal := &govtypes.TextProposal{
					Title:       "IBC Gov Proposal",
					Description: "tokens for all!",
				}
				content, err := codectypes.NewAnyWithValue(testProposal)
				suite.Require().NoError(err)
				legacyPropMsg := &govtypes.MsgSubmitProposal{
					Content:        content,
					InitialDeposit: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(5000))),
					Proposer:       TestOwnerAddress,
				}
				legacyPropAny, err := codectypes.NewAnyWithValue(legacyPropMsg)
				suite.Require().NoError(err)

				delegateMsg := &stakingtypes.MsgDelegate{
					DelegatorAddress: TestOwnerAddress,
					ValidatorAddress: TestOwnerAddress,
					Amount:           sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(5000)),
				}
				delegateAny, err := codectypes.NewAnyWithValue(delegateMsg)
				suite.Require().NoError(err)

				messages := []*codectypes.Any{sendAny, legacyPropAny, delegateAny}

				propMsg := &govtypesv1.MsgSubmitProposal{
					Messages:       messages,
					InitialDeposit: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(5000))),
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
			},
			false,
		},
		{
			"failure: nested unregistered msg",
			func() {
				mockMsg := &mockSdkMsg{}
				mockAny, err := codectypes.NewAnyWithValue(mockMsg)
				suite.Require().NoError(err)

				msgs = []proto.Message{
					&govtypes.MsgSubmitProposal{
						Content:        mockAny,
						InitialDeposit: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(5000))),
						Proposer:       TestOwnerAddress,
					},
				}
			},
			false,
		},
		{
			"failure: nested array of unregistered msg",
			func() {
				mockMsg := &mockSdkMsg{}
				mockAny, err := codectypes.NewAnyWithValue(mockMsg)
				suite.Require().NoError(err)

				messages := []*codectypes.Any{mockAny, mockAny, mockAny}

				propMsg := &govtypesv1.MsgSubmitProposal{
					Messages:       messages,
					InitialDeposit: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(5000))),
					Proposer:       TestOwnerAddress,
					Metadata:       "",
					Title:          "New IBC Gov Proposal",
					Summary:        "more tokens for all!",
				}

				msgs = []proto.Message{propMsg}
			},
			false,
		},
	}

	for _, encoding := range testedEncodings {
		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {
				tc.malleate()

				bz, err := types.SerializeCosmosTx(simapp.MakeTestEncodingConfig().Marshaler, msgs, encoding)
				suite.Require().NoError(err, tc.name)

				msgs, err := types.DeserializeCosmosTx(simapp.MakeTestEncodingConfig().Marshaler, bz, encoding)
				if tc.expPass {
					suite.Require().NoError(err, tc.name)
				} else {
					suite.Require().Error(err, tc.name)
				}

				for i, msg := range msgs {
					suite.Require().Equal(msgs[i], msg)
				}
			})
		}

		// test serializing non sdk.Msg type
		bz, err := types.SerializeCosmosTx(simapp.MakeTestEncodingConfig().Marshaler, []proto.Message{&banktypes.MsgSendResponse{}}, encoding)
		suite.Require().NoError(err)
		suite.Require().NotEmpty(bz)

		// test deserializing unknown bytes
		_, err = types.DeserializeCosmosTx(simapp.MakeTestEncodingConfig().Marshaler, bz, encoding)
		suite.Require().Error(err) // unregistered type

		// test deserializing unknown bytes
		msgs, err := types.DeserializeCosmosTx(simapp.MakeTestEncodingConfig().Marshaler, []byte("invalid"), types.EncodingProtobuf)
		suite.Require().Error(err)
		suite.Require().Empty(msgs)
	}
}

// unregistered bytes causes amino to panic.
// test that DeserializeCosmosTx gracefully returns an error on
// unsupported amino codec.
func (suite *TypesTestSuite) TestProtoDeserializeAndSerializeCosmosTxWithAmino() {
	cdc := codec.NewLegacyAmino()
	marshaler := codec.NewAminoCodec(cdc)

	msgs, err := types.SerializeCosmosTx(marshaler, []proto.Message{&banktypes.MsgSend{}}, types.EncodingProtobuf)
	suite.Require().Error(err)
	suite.Require().Empty(msgs)

	bz, err := types.DeserializeCosmosTx(marshaler, []byte{0x10, 0}, types.EncodingProtobuf)
	suite.Require().Error(err)
	suite.Require().Empty(bz)
}

func (suite *TypesTestSuite) TestUnsupportedEncodingType() {
	// Test serialize
	msgs := []proto.Message{
		&banktypes.MsgSend{
			FromAddress: TestOwnerAddress,
			ToAddress:   TestOwnerAddress,
			Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdk.NewInt(100))),
		},
	}
	_, err := types.SerializeCosmosTx(simapp.MakeTestEncodingConfig().Marshaler, msgs, "unsupported")
	suite.Require().Error(err)

	// Test deserialize
	msgs = []proto.Message{
		&banktypes.MsgSend{
			FromAddress: TestOwnerAddress,
			ToAddress:   TestOwnerAddress,
			Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdk.NewInt(100))),
		},
	}
	data, err := types.SerializeCosmosTx(simapp.MakeTestEncodingConfig().Marshaler, msgs, types.EncodingProtobuf)
	suite.Require().NoError(err)
	_, err = types.DeserializeCosmosTx(simapp.MakeTestEncodingConfig().Marshaler, data, "unsupported")
	suite.Require().Error(err)
}
