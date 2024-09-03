package transfer_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/ibc-go/v9/testing/mock"
)

func (suite *TransferTestSuite) TestIBCModuleV2HappyPath() {
	var (
		path                   *ibctesting.Path
		data                   []channeltypes.PacketData
		expectedMultiAck       channeltypes.MultiAcknowledgement
		expectedStoredMultiAck channeltypes.MultiAcknowledgement
		asyncAckFn             func(channeltypes.PacketV2) error
		expAsync               bool
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success", func() {}, nil,
		},
		{
			"success async", func() {
				expAsync = true
				suite.chainB.GetSimApp().MockV2Module.IBCApp.OnRecvPacketV2 = func(ctx sdk.Context, packet channeltypes.PacketV2, payload channeltypes.Payload, relayer sdk.AccAddress) channeltypes.RecvPacketResult {
					return channeltypes.RecvPacketResult{
						Status:          channeltypes.PacketStatus_Async,
						Acknowledgement: channeltypes.NewResultAcknowledgement([]byte("async")).Acknowledgement(),
					}
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

				expectedStoredMultiAck = channeltypes.MultiAcknowledgement{
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
								Status:          channeltypes.PacketStatus_Async,
								Acknowledgement: channeltypes.NewResultAcknowledgement([]byte("async")).Acknowledgement(),
							},
						},
					},
				}

				asyncAckFn = func(packet channeltypes.PacketV2) error {
					return suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteAcknowledgementAsyncV2(suite.chainB.GetContext(), packet, mock.ModuleNameV2, channeltypes.RecvPacketResult{
						Status:          channeltypes.PacketStatus_Success,
						Acknowledgement: channeltypes.NewResultAcknowledgement([]byte("success")).Acknowledgement(),
					})
				}

			}, nil,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			expAsync = false
			asyncAckFn = nil

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

			actualMultiAck, ok := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.GetMultiAcknowledgement(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ClientID, packet.GetSequence())

			if !expAsync {
				suite.Require().False(ok, "multi ack should not be written in sync case")
				err = path.EndpointA.AcknowledgePacketV2(packet, expectedMultiAck)
				suite.Require().NoError(err)
				return
			}

			// remainder of test handles the async case.
			suite.Require().True(ok, "multi ack should be written in async case")

			suite.Require().Equal(expectedStoredMultiAck, actualMultiAck, "stored multi ack is not as expected")

			// at some future point, the async acknowledgement is written by the application.
			err = asyncAckFn(packet)
			suite.Require().NoError(err, "failed to write async ack")

			suite.Require().NoError(path.EndpointB.UpdateClient())
			suite.Require().NoError(path.EndpointA.UpdateClient())

			err = path.EndpointA.AcknowledgePacketV2(packet, expectedMultiAck)
			suite.Require().NoError(err)

		})
	}
}
