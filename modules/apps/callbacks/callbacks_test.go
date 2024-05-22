package ibccallbacks_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/suite"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/ibc-go/modules/apps/callbacks/testing/simapp"
	"github.com/cosmos/ibc-go/modules/apps/callbacks/types"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	feetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

const maxCallbackGas = uint64(1000000)

func init() {
	ibctesting.DefaultTestingAppInit = SetupTestingApp
}

// SetupTestingApp provides the duplicated simapp which is specific to the callbacks module on chain creation.
func SetupTestingApp() (ibctesting.TestingApp, map[string]json.RawMessage) {
	db := dbm.NewMemDB()
	app := simapp.NewSimApp(log.NewNopLogger(), db, nil, true, simtestutil.EmptyAppOptions{})
	return app, app.DefaultGenesis()
}

// GetSimApp returns the duplicated SimApp from within the callbacks directory.
// This must be used instead of chain.GetSimApp() for tests within this directory.
func GetSimApp(chain *ibctesting.TestChain) *simapp.SimApp {
	app, ok := chain.App.(*simapp.SimApp)
	if !ok {
		panic(errors.New("chain is not a simapp.SimApp"))
	}
	return app
}

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
	s.path.EndpointA.ChannelConfig.Version = transfertypes.V2
	s.path.EndpointB.ChannelConfig.Version = transfertypes.V2

	s.path.Setup()
}

// SetupFeeTransferTest sets up a fee middleware enabled transfer channel between chainA and chainB
func (s *CallbacksTestSuite) SetupFeeTransferTest() {
	s.setupChains()

	feeTransferVersion := string(feetypes.ModuleCdc.MustMarshalJSON(&feetypes.Metadata{FeeVersion: feetypes.Version, AppVersion: transfertypes.V2}))
	s.path.EndpointA.ChannelConfig.Version = feeTransferVersion
	s.path.EndpointB.ChannelConfig.Version = feeTransferVersion
	s.path.EndpointA.ChannelConfig.PortID = transfertypes.PortID
	s.path.EndpointB.ChannelConfig.PortID = transfertypes.PortID

	s.path.Setup()
}

func (s *CallbacksTestSuite) SetupMockFeeTest() {
	s.setupChains()

	ibctesting.EnableFeeOnPath(s.path)
}

// SetupICATest sets up an interchain accounts channel between chainA (controller) and chainB (host).
// It funds and returns the interchain account address owned by chainA's SenderAccount.
func (s *CallbacksTestSuite) SetupICATest() string {
	s.setupChains()
	s.path.SetupConnections()

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

	interchainAccountAddr, found := GetSimApp(s.chainB).ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), s.path.EndpointA.ConnectionID, s.path.EndpointA.ChannelConfig.PortID)
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
	msgRegister := icacontrollertypes.NewMsgRegisterInterchainAccount(s.path.EndpointA.ConnectionID, owner, s.path.EndpointA.ChannelConfig.Version, channeltypes.ORDERED)

	res, err := s.chainA.SendMsgs(msgRegister)
	s.Require().NotEmpty(res)
	s.Require().NoError(err)

	channelID, err := ibctesting.ParseChannelIDFromEvents(res.Events)
	s.Require().NoError(err)

	s.path.EndpointA.ChannelID = channelID
}

// AssertHasExecutedExpectedCallback checks the stateful entries and counters based on callbacktype.
// It assumes that the source chain is chainA and the destination chain is chainB.
func (s *CallbacksTestSuite) AssertHasExecutedExpectedCallback(callbackType types.CallbackType, expSuccess bool) {
	var expStatefulEntries uint8
	if expSuccess {
		// if the callback is expected to be successful,
		// we expect at least one state entry
		expStatefulEntries = 1
	}

	sourceStatefulCounter := GetSimApp(s.chainA).MockContractKeeper.GetStateEntryCounter(s.chainA.GetContext())
	destStatefulCounter := GetSimApp(s.chainB).MockContractKeeper.GetStateEntryCounter(s.chainB.GetContext())

	switch callbackType {
	case "none":
		s.Require().Equal(uint8(0), sourceStatefulCounter)
		s.Require().Equal(uint8(0), destStatefulCounter)

	case types.CallbackTypeSendPacket:
		s.Require().Equal(expStatefulEntries, sourceStatefulCounter, "unexpected stateful entry amount for source send packet callback")
		s.Require().Equal(uint8(0), destStatefulCounter)

	case types.CallbackTypeAcknowledgementPacket, types.CallbackTypeTimeoutPacket:
		expStatefulEntries *= 2 // expect OnAcknowledgement/OnTimeout to be successful as well as the initial SendPacket
		s.Require().Equal(expStatefulEntries, sourceStatefulCounter, "unexpected stateful entry amount for source acknowledgement/timeout callbacks")
		s.Require().Equal(uint8(0), destStatefulCounter)

	case types.CallbackTypeReceivePacket:
		s.Require().Equal(uint8(0), sourceStatefulCounter)
		s.Require().Equal(expStatefulEntries, destStatefulCounter)

	default:
		s.FailNow(fmt.Sprintf("invalid callback type %s", callbackType))
	}

	s.AssertCallbackCounters(callbackType)
}

func (s *CallbacksTestSuite) AssertCallbackCounters(callbackType types.CallbackType) {
	sourceCounters := GetSimApp(s.chainA).MockContractKeeper.Counters
	destCounters := GetSimApp(s.chainB).MockContractKeeper.Counters

	switch callbackType {
	case "none":
		s.Require().Len(sourceCounters, 0)
		s.Require().Len(destCounters, 0)

	case types.CallbackTypeSendPacket:
		s.Require().Len(sourceCounters, 1)
		s.Require().Equal(1, sourceCounters[types.CallbackTypeSendPacket])

	case types.CallbackTypeAcknowledgementPacket:
		s.Require().Len(sourceCounters, 2)
		s.Require().Equal(1, sourceCounters[types.CallbackTypeSendPacket])
		s.Require().Equal(1, sourceCounters[types.CallbackTypeAcknowledgementPacket])

		s.Require().Len(destCounters, 0)

	case types.CallbackTypeReceivePacket:
		s.Require().Len(sourceCounters, 0)
		s.Require().Len(destCounters, 1)
		s.Require().Equal(1, destCounters[types.CallbackTypeReceivePacket])

	case types.CallbackTypeTimeoutPacket:
		s.Require().Len(sourceCounters, 2)
		s.Require().Equal(1, sourceCounters[types.CallbackTypeSendPacket])
		s.Require().Equal(1, sourceCounters[types.CallbackTypeTimeoutPacket])

		s.Require().Len(destCounters, 0)

	default:
		s.FailNow(fmt.Sprintf("invalid callback type %s", callbackType))
	}
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

	// We only check if the fee is paid if the callback is successful.
	if !isTimeout && isSuccessful {
		// check forward relay balance
		s.Require().Equal(
			fee.RecvFee,
			sdk.NewCoins(GetSimApp(s.chainA).BankKeeper.GetBalance(s.chainA.GetContext(), s.chainB.SenderAccount.GetAddress(), ibctesting.TestCoin.Denom)),
		)

		refundCoins := fee.Total().Sub(fee.RecvFee...).Sub(fee.AckFee...)
		s.Require().Equal(
			fee.AckFee.Add(refundCoins...), // ack fee paid, and refund processed
			sdk.NewCoins(
				GetSimApp(s.chainA).BankKeeper.GetBalance(
					s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(),
					ibctesting.TestCoin.Denom),
			).Sub(originalSenderBalance[0]),
		)
	} else if isSuccessful {
		// forward relay balance should be 0
		s.Require().Equal(
			sdk.NewCoin(ibctesting.TestCoin.Denom, sdkmath.ZeroInt()),
			GetSimApp(s.chainA).BankKeeper.GetBalance(s.chainA.GetContext(), s.chainB.SenderAccount.GetAddress(), ibctesting.TestCoin.Denom),
		)

		// all fees should be returned as sender is the reverse relayer
		s.Require().Equal(
			fee.Total(),
			sdk.NewCoins(
				GetSimApp(s.chainA).BankKeeper.GetBalance(
					s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(),
					ibctesting.TestCoin.Denom),
			).Sub(originalSenderBalance[0]),
		)
	}
	s.AssertHasExecutedExpectedCallback(callbackType, isSuccessful)
}

// GetExpectedEvent returns the expected event for a callback.
func GetExpectedEvent(
	ctx sdk.Context, packetDataUnmarshaler porttypes.PacketDataUnmarshaler, remainingGas uint64, data []byte, srcPortID,
	eventPortID, eventChannelID string, seq uint64, callbackType types.CallbackType, expError error,
) (abci.Event, bool) {
	var (
		callbackData types.CallbackData
		err          error
	)

	// Set up gas meter with remainingGas.
	gasMeter := storetypes.NewGasMeter(remainingGas)
	ctx = ctx.WithGasMeter(gasMeter)

	if callbackType == types.CallbackTypeReceivePacket {
		packet := channeltypes.NewPacket(data, seq, "", "", eventPortID, eventChannelID, clienttypes.ZeroHeight(), 0)
		callbackData, err = types.GetDestCallbackData(ctx, packetDataUnmarshaler, packet, maxCallbackGas)
	} else {
		packet := channeltypes.NewPacket(data, seq, eventPortID, eventChannelID, "", "", clienttypes.ZeroHeight(), 0)
		callbackData, err = types.GetSourceCallbackData(ctx, packetDataUnmarshaler, packet, maxCallbackGas)
	}
	if err != nil {
		return abci.Event{}, false
	}

	newCtx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	types.EmitCallbackEvent(newCtx, eventPortID, eventChannelID, seq, callbackType, callbackData, expError)
	return newCtx.EventManager().Events().ToABCIEvents()[0], true
}
