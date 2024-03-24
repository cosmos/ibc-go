package celestia_test

import (
	fmt "fmt"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v8/modules/light-clients/06-solomachine"
	ibccelestia "github.com/cosmos/ibc-go/v8/modules/light-clients/07-celestia"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func (suite *CelestiaTestSuite) TestClientStateUpdateState() {
	var clientMessage exported.ClientMessage

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
			"client message is not tendermint header",
			func() {
				clientMessage = &solomachine.Header{}
			},
			fmt.Errorf("expected type %T, got %T", &ibctm.Header{}, &solomachine.Header{}),
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			path := ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupClients()

			var err error
			suite.coordinator.CommitBlock(suite.chainB)
			trustedHeight := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
			clientMessage, err = path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
			suite.Require().NoError(err)

			tc.malleate()

			tmClientState := path.EndpointA.GetClientState().(*ibctm.ClientState)
			celestiaClientState := ibccelestia.ClientState{BaseClient: tmClientState}
			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)

			expPass := tc.expErr == nil
			if expPass {
				tmClientMessage := clientMessage.(*ibctm.Header)
				consensusHeights := celestiaClientState.UpdateState(suite.chainA.GetContext(), suite.chainA.Codec, clientStore, clientMessage)

				tmClientState = path.EndpointA.GetClientState().(*ibctm.ClientState)
				suite.Require().True(tmClientState.LatestHeight.EQ(tmClientMessage.GetHeight()))
				suite.Require().True(tmClientState.LatestHeight.EQ(consensusHeights[0]))

				// check thata consensus state is overwritten with data hash as commitment root
				consensusState, found := ibctm.GetConsensusState(clientStore, suite.chainA.Codec, tmClientState.LatestHeight)
				suite.Require().True(found)
				suite.Require().Equal(commitmenttypes.NewMerkleRoot(tmClientMessage.Header.GetDataHash()), consensusState.Root)
			} else {
				require.Panics(suite.T(), func() {
					tmClientState.UpdateState(suite.chainA.GetContext(), suite.chainA.Codec, clientStore, clientMessage)
				}, tc.expErr.Error())
			}
		})
	}
}
