package tendermint_test

import (
	"testing"
	"time"

	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	tmbytes "github.com/cometbft/cometbft/libs/bytes"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibctestingmock "github.com/cosmos/ibc-go/v7/testing/mock"
	"github.com/cosmos/ibc-go/v7/testing/simapp"
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

type TendermintTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	// TODO: deprecate usage in favor of testing package
	ctx        sdk.Context
	cdc        codec.Codec
	privVal    tmtypes.PrivValidator
	valSet     *tmtypes.ValidatorSet
	signers    map[string]tmtypes.PrivValidator
	valsHash   tmbytes.HexBytes
	header     *ibctm.Header
	now        time.Time
	headerTime time.Time
	clientTime time.Time
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

	s.privVal = ibctestingmock.NewPV()

	pubKey, err := s.privVal.GetPubKey()
	s.Require().NoError(err)

	heightMinus1 := clienttypes.NewHeight(0, height.RevisionHeight-1)

	val := tmtypes.NewValidator(pubKey, 10)
	s.signers = make(map[string]tmtypes.PrivValidator)
	s.signers[val.Address.String()] = s.privVal
	s.valSet = tmtypes.NewValidatorSet([]*tmtypes.Validator{val})
	s.valsHash = s.valSet.Hash()
	s.header = s.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), heightMinus1, s.now, s.valSet, s.valSet, s.valSet, s.signers)
	s.ctx = app.BaseApp.NewContext(checkTx, tmproto.Header{Height: 1, Time: s.now})
}

func getAltSigners(altVal *tmtypes.Validator, altPrivVal tmtypes.PrivValidator) map[string]tmtypes.PrivValidator {
	return map[string]tmtypes.PrivValidator{altVal.Address.String(): altPrivVal}
}

func getBothSigners(s *TendermintTestSuite, altVal *tmtypes.Validator, altPrivVal tmtypes.PrivValidator) (*tmtypes.ValidatorSet, map[string]tmtypes.PrivValidator) {
	// Create bothValSet with both suite validator and altVal. Would be valid update
	bothValSet := tmtypes.NewValidatorSet(append(s.valSet.Validators, altVal))
	// Create signer array and ensure it is in same order as bothValSet
	_, suiteVal := s.valSet.GetByIndex(0)
	bothSigners := map[string]tmtypes.PrivValidator{
		suiteVal.Address.String(): s.privVal,
		altVal.Address.String():   altPrivVal,
	}
	return bothValSet, bothSigners
}

func TestTendermintTestSuite(t *testing.T) {
	testifysuite.Run(t, new(TendermintTestSuite))
}
