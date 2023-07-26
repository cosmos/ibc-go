package types_test

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

var largeMemo = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum"

func (suite *TypesTestSuite) TestValidateBasic() {
	testCases := []struct {
		name       string
		packetData types.InterchainAccountPacketData
		expPass    bool
	}{
		{
			"success",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: "memo",
			},
			true,
		},
		{
			"success, empty memo",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
			},
			true,
		},
		{
			"type unspecified",
			types.InterchainAccountPacketData{
				Type: types.UNSPECIFIED,
				Data: []byte("data"),
				Memo: "memo",
			},
			false,
		},
		{
			"empty data",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte{},
				Memo: "memo",
			},
			false,
		},
		{
			"nil data",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: nil,
				Memo: "memo",
			},
			false,
		},
		{
			"memo too large",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: largeMemo,
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			err := tc.packetData.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestAdditionalPacketDataProvider() {
	expCallbackAddr := ibctesting.TestAccAddress
	sender := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

	testCases := []struct {
		name              string
		packetData        types.InterchainAccountPacketData
		expAdditionalData map[string]interface{}
		expPacketSender   string
	}{
		{
			"success: src_callback key in memo",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, expCallbackAddr),
			},
			map[string]interface{}{
				"address": expCallbackAddr,
			},
			sender,
		},
		{
			"success: src_callback key in memo with additional fields",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "200000"}}`, expCallbackAddr),
			},
			map[string]interface{}{
				"address":   expCallbackAddr,
				"gas_limit": "200000",
			},
			sender,
		},
		{
			"failure: empty memo",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: "",
			},
			nil,
			sender,
		},
		{
			"failure: non-json memo",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: "invalid",
			},
			nil,
			sender,
		},
		{
			"failure: invalid src_callback key",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"src_callback": "invalid"}`,
			},
			nil,
			sender,
		},
	}

	for _, tc := range testCases {
		additionalData := tc.packetData.GetAdditionalData("src_callback")
		suite.Require().Equal(tc.expAdditionalData, additionalData)
		suite.Require().Equal(tc.expPacketSender, tc.packetData.GetPacketSender(types.ControllerPortPrefix+sender))
	}
}

func (suite *TypesTestSuite) TestGetPacketSender() {
	// dest user defined gas limits are not supported for ICS 27
	testCases := []struct {
		name      string
		srcPortID string
		expSender string
	}{
		{
			"success: port id has prefix",
			types.ControllerPortPrefix + ibctesting.TestAccAddress,
			ibctesting.TestAccAddress,
		},
		{
			"failure: missing prefix",
			ibctesting.TestAccAddress,
			"",
		},
		{
			"failure: empty port id",
			"",
			"",
		},
	}

	for _, tc := range testCases {
		packetData := types.InterchainAccountPacketData{}
		suite.Require().Equal(tc.expSender, packetData.GetPacketSender(tc.srcPortID))
	}
}
