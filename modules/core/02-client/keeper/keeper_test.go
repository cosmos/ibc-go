package keeper_test

import (
	"math/rand"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	tmbytes "github.com/cometbft/cometbft/libs/bytes"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/codec"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/ibc-go/v7/modules/core/02-client/keeper"
	"github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v7/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	localhost "github.com/cosmos/ibc-go/v7/modules/light-clients/09-localhost"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibctestingmock "github.com/cosmos/ibc-go/v7/testing/mock"
	"github.com/cosmos/ibc-go/v7/testing/simapp"
	"github.com/stretchr/testify/suite"
)

const (
	testChainID          = "gaiahub-0"
	testChainIDRevision1 = "gaiahub-1"

	testClientID  = "tendermint-0"
	testClientID2 = "tendermint-1"
	testClientID3 = "tendermint-2"

	height = 5

	trustingPeriod time.Duration = time.Hour * 24 * 7 * 2
	ubdPeriod      time.Duration = time.Hour * 24 * 7 * 3
	maxClockDrift  time.Duration = time.Second * 10
)

var (
	testClientHeight          = types.NewHeight(0, 5)
	testClientHeightRevision1 = types.NewHeight(1, 5)
)

type KeeperTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	cdc            codec.Codec
	ctx            sdk.Context
	keeper         *keeper.Keeper
	consensusState *ibctm.ConsensusState
	header         *ibctm.Header
	valSet         *tmtypes.ValidatorSet
	valSetHash     tmbytes.HexBytes
	privVal        tmtypes.PrivValidator
	now            time.Time
	past           time.Time
	solomachine    *ibctesting.Solomachine

	signers map[string]tmtypes.PrivValidator
}

func (s *KeeperTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)

	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))

	isCheckTx := false
	s.now = time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	s.past = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	now2 := s.now.Add(time.Hour)
	app := simapp.Setup()

	s.cdc = app.AppCodec()
	s.ctx = app.BaseApp.NewContext(isCheckTx, tmproto.Header{Height: height, ChainID: testClientID, Time: now2})
	s.keeper = &app.IBCKeeper.ClientKeeper
	s.privVal = ibctestingmock.NewPV()
	pubKey, err := s.privVal.GetPubKey()
	s.Require().NoError(err)

	testClientHeightMinus1 := types.NewHeight(0, height-1)

	validator := tmtypes.NewValidator(pubKey, 1)
	s.valSet = tmtypes.NewValidatorSet([]*tmtypes.Validator{validator})
	s.valSetHash = s.valSet.Hash()

	s.signers = make(map[string]tmtypes.PrivValidator, 1)
	s.signers[validator.Address.String()] = s.privVal

	s.header = s.chainA.CreateTMClientHeader(testChainID, int64(testClientHeight.RevisionHeight), testClientHeightMinus1, now2, s.valSet, s.valSet, s.valSet, s.signers)
	s.consensusState = ibctm.NewConsensusState(s.now, commitmenttypes.NewMerkleRoot([]byte("hash")), s.valSetHash)

	var validators stakingtypes.Validators
	for i := 1; i < 11; i++ {
		privVal := ibctestingmock.NewPV()
		tmPk, err := privVal.GetPubKey()
		s.Require().NoError(err)
		pk, err := cryptocodec.FromTmPubKeyInterface(tmPk)
		s.Require().NoError(err)
		val, err := stakingtypes.NewValidator(sdk.ValAddress(pk.Address()), pk, stakingtypes.Description{})
		s.Require().NoError(err)

		val.Status = stakingtypes.Bonded
		val.Tokens = sdkmath.NewInt(rand.Int63())
		validators = append(validators, val)

		hi := stakingtypes.NewHistoricalInfo(s.ctx.BlockHeader(), validators, sdk.DefaultPowerReduction)
		app.StakingKeeper.SetHistoricalInfo(s.ctx, int64(i), &hi)
	}

	s.solomachine = ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachinesingle", "testing", 1)
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestSetClientState() {
	clientState := ibctm.NewClientState(testChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, types.ZeroHeight(), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
	s.keeper.SetClientState(s.ctx, testClientID, clientState)

	retrievedState, found := s.keeper.GetClientState(s.ctx, testClientID)
	s.Require().True(found, "GetClientState failed")
	s.Require().Equal(clientState, retrievedState, "Client states are not equal")
}

func (s *KeeperTestSuite) TestSetClientConsensusState() {
	s.keeper.SetClientConsensusState(s.ctx, testClientID, testClientHeight, s.consensusState)

	retrievedConsState, found := s.keeper.GetClientConsensusState(s.ctx, testClientID, testClientHeight)
	s.Require().True(found, "GetConsensusState failed")

	tmConsState, ok := retrievedConsState.(*ibctm.ConsensusState)
	s.Require().True(ok)
	s.Require().Equal(s.consensusState, tmConsState, "ConsensusState not stored correctly")
}

func (s *KeeperTestSuite) TestValidateSelfClient() {
	testClientHeight := types.GetSelfHeight(s.chainA.GetContext())
	testClientHeight.RevisionHeight--

	testCases := []struct {
		name        string
		clientState exported.ClientState
		expPass     bool
	}{
		{
			"success",
			ibctm.NewClientState(s.chainA.ChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
			true,
		},
		{
			"success with nil UpgradePath",
			ibctm.NewClientState(s.chainA.ChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), nil),
			true,
		},
		{
			"frozen client",
			&ibctm.ClientState{ChainId: s.chainA.ChainID, TrustLevel: ibctm.DefaultTrustLevel, TrustingPeriod: trustingPeriod, UnbondingPeriod: ubdPeriod, MaxClockDrift: maxClockDrift, FrozenHeight: testClientHeight, LatestHeight: testClientHeight, ProofSpecs: commitmenttypes.GetSDKSpecs(), UpgradePath: ibctesting.UpgradePath},
			false,
		},
		{
			"incorrect chainID",
			ibctm.NewClientState("gaiatestnet", ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
			false,
		},
		{
			"invalid client height",
			ibctm.NewClientState(s.chainA.ChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, types.GetSelfHeight(s.chainA.GetContext()).Increment().(types.Height), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
			false,
		},
		{
			"invalid client type",
			solomachine.NewClientState(0, &solomachine.ConsensusState{PublicKey: s.solomachine.ConsensusState().PublicKey, Diversifier: s.solomachine.Diversifier, Timestamp: s.solomachine.Time}),
			false,
		},
		{
			"invalid client revision",
			ibctm.NewClientState(s.chainA.ChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, testClientHeightRevision1, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
			false,
		},
		{
			"invalid proof specs",
			ibctm.NewClientState(s.chainA.ChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, testClientHeight, nil, ibctesting.UpgradePath),
			false,
		},
		{
			"invalid trust level",
			ibctm.NewClientState(s.chainA.ChainID, ibctm.Fraction{Numerator: 0, Denominator: 1}, trustingPeriod, ubdPeriod, maxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath), false,
		},
		{
			"invalid unbonding period",
			ibctm.NewClientState(s.chainA.ChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod+10, maxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
			false,
		},
		{
			"invalid trusting period",
			ibctm.NewClientState(s.chainA.ChainID, ibctm.DefaultTrustLevel, ubdPeriod+10, ubdPeriod, maxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
			false,
		},
		{
			"invalid upgrade path",
			ibctm.NewClientState(s.chainA.ChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), []string{"bad", "upgrade", "path"}),
			false,
		},
	}

	for _, tc := range testCases {
		err := s.chainA.App.GetIBCKeeper().ClientKeeper.ValidateSelfClient(s.chainA.GetContext(), tc.clientState)
		if tc.expPass {
			s.Require().NoError(err, "expected valid client for case: %s", tc.name)
		} else {
			s.Require().Error(err, "expected invalid client for case: %s", tc.name)
		}
	}
}

func (s KeeperTestSuite) TestGetAllGenesisClients() { //nolint:govet // this is a test, we are okay with copying locks
	clientIDs := []string{
		exported.LocalhostClientID, testClientID2, testClientID3, testClientID,
	}
	expClients := []exported.ClientState{
		localhost.NewClientState(types.GetSelfHeight(s.chainA.GetContext())),
		ibctm.NewClientState(testChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, types.ZeroHeight(), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
		ibctm.NewClientState(testChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, types.ZeroHeight(), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
		ibctm.NewClientState(testChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, types.ZeroHeight(), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
	}

	expGenClients := make(types.IdentifiedClientStates, len(expClients))

	for i := range expClients {
		s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), clientIDs[i], expClients[i])
		expGenClients[i] = types.NewIdentifiedClientState(clientIDs[i], expClients[i])
	}

	genClients := s.chainA.App.GetIBCKeeper().ClientKeeper.GetAllGenesisClients(s.chainA.GetContext())

	s.Require().Equal(expGenClients.Sort(), genClients)
}

func (s KeeperTestSuite) TestGetAllGenesisMetadata() { //nolint:govet // this is a test, we are okay with copying locks
	expectedGenMetadata := []types.IdentifiedGenesisMetadata{
		types.NewIdentifiedGenesisMetadata(
			"07-tendermint-1",
			[]types.GenesisMetadata{
				types.NewGenesisMetadata(ibctm.ProcessedTimeKey(types.NewHeight(0, 1)), []byte("foo")),
				types.NewGenesisMetadata(ibctm.ProcessedTimeKey(types.NewHeight(0, 2)), []byte("bar")),
				types.NewGenesisMetadata(ibctm.ProcessedTimeKey(types.NewHeight(0, 3)), []byte("baz")),
			},
		),
		types.NewIdentifiedGenesisMetadata(
			"clientB",
			[]types.GenesisMetadata{
				types.NewGenesisMetadata(ibctm.ProcessedTimeKey(types.NewHeight(1, 100)), []byte("val1")),
				types.NewGenesisMetadata(ibctm.ProcessedTimeKey(types.NewHeight(2, 300)), []byte("val2")),
			},
		),
	}

	genClients := []types.IdentifiedClientState{
		types.NewIdentifiedClientState("07-tendermint-1", &ibctm.ClientState{}), types.NewIdentifiedClientState("clientB", &ibctm.ClientState{}),
	}

	s.chainA.App.GetIBCKeeper().ClientKeeper.SetAllClientMetadata(s.chainA.GetContext(), expectedGenMetadata)

	actualGenMetadata, err := s.chainA.App.GetIBCKeeper().ClientKeeper.GetAllClientMetadata(s.chainA.GetContext(), genClients)
	s.Require().NoError(err, "get client metadata returned error unexpectedly")
	s.Require().Equal(expectedGenMetadata, actualGenMetadata, "retrieved metadata is unexpected")
}

func (s KeeperTestSuite) TestGetConsensusState() { //nolint:govet // this is a test, we are okay with copying locks
	s.ctx = s.ctx.WithBlockHeight(10)
	cases := []struct {
		name    string
		height  types.Height
		expPass bool
	}{
		{"zero height", types.ZeroHeight(), false},
		{"height > latest height", types.NewHeight(0, uint64(s.ctx.BlockHeight())+1), false},
		{"latest height - 1", types.NewHeight(0, uint64(s.ctx.BlockHeight())-1), true},
		{"latest height", types.GetSelfHeight(s.ctx), true},
	}

	for i, tc := range cases {
		tc := tc
		cs, err := s.keeper.GetSelfConsensusState(s.ctx, tc.height)
		if tc.expPass {
			s.Require().NoError(err, "Case %d should have passed: %s", i, tc.name)
			s.Require().NotNil(cs, "Case %d should have passed: %s", i, tc.name)
		} else {
			s.Require().Error(err, "Case %d should have failed: %s", i, tc.name)
			s.Require().Nil(cs, "Case %d should have failed: %s", i, tc.name)
		}
	}
}

func (s KeeperTestSuite) TestConsensusStateHelpers() { //nolint:govet // this is a test, we are okay with copying locks
	// initial setup
	clientState := ibctm.NewClientState(testChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, testClientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)

	s.keeper.SetClientState(s.ctx, testClientID, clientState)
	s.keeper.SetClientConsensusState(s.ctx, testClientID, testClientHeight, s.consensusState)

	nextState := ibctm.NewConsensusState(s.now, commitmenttypes.NewMerkleRoot([]byte("next")), s.valSetHash)

	testClientHeightPlus5 := types.NewHeight(0, height+5)

	header := s.chainA.CreateTMClientHeader(testClientID, int64(testClientHeightPlus5.RevisionHeight), testClientHeight, s.header.Header.Time.Add(time.Minute),
		s.valSet, s.valSet, s.valSet, s.signers)

	// mock update functionality
	clientState.LatestHeight = header.GetHeight().(types.Height)
	s.keeper.SetClientConsensusState(s.ctx, testClientID, header.GetHeight(), nextState)
	s.keeper.SetClientState(s.ctx, testClientID, clientState)

	latest, ok := s.keeper.GetLatestClientConsensusState(s.ctx, testClientID)
	s.Require().True(ok)
	s.Require().Equal(nextState, latest, "Latest client not returned correctly")
}

// 2 clients in total are created on chainA. The first client is updated so it contains an initial consensus state
// and a consensus state at the update height.
func (s KeeperTestSuite) TestGetAllConsensusStates() { //nolint:govet // this is a test, we are okay with copying locks
	path := ibctesting.NewPath(s.chainA, s.chainB)
	s.coordinator.SetupClients(path)

	clientState := path.EndpointA.GetClientState()
	expConsensusHeight0 := clientState.GetLatestHeight()
	consensusState0, ok := s.chainA.GetConsensusState(path.EndpointA.ClientID, expConsensusHeight0)
	s.Require().True(ok)

	// update client to create a second consensus state
	err := path.EndpointA.UpdateClient()
	s.Require().NoError(err)

	clientState = path.EndpointA.GetClientState()
	expConsensusHeight1 := clientState.GetLatestHeight()
	s.Require().True(expConsensusHeight1.GT(expConsensusHeight0))
	consensusState1, ok := s.chainA.GetConsensusState(path.EndpointA.ClientID, expConsensusHeight1)
	s.Require().True(ok)

	expConsensus := []exported.ConsensusState{
		consensusState0,
		consensusState1,
	}

	// create second client on chainA
	path2 := ibctesting.NewPath(s.chainA, s.chainB)
	s.coordinator.SetupClients(path2)
	clientState = path2.EndpointA.GetClientState()

	expConsensusHeight2 := clientState.GetLatestHeight()
	consensusState2, ok := s.chainA.GetConsensusState(path2.EndpointA.ClientID, expConsensusHeight2)
	s.Require().True(ok)

	expConsensus2 := []exported.ConsensusState{consensusState2}

	expConsensusStates := types.ClientsConsensusStates{
		types.NewClientConsensusStates(path.EndpointA.ClientID, []types.ConsensusStateWithHeight{
			types.NewConsensusStateWithHeight(expConsensusHeight0.(types.Height), expConsensus[0]),
			types.NewConsensusStateWithHeight(expConsensusHeight1.(types.Height), expConsensus[1]),
		}),
		types.NewClientConsensusStates(path2.EndpointA.ClientID, []types.ConsensusStateWithHeight{
			types.NewConsensusStateWithHeight(expConsensusHeight2.(types.Height), expConsensus2[0]),
		}),
	}.Sort()

	consStates := s.chainA.App.GetIBCKeeper().ClientKeeper.GetAllConsensusStates(s.chainA.GetContext())
	s.Require().Equal(expConsensusStates, consStates, "%s \n\n%s", expConsensusStates, consStates)
}

func (s KeeperTestSuite) TestIterateClientStates() { //nolint:govet // this is a test, we are okay with copying locks
	paths := []*ibctesting.Path{
		ibctesting.NewPath(s.chainA, s.chainB),
		ibctesting.NewPath(s.chainA, s.chainB),
		ibctesting.NewPath(s.chainA, s.chainB),
	}

	solomachines := []*ibctesting.Solomachine{
		ibctesting.NewSolomachine(s.T(), s.chainA.Codec, ibctesting.DefaultSolomachineClientID, "testing", 1),
		ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "06-solomachine-1", "testing", 4),
	}

	var (
		expTMClientIDs = make([]string, len(paths))
		expSMClientIDs = make([]string, len(solomachines))
	)

	// create tendermint clients
	for i, path := range paths {
		s.coordinator.SetupClients(path)
		expTMClientIDs[i] = path.EndpointA.ClientID
	}

	// create solomachine clients
	for i, sm := range solomachines {
		expSMClientIDs[i] = sm.CreateClient(s.chainA)
	}

	testCases := []struct {
		name         string
		prefix       []byte
		expClientIDs func() []string
	}{
		{
			"all clientIDs",
			nil,
			func() []string {
				allClientIDs := []string{exported.LocalhostClientID}
				allClientIDs = append(allClientIDs, expSMClientIDs...)
				allClientIDs = append(allClientIDs, expTMClientIDs...)
				return allClientIDs
			},
		},
		{
			"tendermint clientIDs",
			[]byte(exported.Tendermint),
			func() []string {
				return expTMClientIDs
			},
		},
		{
			"solo machine clientIDs",
			[]byte(exported.Solomachine),
			func() []string {
				return expSMClientIDs
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			var clientIDs []string
			s.chainA.GetSimApp().IBCKeeper.ClientKeeper.IterateClientStates(s.chainA.GetContext(), tc.prefix, func(clientID string, _ exported.ClientState) bool {
				clientIDs = append(clientIDs, clientID)
				return false
			})

			s.Require().ElementsMatch(tc.expClientIDs(), clientIDs)
		})
	}
}

// TestDefaultSetParams tests the default params set are what is expected
func (s *KeeperTestSuite) TestDefaultSetParams() {
	expParams := types.DefaultParams()

	clientKeeper := s.chainA.App.GetIBCKeeper().ClientKeeper
	params := clientKeeper.GetParams(s.chainA.GetContext())

	s.Require().Equal(expParams, params)
	s.Require().Equal(expParams.AllowedClients, clientKeeper.GetParams(s.chainA.GetContext()).AllowedClients)
}

// TestParams tests that Param setting and retrieval works properly
func (s *KeeperTestSuite) TestParams() {
	testCases := []struct {
		name    string
		input   types.Params
		expPass bool
	}{
		{"success: set default params", types.DefaultParams(), true},
		{"success: empty allowedClients", types.NewParams(), true},
		{"success: subset of allowedClients", types.NewParams(exported.Tendermint, exported.Localhost), true},
		{"failure: contains a single empty string value as allowedClient", types.NewParams(exported.Localhost, ""), false},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest() // reset
			ctx := s.chainA.GetContext()
			err := tc.input.Validate()
			s.chainA.GetSimApp().IBCKeeper.ClientKeeper.SetParams(ctx, tc.input)
			if tc.expPass {
				s.Require().NoError(err)
				expected := tc.input
				p := s.chainA.GetSimApp().IBCKeeper.ClientKeeper.GetParams(ctx)
				s.Require().Equal(expected, p)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

// TestUnsetParams tests that trying to get params that are not set panics.
func (s *KeeperTestSuite) TestUnsetParams() {
	s.SetupTest()
	ctx := s.chainA.GetContext()
	store := ctx.KVStore(s.chainA.GetSimApp().GetKey(exported.StoreKey))
	store.Delete([]byte(types.ParamsKey))

	s.Require().Panics(func() {
		s.chainA.GetSimApp().IBCKeeper.ClientKeeper.GetParams(ctx)
	})
}
