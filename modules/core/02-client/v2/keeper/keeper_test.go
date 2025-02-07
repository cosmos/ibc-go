package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	stakingtypes "cosmossdk.io/x/staking/types"
	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/codec"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v9/modules/core/02-client/v2/keeper"
	types2 "github.com/cosmos/ibc-go/v9/modules/core/02-client/v2/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/ibc-go/v9/testing/simapp"
	testifysuite "github.com/stretchr/testify/suite"
	"math/rand"
	"testing"
	"time"
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

	cdc            codec.Codec
	ctx            sdk.Context
	keeper         *keeper.Keeper
	consensusState *ibctm.ConsensusState
	valSet         *cmttypes.ValidatorSet
	valSetHash     cmtbytes.HexBytes
	privVal        cmttypes.PrivValidator
	now            time.Time
	past           time.Time
	solomachine    *ibctesting.Solomachine

	signers map[string]cmttypes.PrivValidator
}

func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)

	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))

	isCheckTx := false
	suite.now = time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	suite.past = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	app := simapp.Setup(suite.T(), isCheckTx)

	suite.cdc = app.AppCodec()
	suite.ctx = app.BaseApp.NewContext(isCheckTx)
	suite.keeper = app.IBCKeeper.ClientV2Keeper
	suite.privVal = cmttypes.NewMockPV()
	pubKey, err := suite.privVal.GetPubKey()
	suite.Require().NoError(err)

	validator := cmttypes.NewValidator(pubKey, 1)
	suite.valSet = cmttypes.NewValidatorSet([]*cmttypes.Validator{validator})
	suite.valSetHash = suite.valSet.Hash()

	suite.signers = make(map[string]cmttypes.PrivValidator, 1)
	suite.signers[validator.Address.String()] = suite.privVal

	suite.consensusState = ibctm.NewConsensusState(suite.now, commitmenttypes.NewMerkleRoot([]byte("hash")), suite.valSetHash)

	var validators stakingtypes.Validators
	for i := 1; i < 11; i++ {
		privVal := cmttypes.NewMockPV()
		tmPk, err := privVal.GetPubKey()
		suite.Require().NoError(err)
		pk, err := cryptocodec.FromCmtPubKeyInterface(tmPk)
		suite.Require().NoError(err)
		val, err := stakingtypes.NewValidator(pk.Address().String(), pk, stakingtypes.Description{})
		suite.Require().NoError(err)

		val.Status = stakingtypes.Bonded
		val.Tokens = sdkmath.NewInt(rand.Int63())
		validators.Validators = append(validators.Validators, val)

	}

	suite.solomachine = ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachinesingle", "testing", 1)
}

func (suite *KeeperTestSuite) TestSetClientCounterparty() {
	counterparty := types2.NewCounterpartyInfo([][]byte{[]byte("ibc"), []byte("channel-7")}, testClientID2)
	suite.keeper.SetClientCounterparty(suite.ctx, testClientID, counterparty)

	retrievedCounterparty, found := suite.keeper.GetClientCounterparty(suite.ctx, testClientID)
	suite.Require().True(found, "GetCounterparty failed")
	suite.Require().Equal(counterparty, retrievedCounterparty, "Counterparties are not equal")
}
