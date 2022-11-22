package types_test

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"

	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
	ibctmtypes "github.com/cosmos/ibc-go/v3/modules/light-clients/01-dymint/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
	ibctestingmock "github.com/cosmos/ibc-go/v3/testing/mock"
	"github.com/cosmos/ibc-go/v3/testing/simapp"
)

const (
	chainID                        = "gaia"
	chainIDRevision0               = "gaia-revision-0"
	chainIDRevision1               = "gaia-revision-1"
	clientID                       = "gaiamainnet"
	trustingPeriod   time.Duration = time.Hour * 24 * 7 * 2
	ubdPeriod        time.Duration = time.Hour * 24 * 7 * 3
	maxClockDrift    time.Duration = time.Second * 10
)

var (
	height          = clienttypes.NewHeight(0, 4)
	newClientHeight = clienttypes.NewHeight(1, 1)
	upgradePath     = []string{"upgrade", "upgradedIBCState"}
)

type DymintTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	// TODO: deprecate usage in favor of testing package
	ctx        sdk.Context
	cdc        codec.Codec
	privVal    tmtypes.PrivValidator
	valSet     *tmtypes.ValidatorSet
	valsHash   tmbytes.HexBytes
	header     *ibctmtypes.Header
	now        time.Time
	headerTime time.Time
	clientTime time.Time

	// consensus setup
	chainAConsensusType string
	chainBConsensusType string
}

func (suite *DymintTestSuite) SetupTest() {
	//suite.SetupTestWithConsensusType(exported.Dymint, exported.Tendermint)
	suite.SetupTestWithConsensusType(suite.chainAConsensusType, suite.chainBConsensusType)
}

func (suite *DymintTestSuite) SetupTestWithConsensusType(chainAConsensusType string, chainBConsensusType string) {
	suite.Require().True(chainAConsensusType == exported.Dymint || chainBConsensusType == exported.Dymint)
	suite.Require().True(chainAConsensusType == exported.Dymint || chainAConsensusType == exported.Tendermint)
	suite.Require().True(chainBConsensusType == exported.Dymint || chainBConsensusType == exported.Tendermint)

	suite.coordinator = ibctesting.NewCoordinatorWithConsensusType(suite.T(), []string{chainAConsensusType, chainBConsensusType})
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)
	suite.coordinator.CommitNBlocks(suite.chainB, 2)

	// TODO: deprecate usage in favor of testing package
	checkTx := false
	app := simapp.Setup(checkTx)

	suite.cdc = app.AppCodec()

	// now is the time of the current chain, must be after the updating header
	// mocks ctx.BlockTime()
	suite.now = time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	suite.clientTime = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	// Header time is intended to be time for any new header used for updates
	suite.headerTime = time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)

	suite.privVal = ibctestingmock.NewPV()

	pubKey, err := suite.privVal.GetPubKey()
	suite.Require().NoError(err)

	heightMinus1 := clienttypes.NewHeight(0, height.RevisionHeight-1)

	val := tmtypes.NewValidator(pubKey, 10)
	suite.valSet = tmtypes.NewValidatorSet([]*tmtypes.Validator{val})
	suite.valsHash = suite.valSet.Hash()
	if chainAConsensusType == exported.Tendermint {
		chainBDymint := suite.chainB.TestChainClient.(*ibctesting.TestChainDymint)
		suite.header = chainBDymint.CreateDMClientHeader(chainID, int64(height.RevisionHeight), heightMinus1, suite.now, suite.valSet, suite.valSet, []tmtypes.PrivValidator{suite.privVal})
	} else {
		// chainA must be Dymint
		chainADymint := suite.chainA.TestChainClient.(*ibctesting.TestChainDymint)
		suite.header = chainADymint.CreateDMClientHeader(chainID, int64(height.RevisionHeight), heightMinus1, suite.now, suite.valSet, suite.valSet, []tmtypes.PrivValidator{suite.privVal})
	}

	suite.ctx = app.BaseApp.NewContext(checkTx, tmproto.Header{Height: 1, Time: suite.now})
}

func getSuiteSigners(suite *DymintTestSuite) []tmtypes.PrivValidator {
	return []tmtypes.PrivValidator{suite.privVal}
}

func getBothSigners(suite *DymintTestSuite, altVal *tmtypes.Validator, altPrivVal tmtypes.PrivValidator) (*tmtypes.ValidatorSet, []tmtypes.PrivValidator) {
	// Create bothValSet with both suite validator and altVal. Would be valid update
	bothValSet := tmtypes.NewValidatorSet(append(suite.valSet.Validators, altVal))
	// Create signer array and ensure it is in same order as bothValSet
	_, suiteVal := suite.valSet.GetByIndex(0)
	bothSigners := ibctesting.CreateSortedSignerArray(altPrivVal, suite.privVal, altVal, suiteVal)
	return bothValSet, bothSigners
}

func TestDymintTestSuiteDymTm(t *testing.T) {
	suite.Run(t, &DymintTestSuite{
		chainAConsensusType: exported.Dymint,
		chainBConsensusType: exported.Tendermint,
	})
}

func TestDymintTestSuiteTmDym(t *testing.T) {
	suite.Run(t, &DymintTestSuite{
		chainAConsensusType: exported.Tendermint,
		chainBConsensusType: exported.Dymint,
	})
}

func TestDymintTestSuiteDymDym(t *testing.T) {
	suite.Run(t, &DymintTestSuite{
		chainAConsensusType: exported.Dymint,
		chainBConsensusType: exported.Dymint,
	})
}
