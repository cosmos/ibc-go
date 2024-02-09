package wasm_test

import (
	"encoding/json"
	"time"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	wasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

const (
	tmClientID   = "07-tendermint-0"
	wasmClientID = "08-wasm-100"
)

func (suite *WasmTestSuite) TestVerifyUpgradeAndUpdateState() {
	var (
		clientID                                              string
		clientState                                           *wasmtypes.ClientState
		upgradedClientState                                   exported.ClientState
		upgradedConsensusState                                exported.ConsensusState
		upgradedClientStateBz, upgradedConsensusStateBz       []byte
		upgradedClientStateProof, upgradedConsensusStateProof []byte
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {
				suite.mockVM.RegisterSudoCallback(types.VerifyUpgradeAndUpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					var payload types.SudoMsg

					err := json.Unmarshal(sudoMsg, &payload)
					suite.Require().NoError(err)

					expectedUpgradedClient, ok := upgradedClientState.(*types.ClientState)
					suite.Require().True(ok)
					expectedUpgradedConsensus, ok := upgradedConsensusState.(*types.ConsensusState)
					suite.Require().True(ok)

					// verify payload values
					suite.Require().Equal(expectedUpgradedClient.Data, payload.VerifyUpgradeAndUpdateState.UpgradeClientState)
					suite.Require().Equal(expectedUpgradedConsensus.Data, payload.VerifyUpgradeAndUpdateState.UpgradeConsensusState)
					suite.Require().Equal(upgradedClientStateProof, payload.VerifyUpgradeAndUpdateState.ProofUpgradeClient)
					suite.Require().Equal(upgradedConsensusStateProof, payload.VerifyUpgradeAndUpdateState.ProofUpgradeConsensusState)

					// verify other Sudo fields are nil
					suite.Require().Nil(payload.UpdateState)
					suite.Require().Nil(payload.UpdateStateOnMisbehaviour)
					suite.Require().Nil(payload.VerifyMembership)
					suite.Require().Nil(payload.VerifyNonMembership)

					data, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					// set new client state and consensus state
					wrappedUpgradedClient := clienttypes.MustUnmarshalClientState(suite.chainA.App.AppCodec(), expectedUpgradedClient.Data)
					store.Set(host.ClientStateKey(), upgradedClientStateBz)
					store.Set(host.ConsensusStateKey(wrappedUpgradedClient.GetLatestHeight()), upgradedConsensusStateBz)

					return &wasmvmtypes.Response{Data: data}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"cannot parse malformed client ID",
			func() {
				clientID = ibctesting.InvalidID
			},
			host.ErrInvalidID,
		},
		{
			"client type is not 08-wasm",
			func() {
				clientID = tmClientID
			},
			clienttypes.ErrInvalidClientType,
		},
		{
			"cannot find client state",
			func() {
				clientID = wasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"upgraded client state is not wasm client state",
			func() {
				upgradedClientStateBz = []byte{}
			},
			clienttypes.ErrInvalidClient,
		},
		{
			"upgraded consensus state is not wasm consensus sate",
			func() {
				upgradedConsensusStateBz = []byte{}
			},
			clienttypes.ErrInvalidConsensus,
		},
		{
			"upgraded client state height is not greater than current height",
			func() {
				latestHeight := clientState.GetLatestHeight()
				newLatestHeight := clienttypes.NewHeight(latestHeight.GetRevisionNumber(), latestHeight.GetRevisionHeight()-1)

				wrappedUpgradedClient := wasmtesting.CreateMockTendermintClientState(newLatestHeight)
				wrappedUpgradedClientBz := clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), wrappedUpgradedClient)
				upgradedClientState = types.NewClientState(wrappedUpgradedClientBz, clientState.Checksum, newLatestHeight)
				upgradedClientStateBz = clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), upgradedClientState)
			},
			ibcerrors.ErrInvalidHeight,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM() // reset suite
			cdc := suite.chainA.App.AppCodec()
			ctx := suite.chainA.GetContext()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)
			clientID = endpoint.ClientID

			clientState = endpoint.GetClientState().(*wasmtypes.ClientState)
			latestHeight := clientState.GetLatestHeight()

			newLatestHeight := clienttypes.NewHeight(latestHeight.GetRevisionNumber(), latestHeight.GetRevisionHeight()+1)
			wrappedUpgradedClient := wasmtesting.CreateMockTendermintClientState(newLatestHeight)
			wrappedUpgradedClientBz := clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), wrappedUpgradedClient)
			upgradedClientState = types.NewClientState(wrappedUpgradedClientBz, clientState.Checksum, newLatestHeight)
			upgradedClientStateBz = clienttypes.MustMarshalClientState(cdc, upgradedClientState)

			wrappedUpgradedConsensus := ibctm.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("new-hash")), []byte("new-nextValsHash"))
			wrappedUpgradedConsensusBz := clienttypes.MustMarshalConsensusState(cdc, wrappedUpgradedConsensus)
			upgradedConsensusState = types.NewConsensusState(wrappedUpgradedConsensusBz)
			upgradedConsensusStateBz = clienttypes.MustMarshalConsensusState(cdc, upgradedConsensusState)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetRouter().GetRoute(clientID)
			suite.Require().True(found)

			tc.malleate()

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), clientID)

			upgradedClientStateProof = wasmtesting.MockUpgradedClientStateProofBz
			upgradedConsensusStateProof = wasmtesting.MockUpgradedConsensusStateProofBz

			err = lightClientModule.VerifyUpgradeAndUpdateState(
				ctx,
				clientID,
				upgradedClientStateBz,
				upgradedConsensusStateBz,
				upgradedClientStateProof,
				upgradedConsensusStateProof,
			)
			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				// verify new client state and consensus state
				clientStateBz := clientStore.Get(host.ClientStateKey())
				suite.Require().NotEmpty(clientStateBz)
				suite.Require().Equal(upgradedClientStateBz, clientStateBz)

				consensusStateBz := clientStore.Get(host.ConsensusStateKey(upgradedClientState.GetLatestHeight()))
				suite.Require().NotEmpty(consensusStateBz)
				suite.Require().NotEmpty(upgradedConsensusStateBz, consensusStateBz)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
