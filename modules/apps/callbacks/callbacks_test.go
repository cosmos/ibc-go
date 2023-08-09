package ibccallbacks_test

import (
	"encoding/json"
	"fmt"
	"testing"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	icacontroller "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller"
	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icahost "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	ibcfee "github.com/cosmos/ibc-go/v7/modules/apps/29-fee"
	feetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	ibccallbacks "github.com/cosmos/ibc-go/v7/modules/apps/callbacks"
	"github.com/cosmos/ibc-go/v7/modules/apps/callbacks/types"
	"github.com/cosmos/ibc-go/v7/modules/apps/transfer"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
	simapp "github.com/cosmos/ibc-go/v7/testing/simapp"
)

//func init() {
//	ibctesting.DefaultTestingAppInit = SetupCallbacksTestingApp
//}

func (s *CallbacksTestSuite) SetupCallbacksTestingApp() (ibctesting.TestingApp, map[string]json.RawMessage) {
	db := dbm.NewMemDB()
	encCdc := simapp.MakeTestEncodingConfig()

	var mockContractKeeper ibcmock.ContractKeeper

	app := simapp.NewSimappWithOptions(log.NewNopLogger(), db, nil, true, simtestutil.EmptyAppOptions{}, func(app *simapp.SimApp) {

		// Mock Module Stack
		router := porttypes.NewRouter()
		mockContractKeeper = ibcmock.NewContractKeeper(app.GetMemKey(ibcmock.MemStoreKey))

		var transferStack porttypes.IBCModule
		transferStack = transfer.NewIBCModule(app.TransferKeeper)
		transferStack = ibcfee.NewIBCMiddleware(transferStack, app.IBCFeeKeeper)
		transferStack = ibccallbacks.NewIBCMiddleware(transferStack, app.IBCFeeKeeper, mockContractKeeper, maxCallbackGas)
		// Since the callbacks middleware itself is an ics4wrapper, it needs to be passed to the transfer keeper
		app.TransferKeeper.WithICS4Wrapper(transferStack.(porttypes.Middleware))
		router.AddRoute(transfertypes.ModuleName, transferStack)

		mockModule := ibcmock.NewAppModule(&app.IBCKeeper.PortKeeper)

		// The mock module is used for testing IBC
		mockIBCModule := ibcmock.NewIBCModule(&mockModule, ibcmock.NewIBCApp(ibcmock.ModuleName, app.ScopedIBCMockKeeper))
		router.AddRoute(ibcmock.ModuleName, mockIBCModule)

		// initialize ICA module with mock module as the authentication module on the controller side
		var icaControllerStack porttypes.IBCModule
		icaControllerStack = ibcmock.NewIBCModule(&mockModule, ibcmock.NewIBCApp("", app.ScopedICAControllerKeeper))
		app.ICAAuthModule = icaControllerStack.(ibcmock.IBCModule)
		icaControllerStack = icacontroller.NewIBCMiddleware(icaControllerStack, app.ICAControllerKeeper)
		icaControllerStack = ibcfee.NewIBCMiddleware(icaControllerStack, app.IBCFeeKeeper)
		icaControllerStack = ibccallbacks.NewIBCMiddleware(icaControllerStack, app.IBCFeeKeeper, mockContractKeeper, maxCallbackGas)
		// Since the callbacks middleware itself is an ics4wrapper, it needs to be passed to the ica controller keeper
		app.ICAControllerKeeper.WithICS4Wrapper(icaControllerStack.(porttypes.Middleware))

		// RecvPacket, message that originates from core IBC and goes down to app, the flow is:
		// channel.RecvPacket -> callbacks.OnRecvPacket -> fee.OnRecvPacket -> icaHost.OnRecvPacket

		var icaHostStack porttypes.IBCModule
		icaHostStack = icahost.NewIBCModule(app.ICAHostKeeper)
		icaHostStack = ibcfee.NewIBCMiddleware(icaHostStack, app.IBCFeeKeeper)

		// Add host, controller & ica auth modules to IBC router
		router.
			// the ICA Controller middleware needs to be explicitly added to the IBC Router because the
			// ICA controller module owns the port capability for ICA. The ICA authentication module
			// owns the channel capability.
			AddRoute(icacontrollertypes.SubModuleName, icaControllerStack).
			AddRoute(icahosttypes.SubModuleName, icaHostStack).
			AddRoute(ibcmock.ModuleName+icacontrollertypes.SubModuleName, icaControllerStack) // ica with mock auth module stack route to ica (top level of middleware stack)

		// Create Mock IBC Fee module stack for testing
		// SendPacket, since it is originating from the application to core IBC:
		// mockModule.SendPacket -> fee.SendPacket -> channel.SendPacket

		// OnRecvPacket, message that originates from core IBC and goes down to app, the flow is the otherway
		// channel.RecvPacket -> fee.OnRecvPacket -> mockModule.OnRecvPacket

		// OnAcknowledgementPacket as this is where fee's are paid out
		// mockModule.OnAcknowledgementPacket -> fee.OnAcknowledgementPacket -> channel.OnAcknowledgementPacket

		// create fee wrapped mock module
		feeMockModule := ibcmock.NewIBCModule(&mockModule, ibcmock.NewIBCApp(ibctesting.MockFeePort, app.ScopedFeeMockKeeper))
		app.FeeMockModule = feeMockModule
		var feeWithMockModule porttypes.Middleware = ibcfee.NewIBCMiddleware(feeMockModule, app.IBCFeeKeeper)
		feeWithMockModule = ibccallbacks.NewIBCMiddleware(feeWithMockModule, app.IBCFeeKeeper, mockContractKeeper, maxCallbackGas)
		router.AddRoute(ibctesting.MockFeePort, feeWithMockModule)

		app.IBCKeeper.Router = nil
		app.IBCKeeper.SetRouter(router)
	})

	return callbacksSimapp{
		SimApp:         app,
		ContractKeeper: mockContractKeeper,
	}, simapp.NewDefaultGenesisState(encCdc.Codec)
}

type callbacksSimapp struct {
	*simapp.SimApp
	ContractKeeper ibcmock.ContractKeeper
}

const maxCallbackGas = uint64(1000000)

type callbacksTestChain struct {
	*ibctesting.TestChain
	ContractKeeper ibcmock.ContractKeeper
}

// CallbacksTestSuite defines the needed instances and methods to test callbacks
type CallbacksTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	chainA *callbacksTestChain
	chainB *callbacksTestChain

	path *ibctesting.Path
}

// setupChains sets up a coordinator with 2 test chains.
func (s *CallbacksTestSuite) setupChains() {
	ibctesting.DefaultTestingAppInit = s.SetupCallbacksTestingApp
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)
	s.chainA = &callbacksTestChain{TestChain: s.coordinator.GetChain(ibctesting.GetChainID(1))}
	s.chainB = &callbacksTestChain{TestChain: s.coordinator.GetChain(ibctesting.GetChainID(2))}
	s.path = ibctesting.NewPath(s.chainA.TestChain, s.chainB.TestChain)
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

// AssertHasExecutedExpectedCallback checks the stateful entries and counters based on callbacktype.
// It assumes that the source chain is chainA and the destination chain is chainB.
func (s *CallbacksTestSuite) AssertHasExecutedExpectedCallback(callbackType types.CallbackTrigger, expSuccess bool) {
	var expStatefulEntries uint8
	if expSuccess {
		// if the callback is expected to be successful,
		// we expect at least one state entry
		expStatefulEntries = 1
	}

	sourceStatefulCounter := s.chainA.ContractKeeper.GetStateEntryCounter(s.chainA.GetContext())
	destStatefulCounter := s.chainB.ContractKeeper.GetStateEntryCounter(s.chainB.GetContext())

	switch callbackType {
	case "none":
		s.Require().Equal(uint8(0), sourceStatefulCounter)
		s.Require().Equal(uint8(0), destStatefulCounter)

	case types.CallbackTriggerSendPacket:
		s.Require().Equal(expStatefulEntries, sourceStatefulCounter, "unexpected stateful entry amount for source send packet callback")
		s.Require().Equal(uint8(0), destStatefulCounter)

	case types.CallbackTriggerAcknowledgementPacket, types.CallbackTriggerTimeoutPacket:
		expStatefulEntries *= 2 // expect OnAcknowledgement/OnTimeout to be successful as well
		s.Require().Equal(expStatefulEntries, sourceStatefulCounter, "unexpected stateful entry amount for source acknowledgement/timeout callbacks")
		s.Require().Equal(uint8(0), destStatefulCounter)

	case types.CallbackTriggerReceivePacket:
		s.Require().Equal(uint8(0), sourceStatefulCounter)
		s.Require().Equal(expStatefulEntries, destStatefulCounter)

	default:
		s.FailNow(fmt.Sprintf("invalid callback type %s", callbackType))
	}

	s.AssertCallbackCounters(callbackType)
}

func (s *CallbacksTestSuite) AssertCallbackCounters(callbackType types.CallbackTrigger) {
	sourceCounters := s.chainA.ContractKeeper.Counters
	destCounters := s.chainB.ContractKeeper.Counters

	switch callbackType {
	case "none":
		s.Require().Len(sourceCounters, 0)
		s.Require().Len(destCounters, 0)

	case types.CallbackTriggerSendPacket:
		s.Require().Len(sourceCounters, 1)
		s.Require().Equal(1, sourceCounters[types.CallbackTriggerSendPacket])

	case types.CallbackTriggerAcknowledgementPacket:
		s.Require().Len(sourceCounters, 2)
		s.Require().Equal(1, sourceCounters[types.CallbackTriggerSendPacket])
		s.Require().Equal(1, sourceCounters[types.CallbackTriggerAcknowledgementPacket])

		s.Require().Len(destCounters, 0)

	case types.CallbackTriggerReceivePacket:
		s.Require().Len(sourceCounters, 0)
		s.Require().Len(destCounters, 1)
		s.Require().Equal(1, destCounters[types.CallbackTriggerReceivePacket])

	case types.CallbackTriggerTimeoutPacket:
		s.Require().Len(sourceCounters, 2)
		s.Require().Equal(1, sourceCounters[types.CallbackTriggerSendPacket])
		s.Require().Equal(1, sourceCounters[types.CallbackTriggerTimeoutPacket])

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
	callbackType types.CallbackTrigger, isSuccessful bool, isTimeout bool,
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
		// forward relay balance should be 0
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
