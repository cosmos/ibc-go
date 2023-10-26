package types_test

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"time"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v8/modules/light-clients/06-solomachine"
	tmtypes "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	ibcmock "github.com/cosmos/ibc-go/v8/testing/mock"
)

func (suite *TypesTestSuite) TestStatusGrandpa() {
	var (
		ok          bool
		clientState exported.ClientState
	)

	testCases := []struct {
		name      string
		malleate  func()
		expStatus exported.Status
	}{
		{
			"client is active",
			func() {},
			exported.Active,
		},
		{
			"client is frozen",
			func() {
				clientStateData, err := base64.StdEncoding.DecodeString(suite.testData["client_state_frozen"])
				suite.Require().NoError(err)

				clientState = types.NewClientState(clientStateData, suite.codeHash, clienttypes.NewHeight(2000, 5))

				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.ctx, defaultWasmClientID, clientState)
			},
			exported.Frozen,
		},
		{
			"client status without consensus state",
			func() {
				clientStateData, err := base64.StdEncoding.DecodeString(suite.testData["client_state_no_consensus"])
				suite.Require().NoError(err)

				clientState = types.NewClientState(clientStateData, suite.codeHash, clienttypes.NewHeight(2000, 36))

				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.ctx, defaultWasmClientID, clientState)
			},
			exported.Expired,
		},
		{
			"client state for unexisting contract",
			func() {
				clientStateData, err := base64.StdEncoding.DecodeString(suite.testData["client_state_data"])
				suite.Require().NoError(err)

				codeHash := sha256.Sum256([]byte("bytes-of-light-client-wasm-contract-that-does-not-exist")) // code hash for a contract that does not exists in store
				clientState = types.NewClientState(clientStateData, codeHash[:], clienttypes.NewHeight(2000, 5))
			},
			exported.Unknown,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmGrandpaWithChannel()

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.ctx, defaultWasmClientID)
			clientState, ok = suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.ctx, defaultWasmClientID)
			suite.Require().True(ok)

			tc.malleate()

			status := clientState.Status(suite.ctx, clientStore, suite.chainA.App.AppCodec())
			suite.Require().Equal(tc.expStatus, status)
		})
	}
}

func (suite *TypesTestSuite) TestStatus() {
	testCases := []struct {
		name      string
		malleate  func()
		expStatus exported.Status
	}{
		{
			"client is active",
			func() {},
			exported.Active,
		},
		{
			"client is frozen",
			func() {
				suite.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(codeID wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error) {
					resp, err := json.Marshal(types.StatusResult{Status: exported.Frozen.String()})
					suite.Require().NoError(err)
					return resp, types.DefaultGasUsed, nil
				})
			},
			exported.Frozen,
		},
		{
			"client status is expired",
			func() {
				suite.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(codeID wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error) {
					resp, err := json.Marshal(types.StatusResult{Status: exported.Expired.String()})
					suite.Require().NoError(err)
					return resp, types.DefaultGasUsed, nil
				})
			},
			exported.Expired,
		},
		{
			"client status is unknown: vm returns an error",
			func() {
				suite.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(codeID wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error) {
					return nil, 0, wasmtesting.ErrMockContract
				})
			},
			exported.Unknown,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			tc.malleate()

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), endpoint.ClientID)
			clientState := endpoint.GetClientState()

			status := clientState.Status(suite.chainA.GetContext(), clientStore, suite.chainA.App.AppCodec())
			suite.Require().Equal(tc.expStatus, status)
		})
	}
}

func (suite *TypesTestSuite) TestValidate() {
	testCases := []struct {
		name        string
		clientState *types.ClientState
		expPass     bool
	}{
		{
			name:        "valid client",
			clientState: types.NewClientState([]byte{0}, wasmtesting.Code, clienttypes.ZeroHeight()),
			expPass:     true,
		},
		{
			name:        "nil data",
			clientState: types.NewClientState(nil, wasmtesting.Code, clienttypes.ZeroHeight()),
			expPass:     false,
		},
		{
			name:        "empty data",
			clientState: types.NewClientState([]byte{}, wasmtesting.Code, clienttypes.ZeroHeight()),
			expPass:     false,
		},
		{
			name:        "nil code hash",
			clientState: types.NewClientState([]byte{0}, nil, clienttypes.ZeroHeight()),
			expPass:     false,
		},
		{
			name:        "empty code hash",
			clientState: types.NewClientState([]byte{0}, []byte{}, clienttypes.ZeroHeight()),
			expPass:     false,
		},
		{
			name: "longer than 32 bytes code hash",
			clientState: types.NewClientState(
				[]byte{0},
				[]byte{
					0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
					10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
					20, 21, 22, 23, 24, 25, 26, 27, 28, 29,
					30, 31, 32, 33,
				},
				clienttypes.ZeroHeight(),
			),
			expPass: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.clientState.Validate()
			if tc.expPass {
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
}

func (suite *TypesTestSuite) TestInitializeGrandpa() {
	var (
		consensusState exported.ConsensusState
		clientState    exported.ClientState
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			name:     "valid consensus",
			malleate: func() {},
			expErr:   nil,
		},
		{
			name: "invalid consensus: consensus state is solomachine consensus",
			malleate: func() {
				consensusState = ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachine", "", 2).ConsensusState()
			},
			expErr: clienttypes.ErrInvalidConsensus,
		},
		{
			name: "invalid client state: wasm code hasn't been stored",
			malleate: func() {
				clientStateData, err := base64.StdEncoding.DecodeString(suite.testData["client_state_data"])
				suite.Require().NoError(err)
				clientState = types.NewClientState(clientStateData, wasmtesting.Code, clienttypes.NewHeight(2000, 2))
			},
			expErr: types.ErrInvalidCodeHash,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmGrandpa()

			clientStateData, err := base64.StdEncoding.DecodeString(suite.testData["client_state_data"])
			suite.Require().NoError(err)
			clientState = types.NewClientState(clientStateData, suite.codeHash, clienttypes.NewHeight(2000, 2))

			consensusStateData, err := base64.StdEncoding.DecodeString(suite.testData["consensus_state_data"])
			suite.Require().NoError(err)
			consensusState = types.NewConsensusState(consensusStateData, 0)

			tc.malleate()
			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.ctx, defaultWasmClientID)
			err = clientState.Initialize(suite.ctx, suite.chainA.Codec, clientStore, consensusState)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().True(clientStore.Has(host.ClientStateKey()))
				suite.Require().True(clientStore.Has(host.ConsensusStateKey(clientState.GetLatestHeight())))
			} else {
				suite.Require().Error(err)
				suite.Require().False(clientStore.Has(host.ClientStateKey()))
				suite.Require().False(clientStore.Has(host.ConsensusStateKey(clientState.GetLatestHeight())))
			}
		})
	}
}

func (suite *TypesTestSuite) TestInitialize() {
	var (
		consensusState exported.ConsensusState
		clientState    exported.ClientState
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: new mock client",
			func() {},
			nil,
		},
		{
			"failure: invalid consensus state",
			func() {
				// set upgraded consensus state to solomachine consensus state
				consensusState = &solomachine.ConsensusState{}
			},
			clienttypes.ErrInvalidConsensus,
		},
		{
			"failure: code hash has not been stored.",
			func() {
				clientState = types.NewClientState([]byte{1}, []byte("unknown"), clienttypes.NewHeight(0, 1))
			},
			types.ErrInvalidCodeHash,
		},
		{
			"failure: InstantiateFn returns error",
			func() {
				suite.mockVM.InstantiateFn = func(codeID wasmvm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, initMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					return nil, 0, wasmtesting.ErrMockContract
				}
			},
			wasmtesting.ErrMockContract,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			codeHash := sha256.Sum256(wasmtesting.Code)
			clientState = types.NewClientState([]byte{1}, codeHash[:], clienttypes.NewHeight(0, 1))
			consensusState = types.NewConsensusState([]byte{2}, 0)

			clientID := suite.chainA.App.GetIBCKeeper().ClientKeeper.GenerateClientIdentifier(suite.ctx, clientState.ClientType())
			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.ctx, clientID)

			tc.malleate()

			err := clientState.Initialize(suite.chainA.GetContext(), suite.chainA.Codec, clientStore, consensusState)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				expClientState := clienttypes.MustMarshalClientState(suite.chainA.Codec, clientState)
				suite.Require().Equal(expClientState, clientStore.Get(host.ClientStateKey()))

				expConsensusState := clienttypes.MustMarshalConsensusState(suite.chainA.Codec, consensusState)
				suite.Require().Equal(expConsensusState, clientStore.Get(host.ConsensusStateKey(clientState.GetLatestHeight())))
			} else {
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (suite *TypesTestSuite) TestVerifyMembershipGrandpa() {
	const (
		prefix       = "ibc/"
		connectionID = "connection-0"
		portID       = "transfer"
		channelID    = "channel-0"
	)

	var (
		err              error
		proofHeight      exported.Height
		proof            []byte
		path             exported.Path
		value            []byte
		delayTimePeriod  uint64
		delayBlockPeriod uint64
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"successful ClientState verification",
			func() {},
			true,
		},
		{
			"successful Connection verification",
			func() {
				proofHeight = clienttypes.NewHeight(2000, 11)
				key := host.ConnectionPath(connectionID)
				merklePrefix := commitmenttypes.NewMerklePrefix([]byte(prefix))
				merklePath := commitmenttypes.NewMerklePath(key)
				path, err = commitmenttypes.ApplyPrefix(merklePrefix, merklePath)
				suite.Require().NoError(err)

				proof, err = base64.StdEncoding.DecodeString(suite.testData["connection_proof_try"])
				suite.Require().NoError(err)

				value, err = suite.chainA.Codec.Marshal(&connectiontypes.ConnectionEnd{
					ClientId: tmClientID,
					Counterparty: connectiontypes.Counterparty{
						ClientId:     defaultWasmClientID,
						ConnectionId: connectionID,
						Prefix:       suite.chainA.GetPrefix(),
					},
					DelayPeriod: 1000000000, // Hyperspace requires a non-zero delay in seconds. The test data was generated using a 1-second delay
					State:       connectiontypes.TRYOPEN,
					Versions:    []*connectiontypes.Version{connectiontypes.DefaultIBCVersion},
				})
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"successful Channel verification",
			func() {
				proofHeight = clienttypes.NewHeight(2000, 20)
				key := host.ChannelPath(portID, channelID)
				merklePrefix := commitmenttypes.NewMerklePrefix([]byte(prefix))
				merklePath := commitmenttypes.NewMerklePath(key)
				path, err = commitmenttypes.ApplyPrefix(merklePrefix, merklePath)
				suite.Require().NoError(err)

				proof, err = base64.StdEncoding.DecodeString(suite.testData["channel_proof_try"])
				suite.Require().NoError(err)

				value, err = suite.chainA.Codec.Marshal(&channeltypes.Channel{
					State:    channeltypes.TRYOPEN,
					Ordering: channeltypes.UNORDERED,
					Counterparty: channeltypes.Counterparty{
						PortId:    portID,
						ChannelId: channelID,
					},
					ConnectionHops: []string{connectionID},
					Version:        "ics20-1",
				})
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"successful PacketCommitment verification",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["packet_commitment_data"])
				suite.Require().NoError(err)

				proofHeight = clienttypes.NewHeight(2000, 44)
				packet := channeltypes.NewPacket(
					data,
					2, portID, channelID, portID, channelID, clienttypes.NewHeight(0, 3000),
					0,
				)
				key := host.PacketCommitmentPath(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				merklePrefix := commitmenttypes.NewMerklePrefix([]byte(prefix))
				merklePath := commitmenttypes.NewMerklePath(key)
				path, err = commitmenttypes.ApplyPrefix(merklePrefix, merklePath)
				suite.Require().NoError(err)

				proof, err = base64.StdEncoding.DecodeString(suite.testData["packet_commitment_proof"])
				suite.Require().NoError(err)

				value = channeltypes.CommitPacket(suite.chainA.App.GetIBCKeeper().Codec(), packet)
			},
			true,
		},
		{
			"successful Acknowledgement verification",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["ack_data"])
				suite.Require().NoError(err)

				proofHeight = clienttypes.NewHeight(2000, 33)
				packet := channeltypes.NewPacket(
					data,
					uint64(1), portID, channelID, portID, channelID, clienttypes.NewHeight(2000, 1022),
					1693432290702126952,
				)
				key := host.PacketAcknowledgementKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				merklePrefix := commitmenttypes.NewMerklePrefix([]byte(prefix))
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(merklePrefix, merklePath)
				suite.Require().NoError(err)

				proof, err = base64.StdEncoding.DecodeString(suite.testData["ack_proof"])
				suite.Require().NoError(err)

				value, err = base64.StdEncoding.DecodeString(suite.testData["ack"])
				suite.Require().NoError(err)
				value = channeltypes.CommitAcknowledgement(value)
			},
			true,
		},
		{
			"delay time period has passed", func() {
				delayTimePeriod = uint64(time.Second.Nanoseconds() * 2)
			},
			true,
		},
		{
			"delay time period has not passed", func() {
				delayTimePeriod = uint64(time.Hour.Nanoseconds())
			},
			true,
		},
		{
			"delay block period has passed", func() {
				delayBlockPeriod = 1
			},
			true,
		},
		{
			"delay block period has not passed", func() {
				delayBlockPeriod = 1000
			},
			true,
		},
		{
			"latest client height < height", func() {
				proofHeight = proofHeight.Increment()
			}, false,
		},
		{
			"invalid path type",
			func() {
				path = ibcmock.KeyPath{}
			},
			false,
		},
		{
			"failed to unmarshal merkle proof", func() {
				proof = []byte("invalid proof")
			}, false,
		},
		{
			"consensus state not found", func() {
				proofHeight = clienttypes.ZeroHeight()
			}, false,
		},
		{
			"proof verification failed", func() {
				// change the value being proved
				value = []byte("invalid value")
			}, false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmGrandpaWithChannel() // reset
			clientState, ok := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.ctx, defaultWasmClientID)
			suite.Require().True(ok)

			delayTimePeriod = 1000000000 // Hyperspace requires a non-zero delay in seconds. The test data was generated using a 1-second delay
			delayBlockPeriod = 0

			proofHeight = clienttypes.NewHeight(2000, 11)
			key := host.FullClientStateKey(tmClientID)
			merklePrefix := commitmenttypes.NewMerklePrefix([]byte(prefix))
			merklePath := commitmenttypes.NewMerklePath(string(key))
			path, err = commitmenttypes.ApplyPrefix(merklePrefix, merklePath)
			suite.Require().NoError(err)

			proof, err = base64.StdEncoding.DecodeString(suite.testData["client_state_proof"])
			suite.Require().NoError(err)

			value, err = suite.chainA.Codec.MarshalInterface(&tmtypes.ClientState{
				ChainId: "simd",
				TrustLevel: tmtypes.Fraction{
					Numerator:   1,
					Denominator: 3,
				},
				TrustingPeriod:               time.Second * 64000,
				UnbondingPeriod:              time.Second * 1814400,
				MaxClockDrift:                time.Second * 15,
				FrozenHeight:                 clienttypes.ZeroHeight(),
				LatestHeight:                 clienttypes.NewHeight(0, 41),
				ProofSpecs:                   commitmenttypes.GetSDKSpecs(),
				UpgradePath:                  []string{"upgrade", "upgradedIBCState"},
				AllowUpdateAfterExpiry:       false,
				AllowUpdateAfterMisbehaviour: false,
			})
			suite.Require().NoError(err)

			tc.malleate()

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.ctx, defaultWasmClientID)

			err = clientState.VerifyMembership(
				suite.ctx, clientStore, suite.chainA.Codec,
				proofHeight, delayTimePeriod, delayBlockPeriod,
				proof, path, value,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestVerifyMembership() {
	var (
		clientState      exported.ClientState
		expClientStateBz []byte
		path             exported.Path
		proof            []byte
		proofHeight      exported.Height
		value            []byte
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				expClientStateBz = suite.store.Get(host.ClientStateKey())
				suite.mockVM.RegisterSudoCallback(types.VerifyMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, _ wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.Response, uint64, error) {
					var payload types.SudoMsg
					err := json.Unmarshal(sudoMsg, &payload)
					suite.Require().NoError(err)

					suite.Require().NotNil(payload.VerifyMembership)
					suite.Require().Nil(payload.CheckSubstituteAndUpdateState)
					suite.Require().Nil(payload.UpdateState)
					suite.Require().Nil(payload.UpdateStateOnMisbehaviour)
					suite.Require().Nil(payload.VerifyNonMembership)
					suite.Require().Nil(payload.VerifyUpgradeAndUpdateState)
					suite.Require().Equal(proofHeight, payload.VerifyMembership.Height)
					suite.Require().Equal(path, payload.VerifyMembership.Path)
					suite.Require().Equal(proof, payload.VerifyMembership.Proof)
					suite.Require().Equal(value, payload.VerifyMembership.Value)

					bz, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					return &wasmvmtypes.Response{Data: bz}, types.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"success: with update client state",
			func() {
				suite.mockVM.RegisterSudoCallback(types.VerifyMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.Response, uint64, error) {
					var payload types.SudoMsg
					err := json.Unmarshal(sudoMsg, &payload)
					suite.Require().NoError(err)

					suite.Require().NotNil(payload.VerifyMembership)
					suite.Require().Nil(payload.CheckSubstituteAndUpdateState)
					suite.Require().Nil(payload.UpdateState)
					suite.Require().Nil(payload.UpdateStateOnMisbehaviour)
					suite.Require().Nil(payload.VerifyNonMembership)
					suite.Require().Nil(payload.VerifyUpgradeAndUpdateState)
					suite.Require().Equal(proofHeight, payload.VerifyMembership.Height)
					suite.Require().Equal(path, payload.VerifyMembership.Path)
					suite.Require().Equal(proof, payload.VerifyMembership.Proof)
					suite.Require().Equal(value, payload.VerifyMembership.Value)

					bz, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					expClientStateBz = []byte("client-state-data")
					store.Set(host.ClientStateKey(), expClientStateBz)

					return &wasmvmtypes.Response{Data: bz}, types.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"wasm vm returns invalid proof error",
			func() {
				proof = []byte("invalid proof")

				suite.mockVM.RegisterSudoCallback(types.VerifyMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.Response, uint64, error) {
					return nil, types.DefaultGasUsed, commitmenttypes.ErrInvalidProof
				})
			},
			commitmenttypes.ErrInvalidProof,
		},
		{
			"proof height greater than client state latest height",
			func() {
				proofHeight = clienttypes.NewHeight(1, 100)
			},
			ibcerrors.ErrInvalidHeight,
		},
		{
			"invalid path argument",
			func() {
				path = ibcmock.KeyPath{}
			},
			ibcerrors.ErrInvalidType,
		},
		{
			"proof height is invalid type",
			func() {
				proofHeight = ibcmock.Height{}
			},
			ibcerrors.ErrInvalidType,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			path = commitmenttypes.NewMerklePath("/ibc/key/path")
			proof = []byte("valid proof")
			proofHeight = clienttypes.NewHeight(0, 1)
			value = []byte("value")

			tc.malleate()

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), endpoint.ClientID)
			clientState = endpoint.GetClientState()

			err = clientState.VerifyMembership(suite.chainA.GetContext(), clientStore, suite.chainA.Codec, proofHeight, 0, 0, proof, path, value)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				clientStateBz := clientStore.Get(host.ClientStateKey())
				suite.Require().Equal(expClientStateBz, clientStateBz)
			} else {
				suite.Require().ErrorIs(err, tc.expError, "unexpected error in VerifyMembership")
			}
		})
	}
}

func (suite *TypesTestSuite) TestVerifyNonMembershipGrandpa() {
	const (
		prefix              = "ibc/"
		portID              = "transfer"
		invalidClientID     = "09-tendermint-0"
		invalidConnectionID = "connection-100"
		invalidChannelID    = "channel-800"
	)

	var (
		clientState      exported.ClientState
		err              error
		height           exported.Height
		path             exported.Path
		proof            []byte
		delayTimePeriod  uint64
		delayBlockPeriod uint64
		ok               bool
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"successful ClientState verification of non membership",
			func() {
			},
			true,
		},
		{
			"successful ConsensusState verification of non membership", func() {
				height = clienttypes.NewHeight(2000, 11)
				key := host.FullConsensusStateKey(invalidClientID, clientState.GetLatestHeight())
				merklePrefix := commitmenttypes.NewMerklePrefix([]byte(prefix))
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(merklePrefix, merklePath)
				suite.Require().NoError(err)

				proof, err = base64.StdEncoding.DecodeString(suite.testData["client_state_proof"])
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"successful Connection verification of non membership", func() {
				height = clienttypes.NewHeight(2000, 11)
				key := host.ConnectionKey(invalidConnectionID)
				merklePrefix := commitmenttypes.NewMerklePrefix([]byte(prefix))
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(merklePrefix, merklePath)
				suite.Require().NoError(err)

				proof, err = base64.StdEncoding.DecodeString(suite.testData["connection_proof_try"])
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"successful Channel verification of non membership", func() {
				height = clienttypes.NewHeight(2000, 20)
				key := host.ChannelKey(portID, invalidChannelID)
				merklePrefix := commitmenttypes.NewMerklePrefix([]byte(prefix))
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(merklePrefix, merklePath)
				suite.Require().NoError(err)

				proof, err = base64.StdEncoding.DecodeString(suite.testData["channel_proof_try"])
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"successful PacketCommitment verification of non membership", func() {
				height = clienttypes.NewHeight(2000, 44)
				key := host.PacketCommitmentKey(portID, invalidChannelID, 1)
				merklePrefix := commitmenttypes.NewMerklePrefix([]byte(prefix))
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(merklePrefix, merklePath)
				suite.Require().NoError(err)

				proof, err = base64.StdEncoding.DecodeString(suite.testData["packet_commitment_proof"])
				suite.Require().NoError(err)
			}, true,
		},
		{
			"successful Acknowledgement verification of non membership", func() {
				height = clienttypes.NewHeight(2000, 33)
				key := host.PacketAcknowledgementKey(portID, invalidChannelID, 1)
				merklePrefix := commitmenttypes.NewMerklePrefix([]byte(prefix))
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(merklePrefix, merklePath)
				suite.Require().NoError(err)

				proof, err = base64.StdEncoding.DecodeString(suite.testData["ack_proof"])
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"delay time period has passed", func() {
				delayTimePeriod = uint64(time.Second.Nanoseconds())
			},
			true,
		},
		{
			"delay time period has not passed", func() {
				delayTimePeriod = uint64(time.Hour.Nanoseconds())
			},
			true,
		},
		{
			"delay block period has passed", func() {
				delayBlockPeriod = 1
			},
			true,
		},
		{
			"delay block period has not passed", func() {
				delayBlockPeriod = 1000
			},
			true,
		},
		{
			"latest client height < height", func() {
				height = clientState.GetLatestHeight().Increment()
			}, false,
		},
		{
			"invalid path type",
			func() {
				path = ibcmock.KeyPath{}
			},
			false,
		},
		{
			"failed to unmarshal merkle proof", func() {
				proof = []byte("invalid proof")
			}, false,
		},
		{
			"consensus state not found", func() {
				height = clienttypes.ZeroHeight()
			}, false,
		},
		{
			"verify non membership fails as path exists", func() {
				height = clienttypes.NewHeight(2000, 11)
				// change the value being proved
				key := host.FullClientStateKey(tmClientID)
				merklePrefix := commitmenttypes.NewMerklePrefix([]byte(prefix))
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(merklePrefix, merklePath)
				suite.Require().NoError(err)

				proof, err = base64.StdEncoding.DecodeString(suite.testData["client_state_proof"])
				suite.Require().NoError(err)
			}, false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmGrandpaWithChannel() // reset
			clientState, ok = suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.ctx, defaultWasmClientID)
			suite.Require().True(ok)

			delayTimePeriod = 1000000000 // Hyperspace requires a non-zero delay in seconds. The test data was generated using a 1-second delay
			delayBlockPeriod = 0
			height = clienttypes.NewHeight(2000, 11)
			key := host.FullClientStateKey(invalidClientID)
			merklePrefix := commitmenttypes.NewMerklePrefix([]byte(prefix))
			merklePath := commitmenttypes.NewMerklePath(string(key))
			path, err = commitmenttypes.ApplyPrefix(merklePrefix, merklePath)
			suite.Require().NoError(err)

			proof, err = base64.StdEncoding.DecodeString(suite.testData["client_state_proof"])
			suite.Require().NoError(err)

			tc.malleate()

			err = clientState.VerifyNonMembership(
				suite.ctx, suite.store, suite.chainA.Codec,
				height, delayTimePeriod, delayBlockPeriod,
				proof, path,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestVerifyNonMembership() {
	var (
		clientState      exported.ClientState
		expClientStateBz []byte
		path             exported.Path
		proof            []byte
		proofHeight      exported.Height
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				expClientStateBz = suite.store.Get(host.ClientStateKey())
				suite.mockVM.RegisterSudoCallback(types.VerifyNonMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, _ wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.Response, uint64, error) {
					var payload types.SudoMsg
					err := json.Unmarshal(sudoMsg, &payload)
					suite.Require().NoError(err)

					suite.Require().NotNil(payload.VerifyNonMembership)
					suite.Require().Nil(payload.CheckSubstituteAndUpdateState)
					suite.Require().Nil(payload.UpdateState)
					suite.Require().Nil(payload.UpdateStateOnMisbehaviour)
					suite.Require().Nil(payload.VerifyMembership)
					suite.Require().Nil(payload.VerifyUpgradeAndUpdateState)
					suite.Require().Equal(proofHeight, payload.VerifyNonMembership.Height)
					suite.Require().Equal(path, payload.VerifyNonMembership.Path)
					suite.Require().Equal(proof, payload.VerifyNonMembership.Proof)

					bz, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					return &wasmvmtypes.Response{Data: bz}, types.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"success: with update client state",
			func() {
				suite.mockVM.RegisterSudoCallback(types.VerifyNonMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.Response, uint64, error) {
					var payload types.SudoMsg
					err := json.Unmarshal(sudoMsg, &payload)
					suite.Require().NoError(err)

					suite.Require().NotNil(payload.VerifyNonMembership)
					suite.Require().Nil(payload.CheckSubstituteAndUpdateState)
					suite.Require().Nil(payload.UpdateState)
					suite.Require().Nil(payload.UpdateStateOnMisbehaviour)
					suite.Require().Nil(payload.VerifyMembership)
					suite.Require().Nil(payload.VerifyUpgradeAndUpdateState)
					suite.Require().Equal(proofHeight, payload.VerifyNonMembership.Height)
					suite.Require().Equal(path, payload.VerifyNonMembership.Path)
					suite.Require().Equal(proof, payload.VerifyNonMembership.Proof)

					bz, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					expClientStateBz = []byte("client-state-data")
					store.Set(host.ClientStateKey(), expClientStateBz)

					return &wasmvmtypes.Response{Data: bz}, types.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"wasm vm returns invalid proof error",
			func() {
				proof = []byte("invalid proof")

				suite.mockVM.RegisterSudoCallback(types.VerifyNonMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.Response, uint64, error) {
					return nil, types.DefaultGasUsed, commitmenttypes.ErrInvalidProof
				})
			},
			commitmenttypes.ErrInvalidProof,
		},
		{
			"proof height greater than client state latest height",
			func() {
				proofHeight = clienttypes.NewHeight(1, 100)
			},
			ibcerrors.ErrInvalidHeight,
		},
		{
			"invalid path argument",
			func() {
				path = ibcmock.KeyPath{}
			},
			ibcerrors.ErrInvalidType,
		},
		{
			"proof height is invalid type",
			func() {
				proofHeight = ibcmock.Height{}
			},
			ibcerrors.ErrInvalidType,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			path = commitmenttypes.NewMerklePath("/ibc/key/path")
			proof = []byte("valid proof")
			proofHeight = clienttypes.NewHeight(0, 1)

			tc.malleate()

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), endpoint.ClientID)
			clientState = endpoint.GetClientState()

			err = clientState.VerifyNonMembership(suite.chainA.GetContext(), clientStore, suite.chainA.Codec, proofHeight, 0, 0, proof, path)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				clientStateBz := clientStore.Get(host.ClientStateKey())
				suite.Require().Equal(expClientStateBz, clientStateBz)
			} else {
				suite.Require().ErrorIs(err, tc.expError, "unexpected error in VerifyNonMembership")
			}
		})
	}
}
