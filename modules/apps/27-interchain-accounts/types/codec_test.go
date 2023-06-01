package types_test

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
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
				cwBytes = []byte{123, 34, 109, 101, 115, 115, 97, 103, 101, 115, 34, 58, 91, 123, 34, 116, 121, 112, 101, 95, 117, 114, 108, 34, 58, 34, 47, 99, 111, 115, 109, 111, 115, 46, 98, 97, 110, 107, 46, 118, 49, 98, 101, 116, 97, 49, 46, 77, 115, 103, 83, 101, 110, 100, 34, 44, 34, 118, 97, 108, 117, 101, 34, 58, 91, 49, 50, 51, 44, 51, 52, 44, 49, 48, 50, 44, 49, 49, 52, 44, 49, 49, 49, 44, 49, 48, 57, 44, 57, 53, 44, 57, 55, 44, 49, 48, 48, 44, 49, 48, 48, 44, 49, 49, 52, 44, 49, 48, 49, 44, 49, 49, 53, 44, 49, 49, 53, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 57, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 52, 57, 44, 53, 53, 44, 49, 48, 48, 44, 49, 49, 54, 44, 49, 48, 56, 44, 52, 56, 44, 49, 48, 57, 44, 49, 48, 54, 44, 49, 49, 54, 44, 53, 49, 44, 49, 49, 54, 44, 53, 53, 44, 53, 53, 44, 49, 48, 55, 44, 49, 49, 50, 44, 49, 49, 55, 44, 49, 48, 52, 44, 49, 48, 51, 44, 53, 48, 44, 49, 48, 49, 44, 49, 48, 48, 44, 49, 49, 51, 44, 49, 50, 50, 44, 49, 48, 54, 44, 49, 49, 50, 44, 49, 49, 53, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 56, 44, 49, 49, 57, 44, 49, 48, 52, 44, 49, 48, 51, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 54, 44, 53, 55, 44, 49, 48, 56, 44, 49, 48, 54, 44, 49, 49, 53, 44, 51, 52, 44, 52, 52, 44, 51, 52, 44, 49, 49, 54, 44, 49, 49, 49, 44, 57, 53, 44, 57, 55, 44, 49, 48, 48, 44, 49, 48, 48, 44, 49, 49, 52, 44, 49, 48, 49, 44, 49, 49, 53, 44, 49, 49, 53, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 57, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 52, 57, 44, 53, 53, 44, 49, 48, 48, 44, 49, 49, 54, 44, 49, 48, 56, 44, 52, 56, 44, 49, 48, 57, 44, 49, 48, 54, 44, 49, 49, 54, 44, 53, 49, 44, 49, 49, 54, 44, 53, 53, 44, 53, 53, 44, 49, 48, 55, 44, 49, 49, 50, 44, 49, 49, 55, 44, 49, 48, 52, 44, 49, 48, 51, 44, 53, 48, 44, 49, 48, 49, 44, 49, 48, 48, 44, 49, 49, 51, 44, 49, 50, 50, 44, 49, 48, 54, 44, 49, 49, 50, 44, 49, 49, 53, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 56, 44, 49, 49, 57, 44, 49, 48, 52, 44, 49, 48, 51, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 54, 44, 53, 55, 44, 49, 48, 56, 44, 49, 48, 54, 44, 49, 49, 53, 44, 51, 52, 44, 52, 52, 44, 51, 52, 44, 57, 55, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 55, 44, 49, 49, 48, 44, 49, 49, 54, 44, 51, 52, 44, 53, 56, 44, 57, 49, 44, 49, 50, 51, 44, 51, 52, 44, 49, 48, 48, 44, 49, 48, 49, 44, 49, 49, 48, 44, 49, 49, 49, 44, 49, 48, 57, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 57, 56, 44, 57, 55, 44, 49, 49, 48, 44, 57, 55, 44, 49, 49, 48, 44, 57, 55, 44, 49, 49, 53, 44, 51, 52, 44, 52, 52, 44, 51, 52, 44, 57, 55, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 55, 44, 49, 49, 48, 44, 49, 49, 54, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 52, 57, 44, 52, 56, 44, 52, 56, 44, 51, 52, 44, 49, 50, 53, 44, 57, 51, 44, 49, 50, 53, 93, 125, 44, 123, 34, 116, 121, 112, 101, 95, 117, 114, 108, 34, 58, 34, 47, 99, 111, 115, 109, 111, 115, 46, 103, 111, 118, 46, 118, 49, 98, 101, 116, 97, 49, 46, 77, 115, 103, 83, 117, 98, 109, 105, 116, 80, 114, 111, 112, 111, 115, 97, 108, 34, 44, 34, 118, 97, 108, 117, 101, 34, 58, 91, 49, 50, 51, 44, 51, 52, 44, 49, 48, 53, 44, 49, 49, 48, 44, 49, 48, 53, 44, 49, 49, 54, 44, 49, 48, 53, 44, 57, 55, 44, 49, 48, 56, 44, 57, 53, 44, 49, 48, 48, 44, 49, 48, 49, 44, 49, 49, 50, 44, 49, 49, 49, 44, 49, 49, 53, 44, 49, 48, 53, 44, 49, 49, 54, 44, 51, 52, 44, 53, 56, 44, 57, 49, 44, 49, 50, 51, 44, 51, 52, 44, 49, 48, 48, 44, 49, 48, 49, 44, 49, 49, 48, 44, 49, 49, 49, 44, 49, 48, 57, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 57, 56, 44, 57, 55, 44, 49, 49, 48, 44, 57, 55, 44, 49, 49, 48, 44, 57, 55, 44, 49, 49, 53, 44, 51, 52, 44, 52, 52, 44, 51, 52, 44, 57, 55, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 55, 44, 49, 49, 48, 44, 49, 49, 54, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 52, 57, 44, 52, 56, 44, 52, 56, 44, 51, 52, 44, 49, 50, 53, 44, 57, 51, 44, 52, 52, 44, 51, 52, 44, 49, 49, 50, 44, 49, 49, 52, 44, 49, 49, 49, 44, 49, 49, 50, 44, 49, 49, 49, 44, 49, 49, 53, 44, 49, 48, 49, 44, 49, 49, 52, 44, 51, 52, 44, 53, 56, 44, 51, 52, 44, 57, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 49, 48, 57, 44, 49, 49, 49, 44, 49, 49, 53, 44, 52, 57, 44, 53, 53, 44, 49, 48, 48, 44, 49, 49, 54, 44, 49, 48, 56, 44, 52, 56, 44, 49, 48, 57, 44, 49, 48, 54, 44, 49, 49, 54, 44, 53, 49, 44, 49, 49, 54, 44, 53, 53, 44, 53, 53, 44, 49, 48, 55, 44, 49, 49, 50, 44, 49, 49, 55, 44, 49, 48, 52, 44, 49, 48, 51, 44, 53, 48, 44, 49, 48, 49, 44, 49, 48, 48, 44, 49, 49, 51, 44, 49, 50, 50, 44, 49, 48, 54, 44, 49, 49, 50, 44, 49, 49, 53, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 56, 44, 49, 49, 57, 44, 49, 48, 52, 44, 49, 48, 51, 44, 49, 50, 50, 44, 49, 49, 55, 44, 49, 48, 54, 44, 53, 55, 44, 49, 48, 56, 44, 49, 48, 54, 44, 49, 49, 53, 44, 51, 52, 44, 49, 50, 53, 93, 125, 93, 125}
				protoMessages = []proto.Message{
					&banktypes.MsgSend{
						FromAddress: TestOwnerAddress,
						ToAddress:   TestOwnerAddress,
						Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdk.NewInt(100))),
					},
					&govtypes.MsgSubmitProposal{
						InitialDeposit: sdk.NewCoins(sdk.NewCoin("bananas", sdk.NewInt(100))),
						Proposer:       TestOwnerAddress,
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

func (suite *TypesTestSuite) TestProtoSerializeAndDeserializeCosmosTx() {
	testCases := []struct {
		name    string
		msgs    []proto.Message
		expPass bool
	}{
		{
			"success: single msg, proto encoded",
			[]proto.Message{
				&banktypes.MsgSend{
					FromAddress: TestOwnerAddress,
					ToAddress:   TestOwnerAddress,
					Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdk.NewInt(100))),
				},
			},
			true,
		},
		{
			"success: multiple msgs, same types, proto encoded",
			[]proto.Message{
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
			},
			true,
		},
		{
			"success: multiple msgs, different types, proto encoded",
			[]proto.Message{
				&banktypes.MsgSend{
					FromAddress: TestOwnerAddress,
					ToAddress:   TestOwnerAddress,
					Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdk.NewInt(100))),
				},
				&govtypes.MsgSubmitProposal{
					InitialDeposit: sdk.NewCoins(sdk.NewCoin("bananas", sdk.NewInt(100))),
					Proposer:       TestOwnerAddress,
				},
			},
			true,
		},
		{
			"failure: unregistered msg type, proto encoded",
			[]proto.Message{
				&mockSdkMsg{},
			},
			false,
		},
		{
			"failure: multiple unregistered msg types, proto encoded",
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
			bz, err := types.SerializeCosmosTx(simapp.MakeTestEncodingConfig().Marshaler, tc.msgs, types.EncodingProtobuf)
			suite.Require().NoError(err, tc.name)

			msgs, err := types.DeserializeCosmosTx(simapp.MakeTestEncodingConfig().Marshaler, bz, types.EncodingProtobuf)
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

	// test serializing non sdk.Msg type
	bz, err := types.SerializeCosmosTx(simapp.MakeTestEncodingConfig().Marshaler, []proto.Message{&banktypes.MsgSendResponse{}}, types.EncodingProtobuf)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(bz)

	// test deserializing unknown bytes
	_, err = types.DeserializeCosmosTx(simapp.MakeTestEncodingConfig().Marshaler, bz, types.EncodingProtobuf)
	suite.Require().Error(err) // unregistered type

	// test deserializing unknown bytes
	msgs, err := types.DeserializeCosmosTx(simapp.MakeTestEncodingConfig().Marshaler, []byte("invalid"), types.EncodingProtobuf)
	suite.Require().Error(err)
	suite.Require().Empty(msgs)
}

func (suite *TypesTestSuite) TestJsonSerializeAndDeserializeCosmosTx() {
	testCases := []struct {
		name    string
		msgs    []proto.Message
		expPass bool
	}{
		{
			"success: single msg, json encoded",
			[]proto.Message{
				&banktypes.MsgSend{
					FromAddress: TestOwnerAddress,
					ToAddress:   TestOwnerAddress,
					Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdk.NewInt(100))),
				},
			},
			true,
		},
		{
			"success: multiple msgs, same types, json encoded",
			[]proto.Message{
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
			},
			true,
		},
		{
			"success: multiple msgs, different types, json encoded",
			[]proto.Message{
				&banktypes.MsgSend{
					FromAddress: TestOwnerAddress,
					ToAddress:   TestOwnerAddress,
					Amount:      sdk.NewCoins(sdk.NewCoin("bananas", sdk.NewInt(100))),
				},
				&govtypes.MsgSubmitProposal{
					InitialDeposit: sdk.NewCoins(sdk.NewCoin("bananas", sdk.NewInt(100))),
					Proposer:       TestOwnerAddress,
				},
			},
			true,
		},
		{
			"failure: unregistered msg type, json encoded",
			[]proto.Message{
				&mockSdkMsg{},
			},
			false,
		},
		{
			"failure: multiple unregistered msg types, json encoded",
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
			bz, errSerialize := types.SerializeCosmosTx(simapp.MakeTestEncodingConfig().Marshaler, tc.msgs, types.EncodingJSON)
			msgs, errDeserialize := types.DeserializeCosmosTx(simapp.MakeTestEncodingConfig().Marshaler, bz, types.EncodingJSON)
			if tc.expPass {
				suite.Require().NoError(errSerialize, tc.name)
				suite.Require().NoError(errDeserialize, tc.name)
				for i, msg := range msgs {
					suite.Require().Equal(tc.msgs[i], msg)
				}
			} else {
				suite.Require().Error(errSerialize, tc.name)
				suite.Require().Error(errDeserialize, tc.name)
			}
		})
	}

	// test serializing non sdk.Msg type
	bz, err := types.SerializeCosmosTx(simapp.MakeTestEncodingConfig().Marshaler, []proto.Message{&banktypes.MsgSendResponse{}}, types.EncodingJSON)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(bz)

	// test deserializing unknown bytes
	_, err = types.DeserializeCosmosTx(simapp.MakeTestEncodingConfig().Marshaler, bz, types.EncodingJSON)
	suite.Require().Error(err) // unregistered type

	// test deserializing unknown bytes
	msgs, err := types.DeserializeCosmosTx(simapp.MakeTestEncodingConfig().Marshaler, []byte("invalid"), types.EncodingJSON)
	suite.Require().Error(err)
	suite.Require().Empty(msgs)
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
