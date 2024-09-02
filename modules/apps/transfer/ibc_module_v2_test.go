package transfer_test

import (
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/ibc-go/v9/testing/mock"
)

func (suite *TransferTestSuite) TestIBCModuleV2HappyPath() {
	var (
		path             *ibctesting.Path
		data             []channeltypes.PacketData
		expectedMultiAck channeltypes.MultiAcknowledgement
	)

	testCases := []struct {
		name       string
		malleate   func()
		expError   error
		expVersion string
	}{
		{
			"success", func() {}, nil, types.V2,
		},
		{
			"success async", func() {

		}, nil, types.V2,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.SetupV2()

			ftpd := types.FungibleTokenPacketDataV2{
				Tokens: []types.Token{
					{
						Denom:  types.NewDenom(ibctesting.TestCoin.Denom),
						Amount: "1000",
					},
				},
				Sender:     suite.chainA.SenderAccount.GetAddress().String(),
				Receiver:   suite.chainB.SenderAccount.GetAddress().String(),
				Memo:       "",
				Forwarding: types.ForwardingPacketData{},
			}

			bz, err := ftpd.Marshal()
			suite.Require().NoError(err)

			data = []channeltypes.PacketData{
				{
					AppName: types.ModuleName,
					Payload: channeltypes.Payload{
						Value:   bz,
						Version: types.V2,
					},
				},
				{

					AppName: mock.ModuleNameV2,
					Payload: channeltypes.Payload{
						Value: []byte("data"),
					},
				},
			}

			expectedMultiAck = channeltypes.MultiAcknowledgement{
				AcknowledgementResults: []channeltypes.AcknowledgementResult{
					{
						AppName: types.ModuleName,
						RecvPacketResult: channeltypes.RecvPacketResult{
							Status:          channeltypes.PacketStatus_Success,
							Acknowledgement: channeltypes.NewResultAcknowledgement([]byte{byte(1)}).Acknowledgement(),
						},
					},
					{
						AppName: mock.ModuleNameV2,
						RecvPacketResult: channeltypes.RecvPacketResult{
							Status:          channeltypes.PacketStatus_Success,
							Acknowledgement: channeltypes.NewResultAcknowledgement([]byte("success")).Acknowledgement(),
						},
					},
				},
			}

			tc.malleate()

			timeoutHeight := suite.chainA.GetTimeoutHeight()

			sequence, err := path.EndpointA.SendPacketV2POC(timeoutHeight, 0, data)
			suite.Require().NoError(err)

			packet := channeltypes.NewPacketV2(data, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ClientID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ClientID, timeoutHeight, 0)

			err = path.EndpointB.RecvPacketV2(packet)
			suite.Require().NoError(err)

			err = path.EndpointA.AcknowledgePacketV2(packet, expectedMultiAck)
			suite.Require().NoError(err)

		})
	}
}
