package types_test

import (
	"encoding/base64"
	"time"

	storetypes "cosmossdk.io/store/types"
	tmtypes "github.com/cometbft/cometbft/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibctestingmock "github.com/cosmos/ibc-go/v7/testing/mock"
)

func (suite *TypesTestSuite) TestVerifyHeaderGrandpa() {
	var (
		ok          bool
		clientMsg   exported.ClientMessage
		clientState exported.ClientState
	)

	testCases := []struct {
		name    string
		setup   func()
		expPass bool
	}{
		{
			"successful verify header", func() {},
			true,
		},
		{
			"unsuccessful verify header: para id mismatch", func() {
				clientStateData, err := base64.StdEncoding.DecodeString(suite.testData["client_state_para_id_mismatch"])
				suite.Require().NoError(err)

				clientState = &types.ClientState{
					Data:         clientStateData,
					CodeHash:     suite.codeHash,
					LatestHeight: clienttypes.NewHeight(2000, 39),
				}
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.ctx, grandpaClientID, clientState)
			},
			false,
		},
		{
			"unsuccessful verify header: head height < consensus height", func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["header_old"])
				suite.Require().NoError(err)
				clientMsg = &types.ClientMessage{
					Data: data,
				}
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmGrandpaWithChannel()
			clientState, ok = suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.ctx, grandpaClientID)
			suite.Require().True(ok)

			data, err := base64.StdEncoding.DecodeString(suite.testData["header"])
			suite.Require().NoError(err)
			clientMsg = &types.ClientMessage{
				Data: data,
			}

			tc.setup()
			err = clientState.VerifyClientMessage(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestVerifyHeaderTendermint() {
	var (
		path   *ibctesting.Path
		header *types.ClientMessage
	)

	// Setup different validators and signers for testing different types of updates
	altPrivVal := ibctestingmock.NewPV()
	altPubKey, err := altPrivVal.GetPubKey()
	suite.Require().NoError(err)

	revisionHeight := int64(height.RevisionHeight)

	// create modified heights to use for test-cases
	altVal := tmtypes.NewValidator(altPubKey, 100)
	// Create alternative validator set with only altVal, invalid update (too much change in valSet)
	altValSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{altVal})
	altSigners := getAltSigners(altVal, altPrivVal)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name:     "success",
			malleate: func() {},
			expPass:  true,
		},
		{
			name: "successful verify header for header with a previous height",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				// passing the CurrentHeader.Height as the block height as it will become a previous height once we commit N blocks
				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)

				// commit some blocks so that the created Header now has a previous height as the BlockHeight
				suite.coordinator.CommitNBlocks(suite.chainB, 5)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)
			},
			expPass: true,
		},
		{
			name: "successful verify header: header with future height and different validator set",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				// Create bothValSet with both suite validator and altVal
				bothValSet := tmtypes.NewValidatorSet(append(suite.chainB.Vals.Validators, altVal))
				bothSigners := suite.chainB.Signers
				bothSigners[altVal.Address.String()] = altPrivVal

				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+5, trustedHeight, suite.chainB.CurrentHeader.Time, bothValSet, suite.chainB.NextVals, trustedVals, bothSigners)
			},
			expPass: true,
		},
		{
			name: "successful verify header: header with next height and different validator set",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				// Create bothValSet with both suite validator and altVal
				bothValSet := tmtypes.NewValidatorSet(append(suite.chainB.Vals.Validators, altVal))
				bothSigners := suite.chainB.Signers
				bothSigners[altVal.Address.String()] = altPrivVal

				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, bothValSet, suite.chainB.NextVals, trustedVals, bothSigners)
			},
			expPass: true,
		},
		{
			name: "unsuccessful updates, passed in incorrect trusted validators for given consensus state",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				// Create bothValSet with both suite validator and altVal
				bothValSet := tmtypes.NewValidatorSet(append(suite.chainB.Vals.Validators, altVal))
				bothSigners := suite.chainB.Signers
				bothSigners[altVal.Address.String()] = altPrivVal

				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+1, trustedHeight, suite.chainB.CurrentHeader.Time, bothValSet, bothValSet, bothValSet, bothSigners)
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header with next height: update header mismatches nextValSetHash",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				// this will err as altValSet.Hash() != consState.NextValidatorsHash
				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+1, trustedHeight, suite.chainB.CurrentHeader.Time, altValSet, altValSet, trustedVals, altSigners)
			},
			expPass: false,
		},
		{
			name: "unsuccessful update with future height: too much change in validator set",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+1, trustedHeight, suite.chainB.CurrentHeader.Time, altValSet, altValSet, trustedVals, altSigners)
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header: header height revision and trusted height revision mismatch",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				header = suite.chainB.CreateWasmClientHeader("gaia-revision-1", 3, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header: header height < consensus height",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				heightMinus1 := clienttypes.NewHeight(trustedHeight.RevisionNumber, trustedHeight.RevisionHeight-1)

				// Make new header at height less than latest client state
				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, int64(heightMinus1.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header: header basic validation failed",
			malleate: func() {
				var wasmData exported.ClientMessage
				err := suite.chainA.Codec.UnmarshalInterface(header.Data, &wasmData)
				suite.Require().NoError(err)

				tmHeader, ok := wasmData.(*ibctm.Header)
				suite.Require().True(ok)

				// cause header to fail validatebasic by changing commit height to mismatch header height
				tmHeader.SignedHeader.Commit.Height = revisionHeight - 1

				header.Data, err = suite.chainA.Codec.MarshalInterface(tmHeader)
				suite.Require().NoError(err)
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header: header timestamp is not past last client timestamp",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+1, trustedHeight, suite.chainB.CurrentHeader.Time.Add(-time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header: header with incorrect header chain-id",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				header = suite.chainB.CreateWasmClientHeader("gaia", suite.chainB.CurrentHeader.Height+1, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
			},
			expPass: false,
		},
		{
			name: "unsuccessful update: trusting period has passed since last client timestamp",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				header = suite.chainA.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+1, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)

				suite.chainB.ExpireClient(ibctesting.TrustingPeriod)
			},
			expPass: false,
		},
		{
			name: "unsuccessful update for a previous revision",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				// passing the CurrentHeader.Height as the block height as it will become an update to previous revision once we upgrade the client
				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)

				// increment the revision of the chain
				err := path.EndpointB.UpgradeChain()
				suite.Require().NoError(err)
			},
			expPass: false,
		},
		{
			name: "successful update with identical header to a previous update",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				// passing the CurrentHeader.Height as the block height as it will become a previous height once we commit N blocks
				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)

				// update client so the header constructed becomes a duplicate
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)
			},
			expPass: true,
		},
		{
			name: "unsuccessful update to a future revision",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID+"-1", suite.chainB.CurrentHeader.Height+5, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
			},
			expPass: false,
		},
		{
			name: "unsuccessful update: header height revision and trusted height revision mismatch",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				// increment the revision of the chain
				err := path.EndpointB.UpgradeChain()
				suite.Require().NoError(err)

				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmTendermint()
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			err := path.EndpointA.CreateClient()
			suite.Require().NoError(err)

			// ensure counterparty state is committed
			suite.coordinator.CommitBlock(suite.chainB)
			header, height, err = path.EndpointA.Chain.ConstructUpdateWasmClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
			suite.Require().NoError(err)

			tc.malleate()

			clientState := path.EndpointA.GetClientState()

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)

			err = clientState.VerifyClientMessage(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), clientStore, header)

			if tc.expPass {
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestUpdateStateGrandpa() {
	var (
		ok          bool
		clientMsg   exported.ClientMessage
		clientState exported.ClientState
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success with height later than latest height",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["header"])
				suite.Require().NoError(err)
				clientMsg = &types.ClientMessage{
					Data: data,
				}
				// VerifyClientMessage must be run first
				err = clientState.VerifyClientMessage(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"success with not verifying client message",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["header"])
				suite.Require().NoError(err)
				clientMsg = &types.ClientMessage{
					Data: data,
				}
			},
			true,
		},
		{
			"invalid ClientMessage type", func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["misbehaviour"])
				suite.Require().NoError(err)
				clientMsg = &types.ClientMessage{
					Data: data,
				}
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmGrandpaWithChannel()
			clientState, ok = suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.ctx, grandpaClientID)
			suite.Require().True(ok)

			tc.malleate()

			if tc.expPass {
				consensusHeights := clientState.UpdateState(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)

				clientStateBz := suite.store.Get(host.ClientStateKey())
				suite.Require().NotEmpty(clientStateBz)

				newClientState := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, clientStateBz)

				suite.Require().Len(consensusHeights, 1)
				suite.Require().Equal(clienttypes.NewHeight(2000, 47), consensusHeights[0])
				suite.Require().Equal(consensusHeights[0], newClientState.(*types.ClientState).LatestHeight)
			} else {
				suite.Require().Panics(func() {
					clientState.UpdateState(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)
				})
			}
		})
	}
}

func (suite *TypesTestSuite) TestUpdateStateTendermint() {
	var (
		path               *ibctesting.Path
		clientMessage      exported.ClientMessage
		clientStore        storetypes.KVStore
		consensusHeights   []exported.Height
		pruneHeight        clienttypes.Height
		prevClientState    exported.ClientState
		prevConsensusState exported.ConsensusState
	)

	testCases := []struct {
		name      string
		malleate  func()
		expResult func()
		expPass   bool
	}{
		{
			"success with height later than latest height", func() {
				suite.Require().True(path.EndpointA.GetClientState().GetLatestHeight().LT(height))
			},
			func() {
				clientState := path.EndpointA.GetClientState()
				suite.Require().True(clientState.GetLatestHeight().EQ(height)) // new update, updated client state should have changed
				suite.Require().True(clientState.GetLatestHeight().EQ(consensusHeights[0]))
			}, true,
		},
		{
			"success with height earlier than latest height", func() {
				// commit a block so the pre-created ClientMessage
				// isn't used to update the client to a newer height
				suite.coordinator.CommitBlock(suite.chainB)
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				suite.Require().True(path.EndpointA.GetClientState().GetLatestHeight().GT(height))

				prevClientState = path.EndpointA.GetClientState()
			},
			func() {
				clientState := path.EndpointA.GetClientState()
				suite.Require().Equal(clientState, prevClientState) // fill in height, no change to client state
				suite.Require().True(clientState.GetLatestHeight().GT(consensusHeights[0]))
			}, true,
		},
		{
			"success with duplicate header", func() {
				// update client in advance
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				// use the same header which just updated the client
				clientMessage, height, err = path.EndpointA.Chain.ConstructUpdateWasmClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
				suite.Require().NoError(err)
				suite.Require().Equal(path.EndpointA.GetClientState().GetLatestHeight(), height)

				prevClientState = path.EndpointA.GetClientState()
				prevConsensusState = path.EndpointA.GetConsensusState(height)
			},
			func() {
				clientState := path.EndpointA.GetClientState()
				suite.Require().Equal(clientState, prevClientState)
				suite.Require().True(clientState.GetLatestHeight().EQ(consensusHeights[0]))
				suite.Require().Equal(path.EndpointA.GetConsensusState(height), prevConsensusState)
			}, true,
		},
		{
			"success with pruned consensus state", func() {
				// this height will be expired and pruned
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)
				pruneHeight = path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				// Increment the time by a week
				suite.coordinator.IncrementTimeBy(7 * 24 * time.Hour)

				// create the consensus state that can be used as trusted height for next update
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				// Increment the time by another week, then update the client.
				// This will cause the first two consensus states to become expired.
				suite.coordinator.IncrementTimeBy(7 * 24 * time.Hour)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				// ensure counterparty state is committed
				suite.coordinator.CommitBlock(suite.chainB)
				clientMessage, height, err = path.EndpointA.Chain.ConstructUpdateWasmClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
				suite.Require().NoError(err)
			},
			func() {
				clientState := path.EndpointA.GetClientState()
				suite.Require().True(clientState.GetLatestHeight().EQ(height)) // new update, updated client state should have changed
				suite.Require().True(clientState.GetLatestHeight().EQ(consensusHeights[0]))

				// ensure consensus state was pruned
				_, found := path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, pruneHeight)
				suite.Require().False(found)
			}, true,
		},
		{
			"success with pruned consensus state using duplicate header", func() {
				// this height will be expired and pruned
				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)
				pruneHeight = path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				// assert that a consensus state exists at the prune height
				consensusState, found := path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, pruneHeight)
				suite.Require().True(found)
				suite.Require().NotNil(consensusState)

				// Increment the time by a week
				suite.coordinator.IncrementTimeBy(7 * 24 * time.Hour)

				// create the consensus state that can be used as trusted height for next update
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				// Increment the time by another week, then update the client.
				// This will cause the first two consensus states to become expired.
				suite.coordinator.IncrementTimeBy(7 * 24 * time.Hour)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				// use the same header which just updated the client
				clientMessage, height, err = path.EndpointA.Chain.ConstructUpdateWasmClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
				suite.Require().NoError(err)
			},
			func() {
				clientState := path.EndpointA.GetClientState()
				suite.Require().True(clientState.GetLatestHeight().EQ(height)) // new update, updated client state should have changed
				suite.Require().True(clientState.GetLatestHeight().EQ(consensusHeights[0]))

				// ensure consensus state was pruned
				_, found := path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, pruneHeight)
				suite.Require().False(found)
			}, true,
		},
		{
			"invalid ClientMessage type", func() {
				clientMessage = &ibctm.Misbehaviour{}
			},
			func() {},
			false,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmTendermint() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			err := path.EndpointA.CreateClient()
			suite.Require().NoError(err)

			// ensure counterparty state is committed
			suite.coordinator.CommitBlock(suite.chainB)
			clientMessage, height, err = path.EndpointA.Chain.ConstructUpdateWasmClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
			suite.Require().NoError(err)

			tc.malleate()

			clientState := path.EndpointA.GetClientState()
			clientStore = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)

			if tc.expPass {
				consensusHeights = clientState.UpdateState(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), clientStore, clientMessage)

				clientMessage, ok := clientMessage.(*types.ClientMessage)
				suite.Require().True(ok)
				var eHeader exported.ClientMessage
				err := suite.chainA.Codec.UnmarshalInterface(clientMessage.Data, &eHeader)
				tmHeader := eHeader.(*ibctm.Header)
				suite.Require().NoError(err)
				expTmConsensusState := &ibctm.ConsensusState{
					Timestamp:          tmHeader.GetTime(),
					Root:               commitmenttypes.NewMerkleRoot(tmHeader.Header.GetAppHash()),
					NextValidatorsHash: tmHeader.Header.NextValidatorsHash,
				}
				wasmData, err := suite.chainA.Codec.MarshalInterface(expTmConsensusState)
				suite.Require().NoError(err)
				expWasmConsensusState := &types.ConsensusState{
					Data: wasmData,
				}

				bz := clientStore.Get(host.ConsensusStateKey(height))
				updatedConsensusState := clienttypes.MustUnmarshalConsensusState(suite.chainA.App.AppCodec(), bz)

				suite.Require().Equal(expWasmConsensusState, updatedConsensusState)

			} else {
				suite.Require().Panics(func() {
					clientState.UpdateState(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), clientStore, clientMessage)
				})
			}

			// perform custom checks
			tc.expResult()
		})
	}
}

func (suite *TypesTestSuite) TestPruneConsensusStateTendermint() {
	suite.SetupWasmTendermint()
	// create path and setup clients
	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupClients(path)

	// get the first height as it will be pruned first.
	var pruneHeight exported.Height
	getFirstHeightCb := func(height exported.Height) bool {
		pruneHeight = height
		return true
	}
	ctx := path.EndpointA.Chain.GetContext()
	clientStore := path.EndpointA.Chain.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, path.EndpointA.ClientID)
	ibctm.IterateConsensusStateAscending(clientStore, getFirstHeightCb)

	// this height will be expired but not pruned
	err := path.EndpointA.UpdateClient()
	suite.Require().NoError(err)
	expiredHeight := path.EndpointA.GetClientState().GetLatestHeight()

	// expected values that must still remain in store after pruning
	expectedConsState, ok := path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, expiredHeight)
	suite.Require().True(ok)
	ctx = path.EndpointA.Chain.GetContext()
	clientStore = path.EndpointA.Chain.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, path.EndpointA.ClientID)
	expectedProcessTime, ok := ibctm.GetProcessedTime(clientStore, expiredHeight)
	suite.Require().True(ok)
	expectedProcessHeight, ok := GetProcessedHeight(clientStore, expiredHeight)
	suite.Require().True(ok)
	expectedConsKey := ibctm.GetIterationKey(clientStore, expiredHeight)
	suite.Require().NotNil(expectedConsKey)

	// Increment the time by a week
	suite.coordinator.IncrementTimeBy(7 * 24 * time.Hour)

	// create the consensus state that can be used as trusted height for next update
	err = path.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	// Increment the time by another week, then update the client.
	// This will cause the first two consensus states to become expired.
	suite.coordinator.IncrementTimeBy(7 * 24 * time.Hour)
	err = path.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	ctx = path.EndpointA.Chain.GetContext()
	clientStore = path.EndpointA.Chain.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, path.EndpointA.ClientID)

	// check that the first expired consensus state got deleted along with all associated metadata
	consState, ok := path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, pruneHeight)
	suite.Require().Nil(consState, "expired consensus state not pruned")
	suite.Require().False(ok)
	// check processed time metadata is pruned
	processTime, ok := ibctm.GetProcessedTime(clientStore, pruneHeight)
	suite.Require().Equal(uint64(0), processTime, "processed time metadata not pruned")
	suite.Require().False(ok)
	processHeight, ok := GetProcessedHeight(clientStore, pruneHeight)
	suite.Require().Zero(processHeight, "processed height metadata not pruned")
	suite.Require().False(ok)

	// check iteration key metadata is pruned
	consKey := ibctm.GetIterationKey(clientStore, pruneHeight)
	suite.Require().Nil(consKey, "iteration key not pruned")

	// check that second expired consensus state doesn't get deleted
	// this ensures that there is a cap on gas cost of UpdateClient
	consState, ok = path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, expiredHeight)
	suite.Require().Equal(expectedConsState, consState, "consensus state incorrectly pruned")
	suite.Require().True(ok)
	// check processed time metadata is not pruned
	processTime, ok = ibctm.GetProcessedTime(clientStore, expiredHeight)
	suite.Require().Equal(expectedProcessTime, processTime, "processed time metadata incorrectly pruned")
	suite.Require().True(ok)

	// check processed height metadata is not pruned
	processHeight, ok = GetProcessedHeight(clientStore, expiredHeight)
	suite.Require().Equal(expectedProcessHeight, processHeight, "processed height metadata incorrectly pruned")
	suite.Require().True(ok)

	// check iteration key metadata is not pruned
	consKey = ibctm.GetIterationKey(clientStore, expiredHeight)
	suite.Require().Equal(expectedConsKey, consKey, "iteration key incorrectly pruned")
}

/* func (suite *TypesTestSuite) TestUpdateStateOnMisbehaviourGrandpa() {
	var (
		ok          bool
		clientMsg   exported.ClientMessage
		clientState exported.ClientState
	)

	testCases := []struct {
		name    string
		setup   func()
		expPass bool
	}{
		{
			"successful update",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["misbehaviour"])
				suite.Require().NoError(err)
				clientMsg = &types.ClientMessage{
					Data: data,
				}

				clientState, ok = suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.ctx, grandpaClientID)
				suite.Require().True(ok)
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmGrandpaWithChannel()
			tc.setup()

			if tc.expPass {
				suite.Require().NotPanics(func() {
					clientState.UpdateStateOnMisbehaviour(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)
				})
				clientStateBz := suite.store.Get(host.ClientStateKey())
				suite.Require().NotEmpty(clientStateBz)

				newClientState := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, clientStateBz)
				status := newClientState.Status(suite.ctx, suite.store, suite.chainA.Codec)
				suite.Require().Equal(exported.Frozen, status)
			} else {
				suite.Require().Panics(func() {
					clientState.UpdateStateOnMisbehaviour(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)
				})
			}
		})
	}
}*/

func (suite *TypesTestSuite) TestUpdateStateOnMisbehaviourTendermint() {
	var path *ibctesting.Path

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// reset suite to create fresh application state
			suite.SetupWasmTendermint()
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			err := path.EndpointA.CreateClient()
			suite.Require().NoError(err)

			tc.malleate()

			clientState := path.EndpointA.GetClientState()
			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)

			misbehaviourHeader, err := path.EndpointA.Chain.ConstructUpdateTMClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
			suite.Require().NoError(err)
			tmMisbehaviour := &ibctm.Misbehaviour{
				Header1: misbehaviourHeader,
				Header2: misbehaviourHeader,
			}
			wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
			suite.Require().NoError(err)
			clientMessage := &types.ClientMessage{
				Data: wasmData,
			}
			clientState.UpdateStateOnMisbehaviour(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), clientStore, clientMessage)

			if tc.expPass {
				clientStateBz := clientStore.Get(host.ClientStateKey())
				suite.Require().NotEmpty(clientStateBz)

				newClientState := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, clientStateBz)
				newWasmClientState := newClientState.(*types.ClientState)

				var innerClientState exported.ClientState
				err = suite.chainA.Codec.UnmarshalInterface(newWasmClientState.Data, &innerClientState)
				suite.Require().NoError(err)
				suite.Require().Equal(misbehaviourHeader.GetHeight(), innerClientState.(*ibctm.ClientState).FrozenHeight)

				status := clientState.Status(suite.chainA.GetContext(), clientStore, suite.chainA.Codec)
				suite.Require().Equal(exported.Frozen, status)
			}
		})
	}
}
