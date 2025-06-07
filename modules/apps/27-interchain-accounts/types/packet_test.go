package types_test

import (
	"fmt"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *TypesTestSuite) TestValidateBasic() {
	testCases := []struct {
		name       string
		packetData types.InterchainAccountPacketData
		expErr     error
	}{
		{
			"success",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: "memo",
			},
			nil,
		},
		{
			"success, empty memo",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
			},
			nil,
		},
		{
			"type unspecified",
			types.InterchainAccountPacketData{
				Type: types.UNSPECIFIED,
				Data: []byte("data"),
				Memo: "memo",
			},
			types.ErrInvalidOutgoingData,
		},
		{
			"empty data",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte{},
				Memo: "memo",
			},
			types.ErrInvalidOutgoingData,
		},
		{
			"nil data",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: nil,
				Memo: "memo",
			},
			types.ErrInvalidOutgoingData,
		},
		{
			"memo too large",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: ibctesting.GenerateString(types.MaxMemoCharLength + 1),
			},
			types.ErrInvalidOutgoingData,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			err := tc.packetData.ValidateBasic()

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (s *TypesTestSuite) TestGetPacketSender() {
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
		s.Require().Equal(tc.expSender, packetData.GetPacketSender(tc.srcPortID))
	}
}

func (s *TypesTestSuite) TestPacketDataProvider() {
	expCallbackAddr := ibctesting.TestAccAddress

	testCases := []struct {
		name          string
		packetData    types.InterchainAccountPacketData
		expCustomData any
	}{
		{
			"success: src_callback key in memo",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, expCallbackAddr),
			},
			map[string]any{
				"address": expCallbackAddr,
			},
		},
		{
			"success: src_callback key in memo with additional fields",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "200000"}}`, expCallbackAddr),
			},
			map[string]any{
				"address":   expCallbackAddr,
				"gas_limit": "200000",
			},
		},
		{
			"success: src_callback has string value",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: `{"src_callback": "string"}`,
			},
			"string",
		},
		{
			"failure: empty memo",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: "",
			},
			nil,
		},
		{
			"failure: non-json memo",
			types.InterchainAccountPacketData{
				Type: types.EXECUTE_TX,
				Data: []byte("data"),
				Memo: "invalid",
			},
			nil,
		},
	}

	for _, tc := range testCases {
		customData := tc.packetData.GetCustomPacketData("src_callback")
		s.Require().Equal(tc.expCustomData, customData)
	}
}

func (s *TypesTestSuite) TestPacketDataUnmarshalerInterface() {
	expPacketData := types.InterchainAccountPacketData{
		Type: types.EXECUTE_TX,
		Data: []byte("data"),
		Memo: "some memo",
	}

	var packetData types.InterchainAccountPacketData
	err := packetData.UnmarshalJSON(expPacketData.GetBytes())
	s.Require().NoError(err)
	s.Require().Equal(expPacketData, packetData)

	// test invalid packet data
	invalidPacketDataBytes := []byte("invalid packet data")

	var invalidPacketData types.InterchainAccountPacketData
	err = packetData.UnmarshalJSON(invalidPacketDataBytes)
	s.Require().Error(err)
	s.Require().Equal(types.InterchainAccountPacketData{}, invalidPacketData)
}
