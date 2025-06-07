package v2_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/suite"

	"cosmossdk.io/log"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/callbacks/testing/simapp"
	"github.com/cosmos/ibc-go/v10/modules/apps/callbacks/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

const maxCallbackGas = uint64(1000000)

// SetupTestingApp provides the duplicated simapp which is specific to the callbacks module on chain creation.
func SetupTestingApp() (ibctesting.TestingApp, map[string]json.RawMessage) {
	db := dbm.NewMemDB()
	app := simapp.NewSimApp(log.NewNopLogger(), db, nil, true, simtestutil.AppOptionsMap{})
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
	s.coordinator = ibctesting.NewCustomAppCoordinator(s.T(), 2, SetupTestingApp)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
	s.path = ibctesting.NewPath(s.chainA, s.chainB)
}

// SetupTransferTest sets up a IBC v2 path between chainA and chainB
func (s *CallbacksTestSuite) SetupTest() {
	s.setupChains()

	s.path.SetupV2()
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
		s.Require().Empty(sourceCounters)
		s.Require().Empty(destCounters)

	case types.CallbackTypeSendPacket:
		s.Require().Len(sourceCounters, 1)
		s.Require().Equal(1, sourceCounters[types.CallbackTypeSendPacket])

	case types.CallbackTypeAcknowledgementPacket:
		s.Require().Len(sourceCounters, 2)
		s.Require().Equal(1, sourceCounters[types.CallbackTypeSendPacket])
		s.Require().Equal(1, sourceCounters[types.CallbackTypeAcknowledgementPacket])

		s.Require().Empty(destCounters)

	case types.CallbackTypeReceivePacket:
		s.Require().Empty(sourceCounters)
		s.Require().Len(destCounters, 1)
		s.Require().Equal(1, destCounters[types.CallbackTypeReceivePacket])

	case types.CallbackTypeTimeoutPacket:
		s.Require().Len(sourceCounters, 2)
		s.Require().Equal(1, sourceCounters[types.CallbackTypeSendPacket])
		s.Require().Equal(1, sourceCounters[types.CallbackTypeTimeoutPacket])

		s.Require().Empty(destCounters)

	default:
		s.FailNow(fmt.Sprintf("invalid callback type %s", callbackType))
	}
}

// GetExpectedEvent returns the expected event for a callback.
func GetExpectedEvent(
	ctx sdk.Context, packetData any, remainingGas uint64, version string,
	eventPortID, eventChannelID string, seq uint64, callbackType types.CallbackType, expError error,
) (abci.Event, bool) {
	callbackKey := types.SourceCallbackKey
	if callbackType == types.CallbackTypeReceivePacket {
		callbackKey = types.DestinationCallbackKey
	}
	callbackData, isCbPacket, err := types.GetCallbackData(packetData, version, eventPortID, remainingGas, maxCallbackGas, callbackKey)
	if !isCbPacket || err != nil {
		return abci.Event{}, false
	}

	newCtx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	types.EmitCallbackEvent(newCtx, eventPortID, eventChannelID, seq, callbackType, callbackData, expError)
	return newCtx.EventManager().Events().ToABCIEvents()[0], true
}

func TestIBCCallbacksTestSuite(t *testing.T) {
	suite.Run(t, new(CallbacksTestSuite))
}
