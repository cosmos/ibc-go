package keeper_test

import (
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

	"github.com/cosmos/ibc-go/v9/modules/core/02-client/keeper"
	"github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/ibc-go/v9/testing/simapp"
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
	suite.keeper = app.IBCKeeper.ClientKeeper
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

		hi := stakingtypes.NewHistoricalInfo(suite.ctx.BlockHeader(), validators, sdk.DefaultPowerReduction)
		err = app.StakingKeeper.SetHistoricalInfo(suite.ctx, int64(i), &hi)
		suite.Require().NoError(err)
	}

	suite.solomachine = ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachinesingle", "testing", 1)
}

func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) TestSetClientState() {
	clientState := ibctm.NewClientState(testChainID, ibctm.DefaultTrustLevel, trustingPeriod, ubdPeriod, maxClockDrift, types.ZeroHeight(), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
	suite.keeper.SetClientState(suite.ctx, testClientID, clientState)

	retrievedState, found := suite.keeper.GetClientState(suite.ctx, testClientID)
	suite.Require().True(found, "GetClientState failed")
	suite.Require().Equal(clientState, retrievedState, "Client states are not equal")
}

func (suite *KeeperTestSuite) TestSetClientConsensusState() {
	suite.keeper.SetClientConsensusState(suite.ctx, testClientID, testClientHeight, suite.consensusState)

	retrievedConsState, found := suite.keeper.GetClientConsensusState(suite.ctx, testClientID, testClientHeight)
	suite.Require().True(found, "GetConsensusState failed")

	tmConsState, ok := retrievedConsState.(*ibctm.ConsensusState)
	suite.Require().True(ok)
	suite.Require().Equal(suite.consensusState, tmConsState, "ConsensusState not stored correctly")
}

func (suite *KeeperTestSuite) TestGetAllGenesisClients() {
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
		suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientIDs[i], expClients[i])
		expGenClients[i] = types.NewIdentifiedClientState(clientIDs[i], expClients[i])
	}

	genClients := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetAllGenesisClients(suite.chainA.GetContext())

	suite.Require().Equal(expGenClients.Sort(), genClients)
}

func (suite *KeeperTestSuite) TestGetAllGenesisMetadata() {
	clientA, clientB := "07-tendermint-1", "clientB"

	// create some starting state
	suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientA, &ibctm.ClientState{})
	suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), clientA, types.NewHeight(0, 1), &ibctm.ConsensusState{})
	suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), clientA, types.NewHeight(0, 2), &ibctm.ConsensusState{})
	suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), clientA, types.NewHeight(0, 3), &ibctm.ConsensusState{})
	suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), clientA, types.NewHeight(2, 300), &ibctm.ConsensusState{})

	suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientB, &ibctm.ClientState{})
	suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), clientB, types.NewHeight(1, 100), &ibctm.ConsensusState{})
	suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(suite.chainA.GetContext(), clientB, types.NewHeight(2, 300), &ibctm.ConsensusState{})

	// NOTE: correct ordering of expected value is required
	// Ordering is typically determined by the lexographic ordering of the height passed into each key.
	expectedGenMetadata := []types.IdentifiedGenesisMetadata{
		types.NewIdentifiedGenesisMetadata(
			clientA,
			[]types.GenesisMetadata{
				types.NewGenesisMetadata([]byte(fmt.Sprintf("%s/%s", host.KeyClientState, "clientMetadata")), []byte("value")),
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

	suite.chainA.App.GetIBCKeeper().ClientKeeper.SetAllClientMetadata(suite.chainA.GetContext(), expectedGenMetadata)

	actualGenMetadata, err := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetAllClientMetadata(suite.chainA.GetContext(), genClients)
	suite.Require().NoError(err, "get client metadata returned error unexpectedly")
	suite.Require().Equal(expectedGenMetadata, actualGenMetadata, "retrieved metadata is unexpected")

	// set invalid key in client store which will cause panic during iteration
	clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), "")
	clientStore.Set([]byte("key"), []byte("val"))
	suite.Require().Panics(func() {
		suite.chainA.App.GetIBCKeeper().ClientKeeper.GetAllClientMetadata(suite.chainA.GetContext(), genClients) //nolint:errcheck // we expect a panic
	})
}

// 2 clients in total are created on chainA. The first client is updated so it contains an initial consensus state
// and a consensus state at the update height.
func (suite *KeeperTestSuite) TestGetAllConsensusStates() {
	path1 := ibctesting.NewPath(suite.chainA, suite.chainB)
	path1.SetupClients()

	expConsensusHeight0 := path1.EndpointA.GetClientLatestHeight()
	consensusState0, ok := suite.chainA.GetConsensusState(path1.EndpointA.ClientID, expConsensusHeight0)
	suite.Require().True(ok)

	// update client to create a second consensus state
	err := path1.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	expConsensusHeight1 := path1.EndpointA.GetClientLatestHeight()
	suite.Require().True(expConsensusHeight1.GT(expConsensusHeight0))
	consensusState1, ok := suite.chainA.GetConsensusState(path1.EndpointA.ClientID, expConsensusHeight1)
	suite.Require().True(ok)

	expConsensus := []exported.ConsensusState{
		consensusState0,
		consensusState1,
	}

	// create second client on chainA
	path2 := ibctesting.NewPath(suite.chainA, suite.chainB)
	path2.SetupClients()

	expConsensusHeight2 := path2.EndpointA.GetClientLatestHeight()
	consensusState2, ok := suite.chainA.GetConsensusState(path2.EndpointA.ClientID, expConsensusHeight2)
	suite.Require().True(ok)

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

	consStates := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetAllConsensusStates(suite.chainA.GetContext())
	suite.Require().Equal(expConsensusStates, consStates, "%s \n\n%s", expConsensusStates, consStates)
}

func (suite *KeeperTestSuite) TestIterateClientStates() {
	paths := []*ibctesting.Path{
		ibctesting.NewPath(suite.chainA, suite.chainB),
		ibctesting.NewPath(suite.chainA, suite.chainB),
		ibctesting.NewPath(suite.chainA, suite.chainB),
	}

	solomachines := []*ibctesting.Solomachine{
		ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, ibctesting.DefaultSolomachineClientID, "testing", 1),
		ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "06-solomachine-1", "testing", 4),
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
		expSMClientIDs[i] = sm.CreateClient(suite.chainA)
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
		tc := tc
		suite.Run(tc.name, func() {
			var clientIDs []string
			suite.chainA.GetSimApp().IBCKeeper.ClientKeeper.IterateClientStates(suite.chainA.GetContext(), tc.prefix, func(clientID string, _ exported.ClientState) bool {
				clientIDs = append(clientIDs, clientID)
				return false
			})

			suite.Require().ElementsMatch(tc.expClientIDs(), clientIDs)
		})
	}
}

func (suite *KeeperTestSuite) TestGetClientLatestHeight() {
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
				suite.chainA.GetSimApp().GetIBCKeeper().ClientKeeper.SetParams(suite.chainA.GetContext(), params)
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
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupConnections()

			tc.malleate()

			height := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientLatestHeight(suite.chainA.GetContext(), path.EndpointA.ClientID)

			if tc.expPass {
				suite.Require().Equal(suite.chainB.LatestCommittedHeader.GetHeight().(types.Height), height)
			} else {
				suite.Require().Equal(types.ZeroHeight(), height)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestGetTimestampAtHeight() {
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
				suite.chainA.GetSimApp().GetIBCKeeper().ClientKeeper.SetParams(suite.chainA.GetContext(), params)
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
				height = suite.chainB.LatestCommittedHeader.GetHeight().Increment()
			},
			types.ErrConsensusStateNotFound,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupConnections()

			height = suite.chainB.LatestCommittedHeader.GetHeight()

			tc.malleate()

			actualTimestamp, err := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientTimestampAtHeight(suite.chainA.GetContext(), path.EndpointA.ClientID, height)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(uint64(suite.chainB.LatestCommittedHeader.GetTime().UnixNano()), actualTimestamp)
			} else {
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

// TestDefaultSetParams tests the default params set are what is expected
func (suite *KeeperTestSuite) TestDefaultSetParams() {
	expParams := types.DefaultParams()

	clientKeeper := suite.chainA.App.GetIBCKeeper().ClientKeeper
	params := clientKeeper.GetParams(suite.chainA.GetContext())

	suite.Require().Equal(expParams, params)
	suite.Require().Equal(expParams.AllowedClients, clientKeeper.GetParams(suite.chainA.GetContext()).AllowedClients)
}

// TestParams tests that Param setting and retrieval works properly
func (suite *KeeperTestSuite) TestParams() {
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

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			ctx := suite.chainA.GetContext()
			err := tc.input.Validate()
			suite.chainA.GetSimApp().IBCKeeper.ClientKeeper.SetParams(ctx, tc.input)
			if tc.expPass {
				suite.Require().NoError(err)
				expected := tc.input
				p := suite.chainA.GetSimApp().IBCKeeper.ClientKeeper.GetParams(ctx)
				suite.Require().Equal(expected, p)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// TestUnsetParams tests that trying to get params that are not set panics.
func (suite *KeeperTestSuite) TestUnsetParams() {
	suite.SetupTest()
	ctx := suite.chainA.GetContext()
	store := ctx.KVStore(suite.chainA.GetSimApp().GetKey(exported.StoreKey))
	store.Delete([]byte(types.ParamsKey))

	suite.Require().Panics(func() {
		suite.chainA.GetSimApp().IBCKeeper.ClientKeeper.GetParams(ctx)
	})
}

// TestIBCSoftwareUpgrade tests that an IBC client upgrade has been properly scheduled
func (suite *KeeperTestSuite) TestIBCSoftwareUpgrade() {
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
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()  // reset
			oldPlan.Height = 0 // reset

			path := ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupClients()
			tmClientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			suite.Require().True(ok)
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
				store := suite.chainA.GetContext().KVStore(suite.chainA.GetSimApp().GetKey(upgradetypes.StoreKey))
				bz := suite.chainA.App.AppCodec().MustMarshal(&oldPlan)
				store.Set(upgradetypes.PlanKey(), bz)

				bz, err := types.MarshalClientState(suite.chainA.App.AppCodec(), upgradedClientState)
				suite.Require().NoError(err)

				suite.Require().NoError(suite.chainA.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainA.GetContext(), oldPlan.Height, bz))
			}

			ctx := suite.chainA.GetContext()
			err := suite.chainA.App.GetIBCKeeper().ClientKeeper.ScheduleIBCSoftwareUpgrade(ctx, plan, upgradedClientState)

			if tc.expError == nil {
				suite.Require().NoError(err)

				// check that the correct plan is returned
				storedPlan, err := suite.chainA.GetSimApp().UpgradeKeeper.GetUpgradePlan(suite.chainA.GetContext())
				suite.Require().NoError(err)
				suite.Require().Equal(plan, storedPlan)

				// check that old upgraded client state is cleared
				cs, err := suite.chainA.GetSimApp().UpgradeKeeper.GetUpgradedClient(suite.chainA.GetContext(), oldPlan.Height)
				suite.Require().ErrorIs(err, upgradetypes.ErrNoUpgradedClientFound)
				suite.Require().Empty(cs)

				// check that client state was set
				storedClientState, err := suite.chainA.GetSimApp().UpgradeKeeper.GetUpgradedClient(suite.chainA.GetContext(), plan.Height)
				suite.Require().NoError(err)
				clientState, err := types.UnmarshalClientState(suite.chainA.App.AppCodec(), storedClientState)
				suite.Require().NoError(err)
				suite.Require().Equal(upgradedClientState, clientState)

				expectedEvents := sdk.Events{
					sdk.NewEvent(
						types.EventTypeScheduleIBCSoftwareUpgrade,
						sdk.NewAttribute(types.AttributeKeyUpgradePlanTitle, plan.Name),
						sdk.NewAttribute(types.AttributeKeyUpgradePlanHeight, fmt.Sprintf("%d", plan.Height)),
					),
				}.ToABCIEvents()

				expectedEvents = sdk.MarkEventsToIndex(expectedEvents, map[string]struct{}{})
				ibctesting.AssertEvents(&suite.Suite, expectedEvents, ctx.EventManager().Events().ToABCIEvents())

			} else {
				// check that the new plan wasn't stored
				storedPlan, err := suite.chainA.GetSimApp().UpgradeKeeper.GetUpgradePlan(suite.chainA.GetContext())
				if oldPlan.Height != 0 {
					// NOTE: this is only true if the ScheduleUpgrade function
					// returns an error before clearing the old plan
					suite.Require().NoError(err)
					suite.Require().Equal(oldPlan, storedPlan)
				} else {
					suite.Require().ErrorIs(err, upgradetypes.ErrNoUpgradePlanFound)
					suite.Require().Empty(storedPlan)
				}

				// check that client state was not set
				cs, err := suite.chainA.GetSimApp().UpgradeKeeper.GetUpgradedClient(suite.chainA.GetContext(), plan.Height)
				suite.Require().Empty(cs)
				suite.Require().ErrorIs(err, upgradetypes.ErrNoUpgradedClientFound)
			}
		})
	}
}
