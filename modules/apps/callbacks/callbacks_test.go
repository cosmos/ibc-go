package ibccallbacks_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	feetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	"github.com/cosmos/ibc-go/v7/modules/apps/callbacks/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
)

const maxCallbackGas = uint64(1000000)

// CallbacksTestSuite defines the needed instances and methods to test callbacks
type CallbacksTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	path *ibctesting.Path
}

// setupChains sets up a coordinator with 2 test chains.
func (s *CallbacksTestSuite) setupChains() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
	s.path = ibctesting.NewPath(s.chainA, s.chainB)
}

// SetupTransferTest sets up a transfer channel between chainA and chainB
func (s *CallbacksTestSuite) SetupTransferTest() {
	s.setupChains()

	s.path.EndpointA.ChannelConfig.PortID = ibctesting.TransferPort
	s.path.EndpointB.ChannelConfig.PortID = ibctesting.TransferPort
	s.path.EndpointA.ChannelConfig.Version = transfertypes.Version
	s.path.EndpointB.ChannelConfig.Version = transfertypes.Version

	s.coordinator.Setup(s.path)
}

// SetupFeeTransferTest sets up a fee middleware enabled transfer channel between chainA and chainB
func (s *CallbacksTestSuite) SetupFeeTransferTest() {
	s.setupChains()

	feeTransferVersion := string(feetypes.ModuleCdc.MustMarshalJSON(&feetypes.Metadata{FeeVersion: feetypes.Version, AppVersion: transfertypes.Version}))
	s.path.EndpointA.ChannelConfig.Version = feeTransferVersion
	s.path.EndpointB.ChannelConfig.Version = feeTransferVersion
	s.path.EndpointA.ChannelConfig.PortID = transfertypes.PortID
	s.path.EndpointB.ChannelConfig.PortID = transfertypes.PortID

	s.coordinator.Setup(s.path)
}

func (s *CallbacksTestSuite) SetupMockFeeTest() {
	s.setupChains()

	mockFeeVersion := string(feetypes.ModuleCdc.MustMarshalJSON(&feetypes.Metadata{FeeVersion: feetypes.Version, AppVersion: ibcmock.Version}))
	s.path.EndpointA.ChannelConfig.Version = mockFeeVersion
	s.path.EndpointB.ChannelConfig.Version = mockFeeVersion
	s.path.EndpointA.ChannelConfig.PortID = ibctesting.MockFeePort
	s.path.EndpointB.ChannelConfig.PortID = ibctesting.MockFeePort
}

// SetupICATest sets up an interchain accounts channel between chainA (controller) and chainB (host).
// It funds and returns the interchain account address owned by chainA's SenderAccount.
func (s *CallbacksTestSuite) SetupICATest() string {
	s.setupChains()
	s.coordinator.SetupConnections(s.path)

	icaOwner := s.chainA.SenderAccount.GetAddress().String()
	// ICAVersion defines a interchain accounts version string
	icaVersion := icatypes.NewDefaultMetadataString(s.path.EndpointA.ConnectionID, s.path.EndpointB.ConnectionID)
	icaControllerPortID, err := icatypes.NewControllerPortID(icaOwner)
	s.Require().NoError(err)

	s.path.SetChannelOrdered()
	s.path.EndpointA.ChannelConfig.PortID = icaControllerPortID
	s.path.EndpointB.ChannelConfig.PortID = icatypes.HostPortID
	s.path.EndpointA.ChannelConfig.Version = icaVersion
	s.path.EndpointB.ChannelConfig.Version = icaVersion

	s.RegisterInterchainAccount(icaOwner)
	// open chan init must be skipped. So we cannot use .CreateChannels()
	err = s.path.EndpointB.ChanOpenTry()
	s.Require().NoError(err)
	err = s.path.EndpointA.ChanOpenAck()
	s.Require().NoError(err)
	err = s.path.EndpointB.ChanOpenConfirm()
	s.Require().NoError(err)

	interchainAccountAddr, found := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), s.path.EndpointA.ConnectionID, s.path.EndpointA.ChannelConfig.PortID)
	s.Require().True(found)

	// fund the interchain account on chainB
	msgBankSend := &banktypes.MsgSend{
		FromAddress: s.chainB.SenderAccount.GetAddress().String(),
		ToAddress:   interchainAccountAddr,
		Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100000))),
	}
	res, err := s.chainB.SendMsgs(msgBankSend)
	s.Require().NotEmpty(res)
	s.Require().NoError(err)

	return interchainAccountAddr
}

// RegisterInterchainAccount submits a MsgRegisterInterchainAccount and updates the controller endpoint with the
// channel created.
func (s *CallbacksTestSuite) RegisterInterchainAccount(owner string) {
	msgRegister := icacontrollertypes.NewMsgRegisterInterchainAccount(s.path.EndpointA.ConnectionID, owner, s.path.EndpointA.ChannelConfig.Version)

	res, err := s.chainA.SendMsgs(msgRegister)
	s.Require().NotEmpty(res)
	s.Require().NoError(err)

	channelID, err := ibctesting.ParseChannelIDFromEvents(res.Events)
	s.Require().NoError(err)

	s.path.EndpointA.ChannelID = channelID
}

// AssertHasExecutedExpectedCallback checks if only the expected type of callback has been executed.
// It assumes that the source chain is chainA and the destination chain is chainB.
//
// The callbackType can be one of the following:
//   - types.CallbackTypeAcknowledgement
//   - types.CallbackTypeWriteAcknowledgement
//   - types.CallbackTypeTimeout
//   - "none" (no callback should be executed)
func (s *CallbacksTestSuite) AssertHasExecutedExpectedCallback(callbackType types.CallbackType, isSuccessful bool) {
	successCount := uint64(0)
	if isSuccessful {
		successCount = 1
	}
	switch callbackType {
	case types.CallbackTypeAcknowledgement:
		s.Require().Equal(successCount, s.chainA.GetSimApp().MockKeeper.AckCallbackCounter.Success)
		s.Require().Equal(1-successCount, s.chainA.GetSimApp().MockKeeper.AckCallbackCounter.Failure)
		s.Require().Equal(successCount, s.chainA.GetSimApp().MockKeeper.SendPacketCallbackCounter.Success)
		s.Require().Equal(1-successCount, s.chainA.GetSimApp().MockKeeper.SendPacketCallbackCounter.Failure)
		s.Require().Equal(uint8(2*successCount), s.chainA.GetSimApp().MockKeeper.GetStateCounter(s.chainA.GetContext()))
		s.Require().Equal(uint8(0), s.chainB.GetSimApp().MockKeeper.GetStateCounter(s.chainB.GetContext()))
		s.Require().True(s.chainA.GetSimApp().MockKeeper.TimeoutCallbackCounter.IsZero())
		s.Require().True(s.chainB.GetSimApp().MockKeeper.WriteAcknowledgementCallbackCounter.IsZero())
	case types.CallbackTypeWriteAcknowledgement:
		s.Require().Equal(successCount, s.chainB.GetSimApp().MockKeeper.WriteAcknowledgementCallbackCounter.Success)
		s.Require().Equal(1-successCount, s.chainB.GetSimApp().MockKeeper.WriteAcknowledgementCallbackCounter.Failure)
		s.Require().Equal(uint8(successCount), s.chainB.GetSimApp().MockKeeper.GetStateCounter(s.chainB.GetContext()))
		s.Require().Equal(uint8(0), s.chainA.GetSimApp().MockKeeper.GetStateCounter(s.chainA.GetContext()))
		s.Require().True(s.chainA.GetSimApp().MockKeeper.SendPacketCallbackCounter.IsZero())
		s.Require().True(s.chainA.GetSimApp().MockKeeper.TimeoutCallbackCounter.IsZero())
		s.Require().True(s.chainB.GetSimApp().MockKeeper.AckCallbackCounter.IsZero())
	case types.CallbackTypeTimeoutPacket:
		s.Require().Equal(successCount, s.chainA.GetSimApp().MockKeeper.TimeoutCallbackCounter.Success)
		s.Require().Equal(1-successCount, s.chainA.GetSimApp().MockKeeper.TimeoutCallbackCounter.Failure)
		s.Require().Equal(successCount, s.chainA.GetSimApp().MockKeeper.SendPacketCallbackCounter.Success)
		s.Require().Equal(1-successCount, s.chainA.GetSimApp().MockKeeper.SendPacketCallbackCounter.Failure)
		s.Require().Equal(uint8(2*successCount), s.chainA.GetSimApp().MockKeeper.GetStateCounter(s.chainA.GetContext()))
		s.Require().Equal(uint8(0), s.chainB.GetSimApp().MockKeeper.GetStateCounter(s.chainB.GetContext()))
		s.Require().True(s.chainA.GetSimApp().MockKeeper.AckCallbackCounter.IsZero())
		s.Require().True(s.chainB.GetSimApp().MockKeeper.WriteAcknowledgementCallbackCounter.IsZero())
	case "none":
		s.Require().True(s.chainA.GetSimApp().MockKeeper.AckCallbackCounter.IsZero())
		s.Require().True(s.chainA.GetSimApp().MockKeeper.TimeoutCallbackCounter.IsZero())
		s.Require().True(s.chainB.GetSimApp().MockKeeper.WriteAcknowledgementCallbackCounter.IsZero())
		s.Require().True(s.chainA.GetSimApp().MockKeeper.SendPacketCallbackCounter.IsZero())
		s.Require().Equal(uint8(0), s.chainA.GetSimApp().MockKeeper.GetStateCounter(s.chainA.GetContext()))
		s.Require().Equal(uint8(0), s.chainB.GetSimApp().MockKeeper.GetStateCounter(s.chainB.GetContext()))
	default:
		s.FailNow(fmt.Sprintf("invalid callback type %s", callbackType))
	}
	s.Require().True(s.chainB.GetSimApp().MockKeeper.AckCallbackCounter.IsZero())
	s.Require().True(s.chainB.GetSimApp().MockKeeper.TimeoutCallbackCounter.IsZero())
	s.Require().True(s.chainA.GetSimApp().MockKeeper.WriteAcknowledgementCallbackCounter.IsZero())
}

func TestIBCCallbacksTestSuite(t *testing.T) {
	suite.Run(t, new(CallbacksTestSuite))
}

// AssertHasExecutedExpectedCallbackWithFee checks if only the expected type of callback has been executed
// and that the expected ics-29 fee has been paid.
func (s *CallbacksTestSuite) AssertHasExecutedExpectedCallbackWithFee(
	callbackType types.CallbackType, isSuccessful bool, isTimeout bool,
	originalSenderBalance sdk.Coins, fee feetypes.Fee,
) {
	// Recall that:
	// - the source chain is chainA
	// - forward relayer is chainB.SenderAccount
	// - reverse relayer is chainA.SenderAccount
	// - The counterparty payee of the forward relayer in chainA is chainB.SenderAccount (as a chainA account)

	if !isTimeout {
		// check forward relay balance
		s.Require().Equal(
			fee.RecvFee,
			sdk.NewCoins(s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainB.SenderAccount.GetAddress(), ibctesting.TestCoin.Denom)),
		)

		s.Require().Equal(
			fee.AckFee.Add(fee.TimeoutFee...), // ack fee paid, timeout fee refunded
			sdk.NewCoins(
				s.chainA.GetSimApp().BankKeeper.GetBalance(
					s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(),
					ibctesting.TestCoin.Denom),
			).Sub(originalSenderBalance[0]),
		)
	} else {
		// forwad relay balance should be 0
		s.Require().Equal(
			sdk.NewCoin(ibctesting.TestCoin.Denom, sdkmath.ZeroInt()),
			s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainB.SenderAccount.GetAddress(), ibctesting.TestCoin.Denom),
		)

		// all fees should be returned as sender is the reverse relayer
		s.Require().Equal(
			fee.Total(),
			sdk.NewCoins(
				s.chainA.GetSimApp().BankKeeper.GetBalance(
					s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(),
					ibctesting.TestCoin.Denom),
			).Sub(originalSenderBalance[0]),
		)
	}
	s.AssertHasExecutedExpectedCallback(callbackType, isSuccessful)
}
