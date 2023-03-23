package types_test

import (
	"encoding/base64"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	sdk "github.com/cosmos/cosmos-sdk/types"
	wasmtypes "github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
)

// TestCheckSubstituteAndUpdateState only test the interface to the contract, not the full logic of the contract.
func (suite *WasmTestSuite) TestCheckSubstituteAndUpdateState() {
	var (
		subjectClientState exported.ClientState
		subjectClientStore sdk.KVStore
		substituteClientState exported.ClientState
		substituteClientStore sdk.KVStore
	)
	testCases := []struct {
		name string
		setup func()
		expPass bool
	}{
		{
			"success",
			func() {},
			true,
		},
	}
	for _, tc := range testCases {
		tc := tc

		suite.SetupWithChannel()
		subjectClientState = suite.clientState
		subjectClientStore = suite.store

		// Create a new client
		clientStateData, err := base64.StdEncoding.DecodeString(suite.testData["client_state_data"])
		suite.Require().NoError(err)

		substituteClientState = &wasmtypes.ClientState{
			Data:   clientStateData,
			CodeId: suite.codeID,
			LatestHeight: clienttypes.Height{
				RevisionNumber: 2000,
				RevisionHeight: 4,
			},
		}
		consensusStateData, err := base64.StdEncoding.DecodeString(suite.testData["consensus_state_data"])
		suite.Require().NoError(err)
		substituteConsensusState := wasmtypes.ConsensusState{
			Data:      consensusStateData,
			Timestamp: uint64(1678732170022000000),
		}

		substituteClientStore = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.ctx, "08-wasm-1")
		err = substituteClientState.Initialize(suite.ctx, suite.chainA.Codec, substituteClientStore, &substituteConsensusState)

		tc.setup()
		
		err = subjectClientState.CheckSubstituteAndUpdateState(
			suite.ctx,
			suite.chainA.Codec,
			subjectClientStore,
			substituteClientStore,
			substituteClientState,
		)
		if tc.expPass {
			suite.Require().NoError(err)

			// Verify that the substitute client state is in the subject client store
			clientStateBz := subjectClientStore.Get(host.ClientStateKey())
			suite.Require().NotEmpty(clientStateBz)
			newClientState := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, clientStateBz)
			suite.Require().Equal(substituteClientState.GetLatestHeight(), newClientState.GetLatestHeight())

			// Verify that the substitute consensus state is in the subject client store
			// Contract will increment timestamp by 1, verifying it can read from the substitute store and write to the subject store
			newConsensusState, ok := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientConsensusState(suite.ctx, "08-wasm-0", newClientState.GetLatestHeight())
			suite.Require().True(ok)
			suite.Require().Equal(substituteConsensusState.GetTimestamp() + 1, newConsensusState.GetTimestamp())

		} else {
			suite.Require().Error(err)
		}
	}
}