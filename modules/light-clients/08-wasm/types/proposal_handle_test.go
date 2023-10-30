package types_test

import (
	"encoding/base64"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
)

// var frozenHeight = clienttypes.NewHeight(0, 1)

// TestCheckSubstituteAndUpdateState only tests the interface to the contract, not the full logic of the contract.
func (suite *TypesTestSuite) TestCheckSubstituteAndUpdateStateGrandpa() {
	var (
		ok                                        bool
		subjectClientState, substituteClientState exported.ClientState
		subjectClientStore, substituteClientStore storetypes.KVStore
	)
	testCases := []struct {
		name    string
		setup   func()
		expPass bool
	}{
		{
			"success",
			func() {},
			true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmGrandpaWithChannel()
			subjectClientState, ok = suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.ctx, defaultWasmClientID)
			suite.Require().True(ok)
			subjectClientStore = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.ctx, defaultWasmClientID)

			substituteClientState, ok = suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.ctx, defaultWasmClientID)
			suite.Require().True(ok)

			consensusStateData, err := base64.StdEncoding.DecodeString(suite.testData["consensus_state_data"])
			suite.Require().NoError(err)
			substituteConsensusState := types.ConsensusState{
				Data: consensusStateData,
			}

			substituteClientStore = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.ctx, "08-wasm-1")
			err = substituteClientState.Initialize(suite.ctx, suite.chainA.Codec, substituteClientStore, &substituteConsensusState)
			suite.Require().NoError(err)

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
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func GetProcessedHeight(clientStore storetypes.KVStore, height exported.Height) (uint64, bool) {
	key := ibctm.ProcessedHeightKey(height)
	bz := clientStore.Get(key)
	if len(bz) == 0 {
		return 0, false
	}

	return sdk.BigEndianToUint64(bz), true
}
