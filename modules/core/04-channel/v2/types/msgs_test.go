package types_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	mockv2 "github.com/cosmos/ibc-go/v10/testing/mock/v2"
)

var testProof = []byte("test")

type TypesTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator
	chainA      *ibctesting.TestChain
	chainB      *ibctesting.TestChain
}

func (s *TypesTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
}

func TestTypesTestSuite(t *testing.T) {
	suite.Run(t, new(TypesTestSuite))
}

func (s *TypesTestSuite) TestMsgSendPacketValidateBasic() {
	var msg *types.MsgSendPacket
	var payload types.Payload
	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			name:     "success",
			malleate: func() {},
		},
		{
			name: "success, multiple payloads",
			malleate: func() {
				msg.Payloads = append(msg.Payloads, payload)
			},
		},
		{
			name: "failure: invalid source channel",
			malleate: func() {
				msg.SourceClient = ""
			},
			expError: host.ErrInvalidID,
		},
		{
			name: "failure: invalid timestamp",
			malleate: func() {
				msg.TimeoutTimestamp = 0
			},
			expError: types.ErrInvalidTimeout,
		},
		{
			name: "failure: invalid length for payload",
			malleate: func() {
				msg.Payloads = []types.Payload{}
			},
			expError: types.ErrInvalidPayload,
		},
		{
			name: "failure: invalid packetdata",
			malleate: func() {
				msg.Payloads = []types.Payload{{}}
			},
			expError: host.ErrInvalidID,
		},
		{
			name: "failure: invalid payload",
			malleate: func() {
				msg.Payloads[0].DestinationPort = ""
			},
			expError: host.ErrInvalidID,
		},
		{
			name: "failure: invalid multiple payload",
			malleate: func() {
				payload.DestinationPort = ""
				msg.Payloads = append(msg.Payloads, payload)
			},
			expError: host.ErrInvalidID,
		},
		{
			name: "failure: invalid signer",
			malleate: func() {
				msg.Signer = ""
			},
			expError: ibcerrors.ErrInvalidAddress,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			payload = types.Payload{SourcePort: ibctesting.MockPort, DestinationPort: ibctesting.MockPort, Version: "ics20-1", Encoding: transfertypes.EncodingJSON, Value: ibctesting.MockPacketData}
			msg = types.NewMsgSendPacket(
				ibctesting.FirstChannelID, s.chainA.GetTimeoutTimestamp(),
				s.chainA.SenderAccount.GetAddress().String(),
				payload,
			)

			tc.malleate()

			err := msg.ValidateBasic()
			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				ibctesting.RequireErrorIsOrContains(s.T(), err, tc.expError, err.Error())
			}
		})
	}
}

func (s *TypesTestSuite) TestMsgRecvPacketValidateBasic() {
	var msg *types.MsgRecvPacket
	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			name:     "success",
			malleate: func() {},
		},
		{
			name: "success, multiple payloads",
			malleate: func() {
				msg.Packet.Payloads = append(msg.Packet.Payloads, mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB))
			},
		},
		{
			name: "failure: invalid payload",
			malleate: func() {
				msg.Packet.Payloads = []types.Payload{}
			},
			expError: types.ErrInvalidPayload,
		},
		{
			name: "failure: invalid proof commitment",
			malleate: func() {
				msg.ProofCommitment = []byte{}
			},
			expError: commitmenttypes.ErrInvalidProof,
		},
		{
			name: "failure: invalid length for packet payloads",
			malleate: func() {
				msg.Packet.Payloads = []types.Payload{}
			},
			expError: types.ErrInvalidPayload,
		},
		{
			name: "failure: invalid individual payload",
			malleate: func() {
				msg.Packet.Payloads = []types.Payload{{}}
			},
			expError: host.ErrInvalidID,
		},
		{
			name: "failure: invalid multiple payload",
			malleate: func() {
				msg.Packet.Payloads = append(msg.Packet.Payloads, types.Payload{})
			},
			expError: host.ErrInvalidID,
		},
		{
			name: "failure: invalid signer",
			malleate: func() {
				msg.Signer = ""
			},
			expError: ibcerrors.ErrInvalidAddress,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			packet := types.NewPacket(1, ibctesting.FirstChannelID, ibctesting.SecondChannelID, s.chainA.GetTimeoutTimestamp(), mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB))

			msg = types.NewMsgRecvPacket(packet, testProof, s.chainA.GetTimeoutHeight(), s.chainA.SenderAccount.GetAddress().String())

			tc.malleate()

			err := msg.ValidateBasic()

			expPass := tc.expError == nil

			if expPass {
				s.Require().NoError(err)
			} else {
				ibctesting.RequireErrorIsOrContains(s.T(), err, tc.expError)
			}
		})
	}
}

func (s *TypesTestSuite) TestMsgAcknowledge_ValidateBasic() {
	var msg *types.MsgAcknowledgement
	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			name:     "success",
			malleate: func() {},
		},
		{
			name: "success, multiple payloads",
			malleate: func() {
				msg.Packet.Payloads = append(msg.Packet.Payloads, mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB))
			},
		},
		{
			name: "failure: invalid proof of acknowledgement",
			malleate: func() {
				msg.ProofAcked = []byte{}
			},
			expError: commitmenttypes.ErrInvalidProof,
		},
		{
			name: "failure: invalid length for packet payloads",
			malleate: func() {
				msg.Packet.Payloads = []types.Payload{}
			},
			expError: types.ErrInvalidPayload,
		},
		{
			name: "failure: invalid individual payload",
			malleate: func() {
				msg.Packet.Payloads = []types.Payload{{}}
			},
			expError: host.ErrInvalidID,
		},
		{
			name: "failure: invalid multiple payload",
			malleate: func() {
				msg.Packet.Payloads = append(msg.Packet.Payloads, types.Payload{})
			},
			expError: host.ErrInvalidID,
		},
		{
			name: "failure: invalid signer",
			malleate: func() {
				msg.Signer = ""
			},
			expError: ibcerrors.ErrInvalidAddress,
		},
		{
			name: "failure: invalid packet",
			malleate: func() {
				msg.Packet.Sequence = 0
			},
			expError: types.ErrInvalidPacket,
		},
		{
			name: "failure: invalid acknowledgement",
			malleate: func() {
				msg.Acknowledgement = types.NewAcknowledgement([]byte(""))
			},
			expError: types.ErrInvalidAcknowledgement,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			msg = types.NewMsgAcknowledgement(
				types.NewPacket(1, ibctesting.FirstChannelID, ibctesting.SecondChannelID, s.chainA.GetTimeoutTimestamp(), mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)),
				types.NewAcknowledgement([]byte("appAck1")),
				testProof,
				clienttypes.ZeroHeight(),
				s.chainA.SenderAccount.GetAddress().String(),
			)

			tc.malleate()

			err := msg.ValidateBasic()
			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)
			} else {
				ibctesting.RequireErrorIsOrContains(s.T(), err, tc.expError)
			}
		})
	}
}

func (s *TypesTestSuite) TestMsgTimeoutValidateBasic() {
	var msg *types.MsgTimeout

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			name:     "success",
			malleate: func() {},
		},
		{
			name: "success, multiple payloads",
			malleate: func() {
				msg.Packet.Payloads = append(msg.Packet.Payloads, mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB))
			},
		},
		{
			name: "failure: invalid signer",
			malleate: func() {
				msg.Signer = ""
			},
			expError: ibcerrors.ErrInvalidAddress,
		},
		{
			name: "failure: invalid length for packet payloads",
			malleate: func() {
				msg.Packet.Payloads = []types.Payload{}
			},
			expError: types.ErrInvalidPayload,
		},
		{
			name: "failure: invalid individual payload",
			malleate: func() {
				msg.Packet.Payloads = []types.Payload{{}}
			},
			expError: host.ErrInvalidID,
		},
		{
			name: "failure: invalid multiple payload",
			malleate: func() {
				msg.Packet.Payloads = append(msg.Packet.Payloads, types.Payload{})
			},
			expError: host.ErrInvalidID,
		},
		{
			name: "failure: invalid packet",
			malleate: func() {
				msg.Packet.Sequence = 0
			},
			expError: types.ErrInvalidPacket,
		},
		{
			name: "failure: invalid proof unreceived",
			malleate: func() {
				msg.ProofUnreceived = []byte{}
			},
			expError: commitmenttypes.ErrInvalidProof,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			msg = types.NewMsgTimeout(
				types.NewPacket(1, ibctesting.FirstChannelID, ibctesting.SecondChannelID, s.chainA.GetTimeoutTimestamp(), mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)),
				testProof,
				clienttypes.ZeroHeight(),
				s.chainA.SenderAccount.GetAddress().String(),
			)

			tc.malleate()

			err := msg.ValidateBasic()
			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)
			} else {
				ibctesting.RequireErrorIsOrContains(s.T(), err, tc.expError)
			}
		})
	}
}
