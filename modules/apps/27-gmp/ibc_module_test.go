package gmp_test

import (
	"testing"

	"github.com/cosmos/gogoproto/proto"
	testifysuite "github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	gmp "github.com/cosmos/ibc-go/v10/modules/apps/27-gmp"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

type IBCModuleTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator
	chainA      *ibctesting.TestChain
	chainB      *ibctesting.TestChain
}

const (
	validClientID   = ibctesting.FirstClientID
	invalidClientID = "invalid"
	invalidPort     = "invalid-port"
)

func TestIBCModuleTestSuite(t *testing.T) {
	testifysuite.Run(t, new(IBCModuleTestSuite))
}

func (s *IBCModuleTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
}

func (s *IBCModuleTestSuite) TestOnSendPacket() {
	var (
		module       *gmp.IBCModule
		payload      channeltypesv2.Payload
		signer       sdk.AccAddress
		sourceClient string
		destClient   string
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"failure: invalid source port",
			func() {
				payload.SourcePort = invalidPort
			},
			channeltypesv2.ErrInvalidPacket,
		},
		{
			"failure: invalid destination port",
			func() {
				payload.DestinationPort = invalidPort
			},
			channeltypesv2.ErrInvalidPacket,
		},
		{
			"failure: invalid source client ID",
			func() {
				sourceClient = invalidClientID
			},
			channeltypesv2.ErrInvalidPacket,
		},
		{
			"failure: invalid destination client ID",
			func() {
				destClient = invalidClientID
			},
			channeltypesv2.ErrInvalidPacket,
		},
		{
			"failure: sender != signer",
			func() {
				signer = s.chainA.SenderAccounts[1].SenderAccount.GetAddress()
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: unmarshal packet data error",
			func() {
				payload.Value = []byte("invalid")
			},
			ibcerrors.ErrInvalidType,
		},
		{
			"failure: ValidateBasic error - empty sender",
			func() {
				packetData := types.NewGMPPacketData("", "", []byte("salt"), []byte("payload"), "")
				dataBz, err := types.MarshalPacketData(&packetData, types.Version, types.EncodingProtobuf)
				s.Require().NoError(err)
				payload.Value = dataBz
			},
			ibcerrors.ErrInvalidAddress,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			module = gmp.NewIBCModule(s.chainA.GetSimApp().GMPKeeper)
			signer = s.chainA.SenderAccount.GetAddress()
			sourceClient = validClientID
			destClient = validClientID

			packetData := types.NewGMPPacketData(signer.String(), "", []byte("salt"), []byte("payload"), "")
			dataBz, err := types.MarshalPacketData(&packetData, types.Version, types.EncodingProtobuf)
			s.Require().NoError(err)

			payload = channeltypesv2.NewPayload(types.PortID, types.PortID, types.Version, types.EncodingProtobuf, dataBz)

			tc.malleate()

			err = module.OnSendPacket(
				s.chainA.GetContext(),
				sourceClient,
				destClient,
				1,
				payload,
				signer,
			)

			expPass := tc.expErr == nil
			if expPass {
				s.Require().NoError(err)
			} else {
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (s *IBCModuleTestSuite) TestOnRecvPacket() {
	const testSalt = "test-salt"

	var (
		module         *gmp.IBCModule
		payload        channeltypesv2.Payload
		gmpAccountAddr sdk.AccAddress
		sender         string
		msgPayload     []byte
	)

	testCases := []struct {
		name      string
		malleate  func()
		expStatus channeltypesv2.PacketStatus
	}{
		{
			"success",
			func() {
				s.fundAccount(gmpAccountAddr, sdk.NewCoins(ibctesting.TestCoin))
			},
			channeltypesv2.PacketStatus_Success,
		},
		{
			"failure: invalid source port",
			func() {
				payload.SourcePort = invalidPort
			},
			channeltypesv2.PacketStatus_Failure,
		},
		{
			"failure: invalid destination port",
			func() {
				payload.DestinationPort = invalidPort
			},
			channeltypesv2.PacketStatus_Failure,
		},
		{
			"failure: invalid version",
			func() {
				payload.Version = "invalid-version"
			},
			channeltypesv2.PacketStatus_Failure,
		},
		{
			"failure: invalid packet data - unmarshal error",
			func() {
				payload.Value = []byte("invalid")
			},
			channeltypesv2.PacketStatus_Failure,
		},
		{
			"failure: ValidateBasic error - empty sender",
			func() {
				packetData := types.NewGMPPacketData("", "", []byte(testSalt), msgPayload, "")
				dataBz, err := types.MarshalPacketData(&packetData, types.Version, types.EncodingProtobuf)
				s.Require().NoError(err)
				payload.Value = dataBz
			},
			channeltypesv2.PacketStatus_Failure,
		},
		{
			"failure: keeper OnRecvPacket error - unauthorized signer",
			func() {
				unauthorizedPayload := s.serializeMsgs(s.newMsgSend(s.chainA.SenderAccount.GetAddress(), s.chainB.SenderAccount.GetAddress()))
				packetData := types.NewGMPPacketData(sender, "", []byte(testSalt), unauthorizedPayload, "")
				dataBz, err := types.MarshalPacketData(&packetData, types.Version, types.EncodingProtobuf)
				s.Require().NoError(err)
				payload.Value = dataBz
			},
			channeltypesv2.PacketStatus_Failure,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			module = gmp.NewIBCModule(s.chainA.GetSimApp().GMPKeeper)
			sender = s.chainB.SenderAccount.GetAddress().String()
			recipient := s.chainA.SenderAccount.GetAddress()

			accountID := types.NewAccountIdentifier(ibctesting.FirstClientID, sender, []byte(testSalt))
			addr, err := types.BuildAddressPredictable(&accountID)
			s.Require().NoError(err)
			gmpAccountAddr = addr

			msgPayload = s.serializeMsgs(s.newMsgSend(gmpAccountAddr, recipient))
			packetData := types.NewGMPPacketData(sender, "", []byte(testSalt), msgPayload, "")
			dataBz, err := types.MarshalPacketData(&packetData, types.Version, types.EncodingProtobuf)
			s.Require().NoError(err)

			payload = channeltypesv2.NewPayload(types.PortID, types.PortID, types.Version, types.EncodingProtobuf, dataBz)

			tc.malleate()

			result := module.OnRecvPacket(
				s.chainA.GetContext(),
				validClientID,
				validClientID,
				1,
				payload,
				s.chainA.SenderAccount.GetAddress(),
			)

			s.Require().Equal(tc.expStatus, result.Status)
			if tc.expStatus == channeltypesv2.PacketStatus_Success {
				s.Require().NotEmpty(result.Acknowledgement)
			}
		})
	}
}

func (s *IBCModuleTestSuite) TestOnTimeoutPacket() {
	s.SetupTest()

	module := gmp.NewIBCModule(s.chainA.GetSimApp().GMPKeeper)
	payload := channeltypesv2.Payload{}

	err := module.OnTimeoutPacket(
		s.chainA.GetContext(),
		validClientID,
		validClientID,
		1,
		payload,
		s.chainA.SenderAccount.GetAddress(),
	)

	s.Require().NoError(err)
}

func (s *IBCModuleTestSuite) TestOnAcknowledgementPacket() {
	s.SetupTest()

	module := gmp.NewIBCModule(s.chainA.GetSimApp().GMPKeeper)
	payload := channeltypesv2.Payload{}

	err := module.OnAcknowledgementPacket(
		s.chainA.GetContext(),
		validClientID,
		validClientID,
		1,
		[]byte("ack"),
		payload,
		s.chainA.SenderAccount.GetAddress(),
	)

	s.Require().NoError(err)
}

func (s *IBCModuleTestSuite) TestUnmarshalPacketData() {
	var (
		module  *gmp.IBCModule
		payload channeltypesv2.Payload
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success: valid protobuf payload",
			func() {},
			nil,
		},
		{
			"success: valid JSON payload",
			func() {
				packetData := types.NewGMPPacketData("cosmos1sender", "cosmos1receiver", []byte("salt"), []byte("payload"), "memo")
				dataBz, err := types.MarshalPacketData(&packetData, types.Version, types.EncodingJSON)
				s.Require().NoError(err)
				payload = channeltypesv2.NewPayload(types.PortID, types.PortID, types.Version, types.EncodingJSON, dataBz)
			},
			nil,
		},
		{
			"failure: invalid payload data",
			func() {
				payload.Value = []byte("invalid")
			},
			ibcerrors.ErrInvalidType,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			module = gmp.NewIBCModule(s.chainA.GetSimApp().GMPKeeper)

			// Default: valid protobuf payload
			packetData := types.NewGMPPacketData("cosmos1sender", "cosmos1receiver", []byte("salt"), []byte("payload"), "memo")
			dataBz, err := types.MarshalPacketData(&packetData, types.Version, types.EncodingProtobuf)
			s.Require().NoError(err)
			payload = channeltypesv2.NewPayload(types.PortID, types.PortID, types.Version, types.EncodingProtobuf, dataBz)

			tc.malleate()

			data, err := module.UnmarshalPacketData(payload)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().NotNil(data)

				gmpData, ok := data.(*types.GMPPacketData)
				s.Require().True(ok)
				s.Require().Equal("cosmos1sender", gmpData.Sender)
				s.Require().Equal("cosmos1receiver", gmpData.Receiver)
			} else {
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (s *IBCModuleTestSuite) fundAccount(addr sdk.AccAddress, coins sdk.Coins) {
	err := s.chainA.GetSimApp().BankKeeper.SendCoins(
		s.chainA.GetContext(),
		s.chainA.SenderAccount.GetAddress(),
		addr,
		coins,
	)
	s.Require().NoError(err)
}

func (s *IBCModuleTestSuite) newMsgSend(from, to sdk.AccAddress) *banktypes.MsgSend {
	s.T().Helper()

	return &banktypes.MsgSend{
		FromAddress: from.String(),
		ToAddress:   to.String(),
		Amount:      sdk.NewCoins(ibctesting.TestCoin),
	}
}

func (s *IBCModuleTestSuite) serializeMsgs(msgs ...proto.Message) []byte {
	payload, err := types.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), msgs)
	s.Require().NoError(err)
	return payload
}
