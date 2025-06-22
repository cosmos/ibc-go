package keeper_test

import (
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	cmttypes "github.com/cometbft/cometbft/types"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/keeper"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/cosmos/ibc-go/v10/testing/simapp"
)

const (
	testChainID          = "gaiahub-0"
	testChainIDRevision1 = "gaiahub-1"

	testClientID  = "tendermint-0"
	testClientID2 = "tendermint-1"
	testClientID3 = "tendermint-2"

	trustingPeriod time.Duration = time.Hour * 24 * 7 * 2
	ubdPeriod      time.Duration = time.Hour * 24 * 7 * 3
	maxClockDrift  time.Duration = time.Second * 10
)

var testClientHeight = types.NewHeight(0, 5)

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

func (s *KeeperTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)

	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))

	isCheckTx := false
	s.now = time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	s.past = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	app := simapp.Setup(s.T(), isCheckTx)

	s.cdc = app.AppCodec()
	s.ctx = app.NewContext(isCheckTx)
	s.keeper = app.IBCKeeper.ClientKeeper
	s.privVal = cmttypes.NewMockPV()
	pubKey, err := s.privVal.GetPubKey()
	s.Require().NoError(err)

	validator := cmttypes.NewValidator(pubKey, 1)
	s.valSet = cmttypes.NewValidatorSet([]*cmttypes.Validator{validator})
	s.valSetHash = s.valSet.Hash()

	s.signers = make(map[string]cmttypes.PrivValidator, 1)
	s.signers[validator.Address.String()] = s.privVal

	s.consensusState = ibctm.NewConsensusState(s.now, commitmenttypes.NewMerkleRoot([]byte("hash")), s.valSetHash)

	var validators stakingtypes.Validators
	for i := 1; i < 11; i++ {
		privVal := cmttypes.NewMockPV()
		tmPk, err := privVal.GetPubKey()
		s.Require().NoError(err)
		pk, err := cryptocodec.FromCmtPubKeyInterface(tmPk)
		s.Require().NoError(err)
		val, err := stakingtypes.NewValidator(pk.Address().String(), pk, stakingtypes.Description{})
		s.Require().NoError(err)

		val.Status = stakingtypes.Bonded
		val.Tokens = sdkmath.NewInt(rand.Int63())
		validators.Validators = append(validators.Validators, val)

		hi := stakingtypes.NewHistoricalInfo(s.ctx.BlockHeader(), validators, sdk.DefaultPowerReduction)
		err = app.StakingKeeper.SetHistoricalInfo(s.ctx, int64(i), &hi)
		s.Require().NoError(err)
	}

	s.solomachine = ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachinesingle", "testing", 1)
}

func (s *KeeperTestSuite) TestSetClientState() {
	clientState := ibctm.NewClientState(testChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, types.ZeroHeight(), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
	s.keeper.SetClientState(s.ctx, testClientID, clientState)

	retrievedState, found := s.keeper.GetClientState(s.ctx, testClientID)
	s.Require().True(found, "GetClientState failed")
	s.Require().Equal(clientState, retrievedState, "Client states are not equal")
}

func (s *KeeperTestSuite) TestSetClientCreator() {
	creator := s.chainA.SenderAccount.GetAddress()
	s.keeper.SetClientCreator(s.ctx, testClientID, creator)
	getCreator := s.keeper.GetClientCreator(s.ctx, testClientID)
	s.Require().Equal(creator, getCreator)
	s.keeper.DeleteClientCreator(s.ctx, testClientID)
	getCreator = s.keeper.GetClientCreator(s.ctx, testClientID)
	s.Require().Equal(sdk.AccAddress(nil), getCreator)
}

func (s *KeeperTestSuite) TestSetClientConsensusState() {
	s.keeper.SetClientConsensusState(s.ctx, testClientID, testClientHeight, s.consensusState)

	retrievedConsState, found := s.keeper.GetClientConsensusState(s.ctx, testClientID, testClientHeight)
	s.Require().True(found, "GetConsensusState failed")

	tmConsState, ok := retrievedConsState.(*ibctm.ConsensusState)
	s.Require().True(ok)
	s.Require().Equal(s.consensusState, tmConsState, "ConsensusState not stored correctly")
}

func (s *KeeperTestSuite) TestGetAllGenesisClients() {
	clientIDs := []string{
		testClientID2, testClientID3, testClientID,
	}
	expClients := []exported.ClientState{
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

func (s *KeeperTestSuite) TestGetAllGenesisMetadata() {
	clientA, clientB := "07-tendermint-1", "clientB"

	// create some starting state
	s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), clientA, &ibctm.ClientState{})
	s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), clientA, types.NewHeight(0, 1), &ibctm.ConsensusState{})
	s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), clientA, types.NewHeight(0, 2), &ibctm.ConsensusState{})
	s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), clientA, types.NewHeight(0, 3), &ibctm.ConsensusState{})
	s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), clientA, types.NewHeight(2, 300), &ibctm.ConsensusState{})

	s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), clientB, &ibctm.ClientState{})
	s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), clientB, types.NewHeight(1, 100), &ibctm.ConsensusState{})
	s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(s.chainA.GetContext(), clientB, types.NewHeight(2, 300), &ibctm.ConsensusState{})

	// NOTE: correct ordering of expected value is required
	// Ordering is typically determined by the lexographic ordering of the height passed into each key.
	expectedGenMetadata := []types.IdentifiedGenesisMetadata{
		types.NewIdentifiedGenesisMetadata(
			clientA,
			[]types.GenesisMetadata{
				types.NewGenesisMetadata(fmt.Appendf(nil, "%s/%s", host.KeyClientState, "clientMetadata"), []byte("value")),
				types.NewGenesisMetadata(ibctm.ProcessedTimeKey(types.NewHeight(0, 1)), []byte("foo")),
				types.NewGenesisMetadata(ibctm.ProcessedTimeKey(types.NewHeight(0, 2)), []byte("bar")),
				types.NewGenesisMetadata(ibctm.ProcessedTimeKey(types.NewHeight(0, 3)), []byte("baz")),
				types.NewGenesisMetadata(ibctm.ProcessedHeightKey(types.NewHeight(2, 300)), []byte(types.NewHeight(1, 100).String())),
			},
		),
		types.NewIdentifiedGenesisMetadata(
			clientB,
			[]types.GenesisMetadata{
				types.NewGenesisMetadata(ibctm.ProcessedTimeKey(types.NewHeight(1, 100)), []byte("val1")),
				types.NewGenesisMetadata(ibctm.ProcessedHeightKey(types.NewHeight(2, 300)), []byte(types.NewHeight(1, 100).String())),
				types.NewGenesisMetadata(ibctm.ProcessedTimeKey(types.NewHeight(2, 300)), []byte("val2")),
				types.NewGenesisMetadata([]byte("key"), []byte("value")),
			},
		),
	}

	genClients := []types.IdentifiedClientState{
		types.NewIdentifiedClientState(clientA, &ibctm.ClientState{}), types.NewIdentifiedClientState(clientB, &ibctm.ClientState{}),
	}

	s.chainA.App.GetIBCKeeper().ClientKeeper.SetAllClientMetadata(s.chainA.GetContext(), expectedGenMetadata)

	actualGenMetadata, err := s.chainA.App.GetIBCKeeper().ClientKeeper.GetAllClientMetadata(s.chainA.GetContext(), genClients)
	s.Require().NoError(err, "get client metadata returned error unexpectedly")
	s.Require().Equal(expectedGenMetadata, actualGenMetadata, "retrieved metadata is unexpected")

	// set invalid key in client store which will cause panic during iteration
	clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), "")
	clientStore.Set([]byte("key"), []byte("val"))
	s.Require().Panics(func() {
		s.chainA.App.GetIBCKeeper().ClientKeeper.GetAllClientMetadata(s.chainA.GetContext(), genClients) //nolint:errcheck // we expect a panic
	})
}

// 2 clients in total are created on chainA. The first client is updated so it contains an initial consensus state
// and a consensus state at the update height.
func (s *KeeperTestSuite) TestGetAllConsensusStates() {
	path1 := ibctesting.NewPath(s.chainA, s.chainB)
	path1.SetupClients()

	expConsensusHeight0 := path1.EndpointA.GetClientLatestHeight()
	consensusState0, ok := s.chainA.GetConsensusState(path1.EndpointA.ClientID, expConsensusHeight0)
	s.Require().True(ok)

	// update client to create a second consensus state
	err := path1.EndpointA.UpdateClient()
	s.Require().NoError(err)

	expConsensusHeight1 := path1.EndpointA.GetClientLatestHeight()
	s.Require().True(expConsensusHeight1.GT(expConsensusHeight0))
	consensusState1, ok := s.chainA.GetConsensusState(path1.EndpointA.ClientID, expConsensusHeight1)
	s.Require().True(ok)

	expConsensus := []exported.ConsensusState{
		consensusState0,
		consensusState1,
	}

	// create second client on chainA
	path2 := ibctesting.NewPath(s.chainA, s.chainB)
	path2.SetupClients()

	expConsensusHeight2 := path2.EndpointA.GetClientLatestHeight()
	consensusState2, ok := s.chainA.GetConsensusState(path2.EndpointA.ClientID, expConsensusHeight2)
	s.Require().True(ok)

	expConsensus2 := []exported.ConsensusState{consensusState2}

	expConsensusStates := types.ClientsConsensusStates{
		types.NewClientConsensusStates(path1.EndpointA.ClientID, []types.ConsensusStateWithHeight{
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

func (s *KeeperTestSuite) TestIterateClientStates() {
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
		path.SetupClients()
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
				return append(expSMClientIDs, expTMClientIDs...)
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

func (s *KeeperTestSuite) TestGetClientLatestHeight() {
	var path *ibctesting.Path

	cases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"invalid client type",
			func() {
				path.EndpointA.ClientID = ibctesting.InvalidID
			},
			false,
		},
		{
			"client type is not allowed", func() {
				params := types.NewParams(exported.Localhost)
				s.chainA.GetSimApp().GetIBCKeeper().ClientKeeper.SetParams(s.chainA.GetContext(), params)
			},
			false,
		},
		{
			"client type is not registered on router", func() {
				path.EndpointA.ClientID = types.FormatClientIdentifier("08-wasm", 0)
			},
			false,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupConnections()

			tc.malleate()

			height := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientLatestHeight(s.chainA.GetContext(), path.EndpointA.ClientID)

			if tc.expPass {
				s.Require().Equal(s.chainB.LatestCommittedHeader.GetHeight().(types.Height), height)
			} else {
				s.Require().Equal(types.ZeroHeight(), height)
			}
		})
	}
}

func (s *KeeperTestSuite) TestGetTimestampAtHeight() {
	var (
		height exported.Height
		path   *ibctesting.Path
	)

	cases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"invalid client type",
			func() {
				path.EndpointA.ClientID = ibctesting.InvalidID
			},
			host.ErrInvalidID,
		},
		{
			"client type is not allowed", func() {
				params := types.NewParams(exported.Localhost)
				s.chainA.GetSimApp().GetIBCKeeper().ClientKeeper.SetParams(s.chainA.GetContext(), params)
			},
			types.ErrInvalidClientType,
		},
		{
			"client type is not registered on router", func() {
				path.EndpointA.ClientID = types.FormatClientIdentifier("08-wasm", 0)
			},
			types.ErrRouteNotFound,
		},
		{
			"client state not found", func() {
				path.EndpointA.ClientID = types.FormatClientIdentifier(exported.Tendermint, 100)
			},
			types.ErrClientNotFound,
		},
		{
			"consensus state not found", func() {
				height = s.chainB.LatestCommittedHeader.GetHeight().Increment()
			},
			types.ErrConsensusStateNotFound,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupConnections()

			height = s.chainB.LatestCommittedHeader.GetHeight()

			tc.malleate()

			actualTimestamp, err := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientTimestampAtHeight(s.chainA.GetContext(), path.EndpointA.ClientID, height)

			if tc.expError == nil {
				s.Require().NoError(err)
				s.Require().Equal(uint64(s.chainB.LatestCommittedHeader.GetTime().UnixNano()), actualTimestamp)
			} else {
				s.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (s *KeeperTestSuite) TestVerifyMembership() {
	var path *ibctesting.Path

	cases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"invalid client id",
			func() {
				path.EndpointA.ClientID = ""
			},
			host.ErrInvalidID,
		},
		{
			"failure: client is frozen",
			func() {
				clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
				s.Require().True(ok)
				clientState.FrozenHeight = types.NewHeight(0, 1)
				path.EndpointA.SetClientState(clientState)
			},
			types.ErrClientNotActive,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.Setup()

			// create default proof, merklePath, and value which passes
			key := host.FullClientStateKey(path.EndpointB.ClientID)
			merklePath := commitmenttypes.NewMerklePath(key)
			merklePrefixPath, err := commitmenttypes.ApplyPrefix(s.chainB.GetPrefix(), merklePath)
			s.Require().NoError(err)

			proof, proofHeight := s.chainB.QueryProof(key)

			clientState, ok := path.EndpointB.GetClientState().(*ibctm.ClientState)
			s.Require().True(ok)
			value, err := s.chainB.Codec.MarshalInterface(clientState)
			s.Require().NoError(err)

			tc.malleate()

			err = s.chainA.App.GetIBCKeeper().ClientKeeper.VerifyMembership(s.chainA.GetContext(), path.EndpointA.ClientID, proofHeight, 0, 0, proof, merklePrefixPath, value)

			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)
			} else {
				s.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (s *KeeperTestSuite) TestVerifyNonMembership() {
	var path *ibctesting.Path

	cases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"invalid client id",
			func() {
				path.EndpointA.ClientID = ""
			},
			host.ErrInvalidID,
		},
		{
			"failure: client is frozen",
			func() {
				clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
				s.Require().True(ok)
				clientState.FrozenHeight = types.NewHeight(0, 1)
				path.EndpointA.SetClientState(clientState)
			},
			types.ErrClientNotActive,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.Setup()

			// create default proof, merklePath, and value which passes
			key := host.FullClientStateKey("invalid-client-id")

			merklePath := commitmenttypes.NewMerklePath(key)
			merklePrefixPath, err := commitmenttypes.ApplyPrefix(s.chainB.GetPrefix(), merklePath)
			s.Require().NoError(err)

			proof, proofHeight := s.chainB.QueryProof(key)

			tc.malleate()

			err = s.chainA.App.GetIBCKeeper().ClientKeeper.VerifyNonMembership(s.chainA.GetContext(), path.EndpointA.ClientID, proofHeight, 0, 0, proof, merklePrefixPath)

			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)
			} else {
				s.Require().ErrorIs(err, tc.expError)
			}
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
		name   string
		input  types.Params
		expErr error
	}{
		{"success: set default params", types.DefaultParams(), nil},
		{"success: empty allowedClients", types.NewParams(), nil},
		{"success: subset of allowedClients", types.NewParams(exported.Tendermint, exported.Localhost), nil},
		{"failure: contains a single empty string value as allowedClient", types.NewParams(exported.Localhost, ""), errors.New("client type 1 cannot be blank")},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			ctx := s.chainA.GetContext()
			err := tc.input.Validate()
			s.chainA.GetSimApp().IBCKeeper.ClientKeeper.SetParams(ctx, tc.input)
			if tc.expErr == nil {
				s.Require().NoError(err)
				expected := tc.input
				p := s.chainA.GetSimApp().IBCKeeper.ClientKeeper.GetParams(ctx)
				s.Require().Equal(expected, p)
			} else {
				s.Require().Error(err)
				s.Require().Equal(err.Error(), tc.expErr.Error())
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

// TestIBCSoftwareUpgrade tests that an IBC client upgrade has been properly scheduled
func (s *KeeperTestSuite) TestIBCSoftwareUpgrade() {
	var (
		upgradedClientState *ibctm.ClientState
		oldPlan, plan       upgradetypes.Plan
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"valid upgrade proposal",
			func() {},
			nil,
		},
		{
			"valid upgrade proposal with previous IBC state", func() {
				oldPlan = upgradetypes.Plan{
					Name:   "upgrade IBC clients",
					Height: 100,
				}
			},
			nil,
		},
		{
			"fail: scheduling upgrade with plan height 0",
			func() {
				plan.Height = 0
			},
			sdkerrors.ErrInvalidRequest,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()      // reset
			oldPlan.Height = 0 // reset

			path := ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupClients()
			tmClientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			s.Require().True(ok)
			upgradedClientState = tmClientState.ZeroCustomFields()

			// use height 1000 to distinguish from old plan
			plan = upgradetypes.Plan{
				Name:   "upgrade IBC clients",
				Height: 1000,
			}

			tc.malleate()

			// set the old plan if it is not empty
			if oldPlan.Height != 0 {
				// set upgrade plan in the upgrade store
				store := s.chainA.GetContext().KVStore(s.chainA.GetSimApp().GetKey(upgradetypes.StoreKey))
				bz := s.chainA.App.AppCodec().MustMarshal(&oldPlan)
				store.Set(upgradetypes.PlanKey(), bz)

				bz, err := types.MarshalClientState(s.chainA.App.AppCodec(), upgradedClientState)
				s.Require().NoError(err)

				s.Require().NoError(s.chainA.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainA.GetContext(), oldPlan.Height, bz))
			}

			ctx := s.chainA.GetContext()
			err := s.chainA.App.GetIBCKeeper().ClientKeeper.ScheduleIBCSoftwareUpgrade(ctx, plan, upgradedClientState)

			if tc.expError == nil {
				s.Require().NoError(err)

				// check that the correct plan is returned
				storedPlan, err := s.chainA.GetSimApp().UpgradeKeeper.GetUpgradePlan(s.chainA.GetContext())
				s.Require().NoError(err)
				s.Require().Equal(plan, storedPlan)

				// check that old upgraded client state is cleared
				cs, err := s.chainA.GetSimApp().UpgradeKeeper.GetUpgradedClient(s.chainA.GetContext(), oldPlan.Height)
				s.Require().ErrorIs(err, upgradetypes.ErrNoUpgradedClientFound)
				s.Require().Empty(cs)

				// check that client state was set
				storedClientState, err := s.chainA.GetSimApp().UpgradeKeeper.GetUpgradedClient(s.chainA.GetContext(), plan.Height)
				s.Require().NoError(err)
				clientState, err := types.UnmarshalClientState(s.chainA.App.AppCodec(), storedClientState)
				s.Require().NoError(err)
				s.Require().Equal(upgradedClientState, clientState)

				expectedEvents := sdk.Events{
					sdk.NewEvent(
						types.EventTypeScheduleIBCSoftwareUpgrade,
						sdk.NewAttribute(types.AttributeKeyUpgradePlanTitle, plan.Name),
						sdk.NewAttribute(types.AttributeKeyUpgradePlanHeight, fmt.Sprintf("%d", plan.Height)),
					),
				}.ToABCIEvents()

				expectedEvents = sdk.MarkEventsToIndex(expectedEvents, map[string]struct{}{})
				ibctesting.AssertEvents(&s.Suite, expectedEvents, ctx.EventManager().Events().ToABCIEvents())
			} else {
				// check that the new plan wasn't stored
				storedPlan, err := s.chainA.GetSimApp().UpgradeKeeper.GetUpgradePlan(s.chainA.GetContext())
				if oldPlan.Height != 0 {
					// NOTE: this is only true if the ScheduleUpgrade function
					// returns an error before clearing the old plan
					s.Require().NoError(err)
					s.Require().Equal(oldPlan, storedPlan)
				} else {
					s.Require().ErrorIs(err, upgradetypes.ErrNoUpgradePlanFound)
					s.Require().Empty(storedPlan)
				}

				// check that client state was not set
				cs, err := s.chainA.GetSimApp().UpgradeKeeper.GetUpgradedClient(s.chainA.GetContext(), plan.Height)
				s.Require().Empty(cs)
				s.Require().ErrorIs(err, upgradetypes.ErrNoUpgradedClientFound)
			}
		})
	}
}
