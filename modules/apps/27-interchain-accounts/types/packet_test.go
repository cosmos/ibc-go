package types_test

import (
	"fmt"

	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
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
		expPass    bool
	}{
		{
			"memo is empty",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: "",
			},
			false,
		},
		{
			"memo is not json string",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: "memo",
			},
			false,
		},
		{
			"memo does not have callbacks in json struct",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"Key": 10}`,
			},
			false,
		},
		{
			"memo has callbacks in json struct but does not have src_callback_address key",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"callback": {"Key": 10}}`,
			},
			false,
		},
		{
			"memo has callbacks in json struct but does not have string value for src_callback_address key",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"callback": {"src_callback_address": 10}}`,
			},
			false,
		},
		{
			"memo has callbacks in json struct and properly formatted src_callback_address",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: fmt.Sprintf(`{"callback": {"src_callback_address": "%s"}}`, expSrcCbAddr),
			},
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			srcCbAddr := tc.packetData.GetSourceCallbackAddress()

			if tc.expPass {
				suite.Require().Equal(expSrcCbAddr, srcCbAddr)
			} else {
				suite.Require().Equal("", srcCbAddr)
			}
		})
	}
}

func (suite *TypesTestSuite) TestGetDestCallbackAddress() {
	testCases := []struct {
		name       string
		packetData types.InterchainAccountPacketData
	}{
		{
			"memo is empty",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: "",
			},
		},
		{
			"memo has dest callback address specified in json struct",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"callback": {"dest_callback_address": "testAddress"}}`,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			destCbAddr := tc.packetData.GetDestCallbackAddress()
			suite.Require().Equal("", destCbAddr)
		})
	}
}

func (suite *TypesTestSuite) TestUserDefinedGasLimit() {
	packetData := types.InterchainAccountPacketData{
		Type: types.EXECUTE_TX,
		Data: []byte("data"),
		Memo: `{"callback": {"gas_limit": "100"}}`,
	}

	suite.Require().Equal(uint64(100), packetData.UserDefinedGasLimit())
}
