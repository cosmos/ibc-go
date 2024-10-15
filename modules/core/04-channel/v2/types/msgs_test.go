package types_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/ibc-go/v9/testing/mock"
)

type TypesTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator
	chainA      *ibctesting.TestChain
	chainB      *ibctesting.TestChain

	proof []byte
}

func (s *TypesTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
}

func TestTypesTestSuite(t *testing.T) {
	suite.Run(t, new(TypesTestSuite))
}

// TestMsgProvideCounterpartyValidateBasic tests ValidateBasic for MsgProvideCounterparty
func (s *TypesTestSuite) TestMsgProvideCounterpartyValidateBasic() {
	var msg *types.MsgProvideCounterparty

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"failure: invalid signer address",
			func() {
				msg.Signer = "invalid"
			},
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: invalid channel ID",
			func() {
				msg.ChannelId = ""
			},
			host.ErrInvalidID,
		},
		{
			"failure: invalid counterparty channel ID",
			func() {
				msg.CounterpartyChannelId = ""
			},
			host.ErrInvalidID,
		},
	}

	for _, tc := range testCases {
		msg = types.NewMsgProvideCounterparty(
			ibctesting.FirstChannelID,
			ibctesting.SecondChannelID,
			ibctesting.TestAccAddress,
		)

		tc.malleate()

		err := msg.ValidateBasic()
		expPass := tc.expError == nil
		if expPass {
			s.Require().NoError(err, "valid case %s failed", tc.name)
		} else {
			s.Require().ErrorIs(err, tc.expError, "invalid case %s passed", tc.name)
		}
	}
}

// TestMsgCreateChannelValidateBasic tests ValidateBasic for MsgCreateChannel
func (s *TypesTestSuite) TestMsgCreateChannelValidateBasic() {
	var msg *types.MsgCreateChannel

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"failure: invalid signer address",
			func() {
				msg.Signer = "invalid"
			},
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: invalid client ID",
			func() {
				msg.ClientId = ""
			},
			host.ErrInvalidID,
		},
		{
			"failure: empty key path",
			func() {
				msg.MerklePathPrefix.KeyPath = nil
			},
			errors.New("path cannot have length 0"),
		},
	}

	for _, tc := range testCases {
		msg = types.NewMsgCreateChannel(
			ibctesting.FirstClientID,
			commitmenttypes.NewMerklePath([]byte("key")),
			ibctesting.TestAccAddress,
		)

		tc.malleate()

		err := msg.ValidateBasic()
		expPass := tc.expError == nil
		if expPass {
			s.Require().NoError(err, "valid case %s failed", tc.name)
		} else {
			s.Require().ErrorContains(err, tc.expError.Error(), "invalid case %s passed", tc.name)
		}
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
			name: "failure: invalid packet",
			malleate: func() {
				msg.Packet.Data = []types.PacketData{}
			},
			expError: types.ErrInvalidPacket,
		},
		{
			name: "failure: invalid proof commitment",
			malleate: func() {
				msg.ProofCommitment = []byte{}
			},
			expError: commitmenttypes.ErrInvalidProof,
		},
		{
			name: "failure: invalid proof height",
			malleate: func() {
				msg.ProofHeight = clienttypes.ZeroHeight()
			},
			expError: clienttypes.ErrInvalidHeight,
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
			packet := types.NewPacket(1,
				ibctesting.FirstChannelID, ibctesting.FirstChannelID,
				s.chainA.GetTimeoutTimestamp(),
				types.PacketData{
					SourcePort:      ibctesting.MockPort,
					DestinationPort: ibctesting.MockPort,
					Payload:         types.NewPayload("ics20-1", "json", mock.MockPacketData),
				},
			)

			msg = types.NewMsgRecvPacket(packet, []byte("foo"), s.chainA.GetTimeoutHeight(), s.chainA.SenderAccount.GetAddress().String())

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
