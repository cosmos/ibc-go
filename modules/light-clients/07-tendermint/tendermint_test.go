package tendermint_test

import (
	"testing"
	"time"

	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	cmttypes "github.com/cometbft/cometbft/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/cosmos/ibc-go/v10/testing/simapp"
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
	height             = clienttypes.NewHeight(0, 4)
	newClientHeight    = clienttypes.NewHeight(1, 1)
	upgradePath        = []string{"upgrade", "upgradedIBCState"}
	invalidUpgradePath = []string{"upgrade", ""}
)

type TendermintTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	// TODO: deprecate usage in favor of testing package
	ctx        sdk.Context
	cdc        codec.Codec
	privVal    cmttypes.PrivValidator
	valSet     *cmttypes.ValidatorSet
	signers    map[string]cmttypes.PrivValidator
	valsHash   cmtbytes.HexBytes
	header     *ibctm.Header
	now        time.Time
	headerTime time.Time
	clientTime time.Time
}

func TestTendermintTestSuite(t *testing.T) {
	testifysuite.Run(t, new(TendermintTestSuite))
}

func (s *TendermintTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	s.coordinator.CommitNBlocks(s.chainA, 2)
	s.coordinator.CommitNBlocks(s.chainB, 2)

	// TODO: deprecate usage in favor of testing package
	checkTx := false
	app := simapp.Setup(s.T(), checkTx)

	s.cdc = app.AppCodec()

	// now is the time of the current chain, must be after the updating header
	// mocks ctx.BlockTime()
	s.now = time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	s.clientTime = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	// Header time is intended to be time for any new header used for updates
	s.headerTime = time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)

	s.privVal = cmttypes.NewMockPV()

	pubKey, err := s.privVal.GetPubKey()
	s.Require().NoError(err)

	heightMinus1 := clienttypes.NewHeight(0, height.RevisionHeight-1)

	val := cmttypes.NewValidator(pubKey, 10)
	s.signers = make(map[string]cmttypes.PrivValidator)
	s.signers[val.Address.String()] = s.privVal
	s.valSet = cmttypes.NewValidatorSet([]*cmttypes.Validator{val})
	s.valsHash = s.valSet.Hash()
	s.header = s.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), heightMinus1, s.now, s.valSet, s.valSet, s.valSet, s.signers)
	s.ctx = app.NewContext(checkTx)
}

func getAltSigners(altVal *cmttypes.Validator, altPrivVal cmttypes.PrivValidator) map[string]cmttypes.PrivValidator {
	return map[string]cmttypes.PrivValidator{altVal.Address.String(): altPrivVal}
}

func getBothSigners(s *TendermintTestSuite, altVal *cmttypes.Validator, altPrivVal cmttypes.PrivValidator) (*cmttypes.ValidatorSet, map[string]cmttypes.PrivValidator) {
	// Create bothValSet with both suite validator and altVal. Would be valid update
	bothValSet := cmttypes.NewValidatorSet(append(s.valSet.Validators, altVal))
	// Create signer array and ensure it is in same order as bothValSet
	_, suiteVal := s.valSet.GetByIndex(0)
	bothSigners := map[string]cmttypes.PrivValidator{
		suiteVal.Address.String(): s.privVal,
		altVal.Address.String():   altPrivVal,
	}
	return bothValSet, bothSigners
}
