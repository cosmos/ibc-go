package fee_test

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	feetypes "github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func NewFeeTransferPath(chainA, chainB *ibctesting.TestChain) *ibctesting.Path {
	path := ibctesting.NewTransferPath(chainA, chainB)

	feeTransferVersion := string(feetypes.ModuleCdc.MustMarshalJSON(&feetypes.Metadata{FeeVersion: feetypes.Version, AppVersion: transfertypes.V2}))
	path.EndpointA.ChannelConfig.Version = feeTransferVersion
	path.EndpointB.ChannelConfig.Version = feeTransferVersion
	path.EndpointA.ChannelConfig.PortID = transfertypes.PortID
	path.EndpointB.ChannelConfig.PortID = transfertypes.PortID

	return path
}

func (suite *FeeTestSuite) TestIbcModuleV2HappyPathFeeTransfer() {
	var path *ibctesting.Path

	testCases := []struct {
		name       string
		malleate   func()
		expError   error
		expVersion string
	}{
		{
			"success", func() {}, nil, transfertypes.V2,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = NewFeeTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			// configure counterparty payee address for forward relayer
			suite.chainB.GetSimApp().IBCFeeKeeper.SetCounterpartyPayeeAddress(
				suite.chainB.GetContext(),
				suite.chainB.SenderAccount.GetAddress().String(),
				ibctesting.TestAccAddress,
				path.EndpointB.ChannelID,
			)

			ftpd := transfertypes.FungibleTokenPacketDataV2{
				Tokens: []transfertypes.Token{
					{
						Denom:  transfertypes.NewDenom(ibctesting.TestCoin.Denom),
						Amount: "1000",
					},
				},
				Sender:     suite.chainA.SenderAccount.GetAddress().String(),
				Receiver:   suite.chainB.SenderAccount.GetAddress().String(),
				Memo:       "",
				Forwarding: transfertypes.ForwardingPacketData{},
			}

			bz, err := ftpd.Marshal()
			suite.Require().NoError(err)

			transferV2PacketData := channeltypes.PacketData{
				AppName: transfertypes.ModuleName,
				Payload: channeltypes.Payload{
					Value:   bz,
					Version: transfertypes.V2,
				},
			}

			feePacketData := feetypes.PacketData{
				PacketFee: feetypes.NewPacketFee(testvalues.DefaultFee(sdk.DefaultBondDenom), suite.chainA.SenderAccount.GetAddress().String(), nil),
			}

			bz, err = proto.Marshal(&feePacketData)
			suite.Require().NoError(err)

			feeV2PacketData := channeltypes.PacketData{
				AppName: feetypes.ModuleName,
				Payload: channeltypes.Payload{
					Value:   bz,
					Version: feetypes.Version,
				},
			}

			tc.malleate()

			data := []channeltypes.PacketData{transferV2PacketData, feeV2PacketData}

			timeoutHeight := suite.chainA.GetTimeoutHeight()

			sequence, err := path.EndpointA.SendPacketV2(suite.chainA.GetTimeoutHeight(), 0, data)
			suite.Require().NoError(err)

			packet := channeltypes.NewPacketV2(data, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, 0)

			err = path.EndpointB.RecvPacketV2(packet)
			suite.Require().NoError(err)

			expectedMultiAck := channeltypes.MultiAcknowledgement{
				AcknowledgementResults: []channeltypes.AcknowledgementResult{
					{
						AppName: transfertypes.ModuleName,
						RecvPacketResult: channeltypes.RecvPacketResult{
							Status:          channeltypes.PacketStatus_Success,
							Acknowledgement: channeltypes.NewResultAcknowledgement([]byte{byte(1)}).Acknowledgement(),
						},
					},
					{
						AppName: feetypes.ModuleName,
						RecvPacketResult: channeltypes.RecvPacketResult{
							Status:          channeltypes.PacketStatus_Success,
							Acknowledgement: feetypes.NewFeeAcknowledgement(ibctesting.TestAccAddress).Acknowledgement(),
						},
					},
				},
			}

			err = path.EndpointA.AcknowledgePacketV2(packet, expectedMultiAck)
			suite.Require().NoError(err)

			// assert 29-fee logic is executed correctly by checking the balance of the forward relayer address on chainA
			forwardRelayerAcc, err := sdk.AccAddressFromBech32(ibctesting.TestAccAddress)
			suite.Require().NoError(err)

			expRecvFee := sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(50))
			balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), forwardRelayerAcc, sdk.DefaultBondDenom)

			suite.Require().True(balance.Equal(expRecvFee))
		})
	}
}
