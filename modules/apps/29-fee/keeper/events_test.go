package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	abcitypes "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/ibc-go/v4/modules/apps/29-fee/types"
)

func (suite *KeeperTestSuite) TestIncentivizePacketEvent() {
	var (
		expRecvFees    sdk.Coins
		expAckFees     sdk.Coins
		expTimeoutFees sdk.Coins
	)

	suite.coordinator.Setup(suite.path)

	fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
	msg := types.NewMsgPayPacketFee(
		fee,
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		suite.chainA.SenderAccount.GetAddress().String(),
		nil,
	)

	expRecvFees = expRecvFees.Add(fee.RecvFee...)
	expAckFees = expAckFees.Add(fee.AckFee...)
	expTimeoutFees = expTimeoutFees.Add(fee.TimeoutFee...)

	result, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err)

	var incentivizedPacketEvent abcitypes.Event
	for _, event := range result.Events {
		if event.Type == types.EventTypeIncentivizedPacket {
			incentivizedPacketEvent = event
		}
	}

	for _, attr := range incentivizedPacketEvent.Attributes {
		switch string(attr.Key) {
		case types.AttributeKeyRecvFee:
			suite.Require().Equal(expRecvFees.String(), string(attr.Value))

		case types.AttributeKeyAckFee:
			suite.Require().Equal(expAckFees.String(), string(attr.Value))

		case types.AttributeKeyTimeoutFee:
			suite.Require().Equal(expTimeoutFees.String(), string(attr.Value))
		}
	}

	// send the same messages again a few times
	for i := 0; i < 3; i++ {
		expRecvFees = expRecvFees.Add(fee.RecvFee...)
		expAckFees = expAckFees.Add(fee.AckFee...)
		expTimeoutFees = expTimeoutFees.Add(fee.TimeoutFee...)

		result, err = suite.chainA.SendMsgs(msg)
		suite.Require().NoError(err)
	}

	for _, event := range result.Events {
		if event.Type == types.EventTypeIncentivizedPacket {
			incentivizedPacketEvent = event
		}
	}

	for _, attr := range incentivizedPacketEvent.Attributes {
		switch string(attr.Key) {
		case types.AttributeKeyRecvFee:
			suite.Require().Equal(expRecvFees.String(), string(attr.Value))

		case types.AttributeKeyAckFee:
			suite.Require().Equal(expAckFees.String(), string(attr.Value))

		case types.AttributeKeyTimeoutFee:
			suite.Require().Equal(expTimeoutFees.String(), string(attr.Value))
		}
	}
}
