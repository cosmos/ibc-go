package gmp_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

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
}

const (
	validClientID   = ibctesting.FirstClientID
	invalidClientID = "invalid"
)

func TestIBCModuleTestSuite(t *testing.T) {
	testifysuite.Run(t, new(IBCModuleTestSuite))
}

func (s *IBCModuleTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 1)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
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
				payload.SourcePort = "invalid-port"
			},
			channeltypesv2.ErrInvalidPacket,
		},
		{
			"failure: invalid destination port",
			func() {
				payload.DestinationPort = "invalid-port"
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
