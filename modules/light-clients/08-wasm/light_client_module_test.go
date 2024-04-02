package wasm_test

import (
	"encoding/json"
	"time"

	wasmvm "github.com/CosmWasm/wasmvm/v2"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
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

func (suite *WasmTestSuite) TestRecoverClient() {
	var (
		expectedClientStateBz               []byte
		subjectClientID, substituteClientID string
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		// TODO(02-client routing): add successful test when light client module does not call into 08-wasm ClientState
		// {
		// 	"success",
		// 	func() {
		// 	},
		// 	nil,
		// },
		{
			"cannot parse malformed substitute client ID",
			func() {
				substituteClientID = ibctesting.InvalidID
			},
			host.ErrInvalidID,
		},
		{
			"substitute client ID does not contain 08-wasm prefix",
			func() {
				substituteClientID = tmClientID
			},
			clienttypes.ErrInvalidClientType,
		},
		{
			"cannot find subject client state",
			func() {
				subjectClientID = wasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"cannot find substitute client state",
			func() {
				substituteClientID = wasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()
			expectedClientStateBz = nil

			subjectEndpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := subjectEndpoint.CreateClient()
			suite.Require().NoError(err)
			subjectClientID = subjectEndpoint.ClientID

			subjectClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), subjectClientID)

			substituteEndpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err = substituteEndpoint.CreateClient()
			suite.Require().NoError(err)
			substituteClientID = substituteEndpoint.ClientID

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(subjectClientID)
			suite.Require().True(found)

			tc.malleate()

			err = lightClientModule.RecoverClient(suite.chainA.GetContext(), subjectClientID, substituteClientID)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				clientStateBz := subjectClientStore.Get(host.ClientStateKey())
				suite.Require().Equal(expectedClientStateBz, clientStateBz)
			} else {
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *WasmTestSuite) TestVerifyUpgradeAndUpdateState() {
	var (
		clientID                                              string
		clientState                                           *types.ClientState
		upgradedClientState                                   exported.ClientState
		upgradedConsensusState                                exported.ConsensusState
		upgradedClientStateAny, upgradedConsensusStateAny     *codectypes.Any
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
				suite.mockVM.RegisterSudoCallback(types.VerifyUpgradeAndUpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
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
					bz, err := suite.chainA.Codec.MarshalInterface(upgradedClientState)
					suite.Require().NoError(err)

					store.Set(host.ClientStateKey(), bz)

					bz, err = suite.chainA.Codec.MarshalInterface(upgradedConsensusState)
					suite.Require().NoError(err)

					store.Set(host.ConsensusStateKey(expectedUpgradedClient.LatestHeight), bz)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: data}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
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
				upgradedClientStateAny = &codectypes.Any{
					Value: []byte("invalid client state bytes"),
				}
			},
			clienttypes.ErrInvalidClient,
		},
		{
			"upgraded consensus state is not wasm consensus sate",
			func() {
				upgradedConsensusStateAny = &codectypes.Any{
					Value: []byte("invalid consensus state bytes"),
				}
			},
			clienttypes.ErrInvalidConsensus,
		},
		{
			"upgraded client state height is not greater than current height",
			func() {
				var err error
				latestHeight := clientState.LatestHeight
				newLatestHeight := clienttypes.NewHeight(latestHeight.GetRevisionNumber(), latestHeight.GetRevisionHeight()-1)

				wrappedUpgradedClient := wasmtesting.CreateMockTendermintClientState(newLatestHeight)
				wrappedUpgradedClientBz := clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), wrappedUpgradedClient)
				upgradedClientState = types.NewClientState(wrappedUpgradedClientBz, clientState.Checksum, newLatestHeight)
				upgradedClientStateAny, err = codectypes.NewAnyWithValue(upgradedClientState)
				suite.Require().NoError(err)
			},
			ibcerrors.ErrInvalidHeight,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM() // reset suite

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)
			clientID = endpoint.ClientID

			clientState = endpoint.GetClientState().(*types.ClientState)
			latestHeight := clientState.LatestHeight

			newLatestHeight := clienttypes.NewHeight(latestHeight.GetRevisionNumber(), latestHeight.GetRevisionHeight()+1)
			wrappedUpgradedClient := wasmtesting.CreateMockTendermintClientState(newLatestHeight)
			wrappedUpgradedClientBz := clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), wrappedUpgradedClient)
			upgradedClientState = types.NewClientState(wrappedUpgradedClientBz, clientState.Checksum, newLatestHeight)
			upgradedClientStateAny, err = codectypes.NewAnyWithValue(upgradedClientState)
			suite.Require().NoError(err)

			wrappedUpgradedConsensus := ibctm.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("new-hash")), []byte("new-nextValsHash"))
			wrappedUpgradedConsensusBz := clienttypes.MustMarshalConsensusState(suite.chainA.App.AppCodec(), wrappedUpgradedConsensus)
			upgradedConsensusState = types.NewConsensusState(wrappedUpgradedConsensusBz)
			upgradedConsensusStateAny, err = codectypes.NewAnyWithValue(upgradedConsensusState)
			suite.Require().NoError(err)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			tc.malleate()

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), clientID)

			upgradedClientStateProof = wasmtesting.MockUpgradedClientStateProofBz
			upgradedConsensusStateProof = wasmtesting.MockUpgradedConsensusStateProofBz

			err = lightClientModule.VerifyUpgradeAndUpdateState(
				suite.chainA.GetContext(),
				clientID,
				upgradedClientStateAny.Value,
				upgradedConsensusStateAny.Value,
				upgradedClientStateProof,
				upgradedConsensusStateProof,
			)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				// verify new client state and consensus state
				clientStateBz := clientStore.Get(host.ClientStateKey())
				suite.Require().NotEmpty(clientStateBz)

				expClientStateBz, err := suite.chainA.Codec.MarshalInterface(upgradedClientState)
				suite.Require().NoError(err)
				suite.Require().Equal(expClientStateBz, clientStateBz)

				consensusStateBz := clientStore.Get(host.ConsensusStateKey(endpoint.GetClientLatestHeight()))
				suite.Require().NotEmpty(consensusStateBz)

				expConsensusStateBz, err := suite.chainA.Codec.MarshalInterface(upgradedConsensusState)
				suite.Require().NoError(err)
				suite.Require().Equal(expConsensusStateBz, consensusStateBz)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
