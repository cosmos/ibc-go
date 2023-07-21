package types_test

import (
	"fmt"

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

func (suite *TypesTestSuite) TestGetSourceCallbackAddress() {
	const expSrcCbAddr = "srcCbAddr"

	testCases := []struct {
		name       string
		packetData types.InterchainAccountPacketData
		expAddress string
	}{
		{
			"success: memo has src_callback in json struct and properly formatted address",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, expSrcCbAddr),
			},
			expSrcCbAddr,
		},
		{
			"failure: memo is empty",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: "",
			},
			"",
		},
		{
			"failure: memo is not json string",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: "memo",
			},
			"",
		},
		{
			"failure: memo does not have callbacks in json struct",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"Key": 10}`,
			},
			"",
		},
		{
			"failure: memo has src_callback in json struct but does not have address key",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"src_callback": {"Key": 10}}`,
			},
			"",
		},
		{
			"failure: memo has src_callback in json struct but does not have string value for address key",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"src_callback": {"address": 10}}`,
			},
			"",
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			srcCbAddr := tc.packetData.GetSourceCallbackAddress()
			suite.Require().Equal(tc.expAddress, srcCbAddr)
		})
	}
}

func (suite *TypesTestSuite) TestGetDestCallbackAddress() {
	// dest callback addresses are not supported for ICS 27
	const testDestCbAddr = "destCbAddr"

	testCases := []struct {
		name       string
		packetData types.InterchainAccountPacketData
		expAddress string
	}{
		{
			"failure: memo has dest_callback in json struct and properly formatted address",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, testDestCbAddr),
			},
			"",
		},
		{
			"failure: memo is empty",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: "",
			},
			"",
		},
		{
			"failure: memo is not json string",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: "memo",
			},
			"",
		},
		{
			"failure: memo does not have callbacks in json struct",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"Key": 10}`,
			},
			"",
		},
		{
			"failure: memo has callbacks in json struct but does not have dest_callback address key",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"dest_callback": {"Key": 10}}`,
			},
			"",
		},
		{
			"failure: memo has dest_callback in json struct but does not have string value for address key",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"dest_callback": {"address": 10}}`,
			},
			"",
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			srcCbAddr := tc.packetData.GetDestCallbackAddress()
			suite.Require().Equal(tc.expAddress, srcCbAddr)
		})
	}
}

func (suite *TypesTestSuite) TestSourceUserDefinedGasLimit() {
	testCases := []struct {
		name       string
		packetData types.InterchainAccountPacketData
		expUserGas uint64
	}{
		{
			"success: memo has user defined gas limit",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"src_callback": {"gas_limit": "100"}}`,
			},
			100,
		},
		{
			"failure: memo is empty",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: "",
			},
			0,
		},
		{
			"failure: memo has user defined gas limit as json number",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"src_callback": {"gas_limit": 100}}`,
			},
			0,
		},
		{
			"failure: memo has user defined gas limit as negative",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"src_callback": {"gas_limit": "-100"}}`,
			},
			0,
		},
		{
			"failure: memo has user defined gas limit as string",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"src_callback": {"gas_limit": "invalid"}}`,
			},
			0,
		},
		{
			"failure: memo has user defined gas limit as empty string",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"src_callback": {"gas_limit": ""}}`,
			},
			0,
		},
		{
			"failure: malformed memo",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `invalid`,
			},
			0,
		},
	}

	for _, tc := range testCases {
		suite.Require().Equal(tc.expUserGas, tc.packetData.GetSourceUserDefinedGasLimit())
	}
}

func (suite *TypesTestSuite) TestDestUserDefinedGasLimit() {
	// dest user defined gas limits are not supported for ICS 27
	testCases := []struct {
		name       string
		packetData types.InterchainAccountPacketData
		expUserGas uint64
	}{
		{
			"failure: memo has user defined gas limit",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"dest_callback": {"gas_limit": "100"}}`,
			},
			0,
		},
		{
			"failure: memo is empty",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: "",
			},
			0,
		},
		{
			"failure: memo has user defined gas limit as json number",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"dest_callback": {"gas_limit": 100}}`,
			},
			0,
		},
		{
			"failure: memo has user defined gas limit as negative",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"dest_callback": {"gas_limit": "-100"}}`,
			},
			0,
		},
		{
			"failure: memo has user defined gas limit as string",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"dest_callback": {"gas_limit": "invalid"}}`,
			},
			0,
		},
		{
			"failure: memo has user defined gas limit as empty string",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"dest_callback": {"gas_limit": ""}}`,
			},
			0,
		},
		{
			"failure: malformed memo",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `invalid`,
			},
			0,
		},
	}

	for _, tc := range testCases {
		suite.Require().Equal(tc.expUserGas, tc.packetData.GetDestUserDefinedGasLimit())
	}
}

func (suite *TypesTestSuite) TestGetPacketSenderAndReceiver() {
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
		suite.Require().Equal(tc.expSender, packetData.GetPacketSender(tc.srcPortID, ibctesting.InvalidID))
		// GetPacketReceiver always returns empty string for ICS 27
		suite.Require().Equal("", packetData.GetPacketReceiver(tc.srcPortID, tc.srcPortID))
	}
}
