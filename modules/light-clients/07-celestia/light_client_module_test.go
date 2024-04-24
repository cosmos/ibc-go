package celestia_test

import (
	fmt "fmt"
	"time"

	"github.com/stretchr/testify/require"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	ibccelestia "github.com/cosmos/ibc-go/modules/light-clients/07-celestia"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func (suite *CelestiaTestSuite) TestStatus() {
	var clientID string

	testCases := []struct {
		name      string
		malleate  func()
		expStatus exported.Status
	}{
		{
			"success",
			func() {},
			exported.Active,
		},
		{
			"client state not found",
			func() {
				clientID = fmt.Sprintf("%s-%d", ibccelestia.ModuleName, 100)
			},
			exported.Unknown,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			path := ibctesting.NewPath(suite.chainA, suite.chainB)

			clientID = suite.CreateClient(path.EndpointA)
			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			tc.malleate()

			status := lightClientModule.Status(suite.chainA.GetContext(), clientID)
			suite.Require().Equal(tc.expStatus, status)
		})
	}
}

func (suite *CelestiaTestSuite) TestLatestHeight() {
	var (
		clientID  string
		expHeight exported.Height
	)

	testCases := []struct {
		name     string
		malleate func()
	}{
		{
			"success",
			func() {},
		},
		{
			"client state not found",
			func() {
				expHeight = clienttypes.ZeroHeight()
				clientID = fmt.Sprintf("%s-%d", ibccelestia.ModuleName, 100)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			path := ibctesting.NewPath(suite.chainA, suite.chainB)

			clientID = suite.CreateClient(path.EndpointA)
			expHeight = path.EndpointA.GetClientState().(*ibctm.ClientState).LatestHeight

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			tc.malleate()

			height := lightClientModule.LatestHeight(suite.chainA.GetContext(), clientID)
			suite.Require().Equal(expHeight, height)
		})
	}
}

func (suite *CelestiaTestSuite) TestInitialize() {
	var (
		path                              *ibctesting.Path
		clientStateAny, consensusStateAny *codectypes.Any
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {
			},
			nil,
		},
		{
			"client state is not celestia client state",
			func() {
				clientStateAny = &codectypes.Any{
					Value: []byte("invalid client state bytes"),
				}
			},
			clienttypes.ErrInvalidClient,
		},
		{
			"client state fails validation",
			func() {
				var err error
				tmClientState := path.EndpointA.GetClientState().(*ibctm.ClientState)
				tmClientState.ChainId = ""
				celestiaClientState := &ibccelestia.ClientState{BaseClient: tmClientState}
				clientStateAny, err = codectypes.NewAnyWithValue(celestiaClientState)
				suite.Require().NoError(err)
			},
			ibctm.ErrInvalidChainID,
		},
		{
			"consensus state is not tendermint consensus state",
			func() {
				consensusStateAny = &codectypes.Any{
					Value: []byte("invalid consensus state bytes"),
				}
			},
			clienttypes.ErrInvalidConsensus,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			var err error
			path.SetupClients()
			tmClientState := path.EndpointA.GetClientState().(*ibctm.ClientState)
			celestiaClientState := &ibccelestia.ClientState{BaseClient: tmClientState}
			clientStateAny, err = codectypes.NewAnyWithValue(celestiaClientState)
			suite.Require().NoError(err)
			consensusState := path.EndpointA.GetConsensusState(tmClientState.LatestHeight)
			consensusStateAny, err = codectypes.NewAnyWithValue(consensusState)
			suite.Require().NoError(err)

			clientID := suite.chainA.App.GetIBCKeeper().ClientKeeper.GenerateClientIdentifier(suite.chainA.GetContext(), ibccelestia.ModuleName)
			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			tc.malleate()

			err = lightClientModule.Initialize(suite.chainA.GetContext(), clientID, clientStateAny.Value, consensusStateAny.Value)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), clientID)
				clientStateBz := clientStore.Get(host.ClientStateKey())
				suite.Require().NotEmpty(clientStateBz)
				clientState := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, clientStateBz)
				suite.Require().Equal(tmClientState, clientState)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *CelestiaTestSuite) TestVerifyClientMessage() {
	var clientID string

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {
			},
			nil,
		},
		{
			"client state not found",
			func() {
				clientID = fmt.Sprintf("%s-%d", ibccelestia.ModuleName, 100)
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			path := ibctesting.NewPath(suite.chainA, suite.chainB)

			clientID = suite.CreateClient(path.EndpointA)
			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			suite.coordinator.CommitBlock(suite.chainB)
			trustedHeight := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
			clientMessage, err := path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
			suite.Require().NoError(err)

			tc.malleate()

			err = lightClientModule.VerifyClientMessage(suite.chainA.GetContext(), clientID, clientMessage)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *CelestiaTestSuite) TestUpdateState() {
	var clientID string

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {
			},
			nil,
		},
		{
			"client state not found",
			func() {
				clientID = fmt.Sprintf("%s-%d", ibccelestia.ModuleName, 100)
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			path := ibctesting.NewPath(suite.chainA, suite.chainB)

			clientID = suite.CreateClient(path.EndpointA)
			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			suite.coordinator.CommitBlock(suite.chainB)
			trustedHeight := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
			clientMessage, err := path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
			suite.Require().NoError(err)

			tc.malleate()

			expPass := tc.expErr == nil
			if expPass {
				consensusHeights := lightClientModule.UpdateState(suite.chainA.GetContext(), clientID, clientMessage)

				tmClientState := path.EndpointA.GetClientState().(*ibctm.ClientState)
				suite.Require().True(tmClientState.LatestHeight.EQ(clientMessage.GetHeight()))
				suite.Require().True(tmClientState.LatestHeight.EQ(consensusHeights[0]))

			} else {
				require.Panics(suite.T(), func() {
					lightClientModule.UpdateState(suite.chainA.GetContext(), clientID, clientMessage)
				}, tc.expErr.Error())
			}
		})
	}
}

func (suite *CelestiaTestSuite) TestCheckForMisbehaviour() {
	var (
		clientID      string
		clientMessage exported.ClientMessage
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success: header time is before existing consensus state time",
			func() {
				tmHeader, ok := clientMessage.(*ibctm.Header)
				suite.Require().True(ok)

				// offset header timestamp before existing consensus state timestamp
				tmHeader.Header.Time = tmHeader.GetTime().Add(-time.Hour)
			},
			nil,
		},
		{
			"client state not found",
			func() {
				clientID = fmt.Sprintf("%s-%d", ibccelestia.ModuleName, 100)
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			path := ibctesting.NewPath(suite.chainA, suite.chainB)

			clientID = suite.CreateClient(path.EndpointA)
			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			var err error
			trustedHeight := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
			clientMessage, err = path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
			suite.Require().NoError(err)

			tc.malleate()

			expPass := tc.expErr == nil
			if expPass {
				foundMisbehaviour := lightClientModule.CheckForMisbehaviour(suite.chainA.GetContext(), clientID, clientMessage)
				suite.Require().True(foundMisbehaviour)
			} else {
				require.Panics(suite.T(), func() {
					lightClientModule.CheckForMisbehaviour(suite.chainA.GetContext(), clientID, clientMessage)
				}, tc.expErr.Error())
			}
		})
	}
}

func (suite *CelestiaTestSuite) TestUpdateStateOnMisbehaviour() {
	var (
		clientID     string
		frozenHeight = clienttypes.NewHeight(0, 1)
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"client state not found",
			func() {
				clientID = fmt.Sprintf("%s-%d", ibccelestia.ModuleName, 100)
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			path := ibctesting.NewPath(suite.chainA, suite.chainB)

			clientID = suite.CreateClient(path.EndpointA)
			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			tc.malleate()

			trustedHeight := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
			clientMessage, err := path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
			suite.Require().NoError(err)

			expPass := tc.expErr == nil
			if expPass {
				lightClientModule.UpdateStateOnMisbehaviour(suite.chainA.GetContext(), clientID, clientMessage)

				clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), clientID)
				clientStateBz := clientStore.Get(host.ClientStateKey())
				suite.Require().NotEmpty(clientStateBz)

				newClientState := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, clientStateBz)
				suite.Require().Equal(frozenHeight, newClientState.(*ibctm.ClientState).FrozenHeight)
			} else {
				require.Panics(suite.T(), func() {
					lightClientModule.UpdateStateOnMisbehaviour(suite.chainA.GetContext(), clientID, clientMessage)
				}, tc.expErr.Error())
			}
		})
	}
}

// func (*CelestiaTestSuite) TestVerifyMembership() {
// 	// TODO
// }

func (suite *CelestiaTestSuite) TestRecoverClient() {
	var subjectClientID, substituteClientID string

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {
			},
			nil,
		},
		{
			"cannot parse malformed substitute client ID",
			func() {
				substituteClientID = ibctesting.InvalidID
			},
			host.ErrInvalidID,
		},
		{
			"substitute client ID does not contain 07-celestia prefix",
			func() {
				substituteClientID = fmt.Sprintf("%s-%d", exported.Solomachine, 100)
			},
			clienttypes.ErrInvalidClientType,
		},
		{
			"cannot find subject client state",
			func() {
				subjectClientID = fmt.Sprintf("%s-%d", ibccelestia.ModuleName, 100)
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"cannot find substitute client state",
			func() {
				substituteClientID = fmt.Sprintf("%s-%d", ibccelestia.ModuleName, 100)
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx := suite.chainA.GetContext()

			subjectPath := ibctesting.NewPath(suite.chainA, suite.chainB)
			subjectClientID = suite.CreateClient(subjectPath.EndpointA)
			tmSubjectClientState := subjectPath.EndpointA.GetClientState()

			substitutePath := ibctesting.NewPath(suite.chainA, suite.chainB)
			substituteClientID = suite.CreateClient(substitutePath.EndpointA)

			tmClientState, ok := tmSubjectClientState.(*ibctm.ClientState)
			suite.Require().True(ok)
			tmClientState.FrozenHeight = tmClientState.LatestHeight
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(ctx, subjectClientID, tmClientState)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(subjectClientID)
			suite.Require().True(found)

			tc.malleate()

			err := lightClientModule.RecoverClient(ctx, subjectClientID, substituteClientID)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				// assert that status of subject client is now Active
				clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, subjectClientID)
				bz := clientStore.Get(host.ClientStateKey())
				tmClientState := clienttypes.MustUnmarshalClientState(suite.chainA.App.AppCodec(), bz).(*ibctm.ClientState)
				suite.Require().Equal(exported.Active, tmClientState.Status(ctx, clientStore, suite.chainA.App.AppCodec()))
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
