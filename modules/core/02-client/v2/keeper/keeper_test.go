package keeper_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/keeper"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/cosmos/ibc-go/v10/testing/simapp"
)

const (
	testClientID  = "tendermint-0"
	testClientID2 = "tendermint-1"
)

type KeeperTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	cdc    codec.Codec
	ctx    sdk.Context
	keeper *keeper.Keeper
}

func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)

	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))

	isCheckTx := false
	app := simapp.Setup(s.T(), isCheckTx)

	s.cdc = app.AppCodec()
	s.ctx = app.NewContext(isCheckTx)
	s.keeper = app.IBCKeeper.ClientV2Keeper
}

func (s *KeeperTestSuite) TestSetClientCounterparty() {
	counterparty := types.NewCounterpartyInfo([][]byte{[]byte("ibc"), []byte("channel-7")}, testClientID2)
	s.keeper.SetClientCounterparty(s.ctx, testClientID, counterparty)

	retrievedCounterparty, found := s.keeper.GetClientCounterparty(s.ctx, testClientID)
	s.Require().True(found, "GetCounterparty failed")
	s.Require().Equal(counterparty, retrievedCounterparty, "Counterparties are not equal")
}

func (s *KeeperTestSuite) TestSetConfig() {
	config := s.keeper.GetConfig(s.ctx, testClientID)
	s.Require().Equal(config, types.DefaultConfig(), "did not return default config on initialization")

	newConfig := types.NewConfig(ibctesting.TestAccAddress)
	s.keeper.SetConfig(s.ctx, testClientID, newConfig)

	config = s.keeper.GetConfig(s.ctx, testClientID)
	s.Require().Equal(newConfig, config, "config not set correctly")

	// config should be empty for a different clientID
	config = s.keeper.GetConfig(s.ctx, testClientID2)
	s.Require().Equal(types.DefaultConfig(), config, "config should be empty for different clientID")

	// set config for a different clientID
	newConfig2 := types.NewConfig(ibctesting.TestAccAddress, s.chainA.SenderAccount.GetAddress().String())
	s.keeper.SetConfig(s.ctx, testClientID2, newConfig2)

	config = s.keeper.GetConfig(s.ctx, testClientID2)
	s.Require().Equal(newConfig2, config, "config not set correctly for different clientID")

	// config for original client unaffected
	config = s.keeper.GetConfig(s.ctx, testClientID)
	s.Require().Equal(newConfig, config, "config not set correctly for original clientID")
}
