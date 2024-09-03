package keeper_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/keeper"
	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

const (
	testClientID = "tendermint-0"
)

var (
	defaultTimeoutHeight     = clienttypes.NewHeight(1, 100)
	disabledTimeoutTimestamp = uint64(0)
)

// KeeperTestSuite is a testing suite to test keeper functions.
type KeeperTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	ctx    sdk.Context
	keeper *keeper.Keeper
}

// TestKeeperTestSuite runs all the tests within this package.
func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}

// SetupTest creates a coordinator with 2 test chains.
func (suite *KeeperTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))

	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)
	suite.coordinator.CommitNBlocks(suite.chainB, 2)

	suite.ctx = suite.chainA.GetContext()
	suite.keeper = suite.chainA.App.GetPacketServer()
}

func (suite *KeeperTestSuite) TestSetCounterparty() {
	merklePathPrefix := commitmenttypes.NewMerklePath([]byte("ibc"), []byte(""))
	counterparty := types.Counterparty{
		ClientId:               testClientID,
		CounterpartyPacketPath: merklePathPrefix,
	}
	suite.keeper.SetCounterparty(suite.ctx, testClientID, counterparty)

	retrievedCounterparty, found := suite.keeper.GetCounterparty(suite.ctx, testClientID)
	suite.Require().True(found, "GetCounterparty does not return counterparty")
	suite.Require().Equal(counterparty, retrievedCounterparty, "Counterparty retrieved not equal")

	retrievedCounterparty, found = suite.keeper.GetCounterparty(suite.ctx, "client-0")
	suite.Require().False(found, "GetCounterparty unexpectedly returned a counterparty")
	suite.Require().Equal(types.Counterparty{}, retrievedCounterparty, "Counterparty retrieved not empty")
}
