package solomachine_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v8/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	ibcmock "github.com/cosmos/ibc-go/v8/testing/mock"
)

const (
	smClientID   = "06-solomachine-100"
	wasmClientID = "08-wasm-0"
)

func (suite *SoloMachineTestSuite) TestStatus() {
	clientID := suite.solomachine.ClientID
	clientState := suite.solomachine.ClientState()

	// Set a client state in store.
	suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, clientState)

	lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
	suite.Require().True(found)

	status := lightClientModule.Status(suite.chainA.GetContext(), clientID)

	// solo machine discards arguments
	suite.Require().Equal(exported.Active, status)

	// freeze solo machine and update it in store.
	clientState.IsFrozen = true
	suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, clientState)

	status = clientState.Status(suite.chainA.GetContext(), nil, nil)
	suite.Require().Equal(exported.Frozen, status)
}

func (suite *SoloMachineTestSuite) TestGetTimestampAtHeight() {
	clientID := suite.solomachine.ClientID
	height := clienttypes.NewHeight(0, suite.solomachine.ClientState().Sequence)
	// Single setup for all test cases.
	suite.SetupTest()

	testCases := []struct {
		name     string
		clientID string
		expValue uint64
		expErr   error
	}{
		{
			"get timestamp at height exists",
			clientID,
			suite.solomachine.ClientState().ConsensusState.Timestamp,
			nil,
		},
		{
			"client not found",
			"non existent client",
			0,
			clienttypes.ErrClientNotFound,
		},
	}

	for i, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			ctx := suite.chainA.GetContext()

			// Set a client state in store and grab light client module for _clientID_, the lookup for the timestamp
			// is performed on the _tc.clientID_.
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(ctx, clientID, suite.solomachine.ClientState())

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			ts, err := lightClientModule.TimestampAtHeight(ctx, tc.clientID, height)

			suite.Require().Equal(tc.expValue, ts)
			suite.Require().ErrorIs(err, tc.expErr, "valid test case %d failed: %s", i, tc.name)
		})
	}
}

func (suite *SoloMachineTestSuite) TestInitialize() {
	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {
		malleatedConsensus := sm.ClientState().ConsensusState
		malleatedConsensus.Timestamp += 10

		testCases := []struct {
			name        string
			consState   exported.ConsensusState
			clientState exported.ClientState
			expErr      error
		}{
			{
				"valid consensus state",
				sm.ConsensusState(),
				sm.ClientState(),
				nil,
			},
			{
				"nil consensus state",
				nil,
				sm.ClientState(),
				clienttypes.ErrInvalidConsensus,
			},
			{
				"invalid consensus state: Tendermint consensus state",
				&ibctm.ConsensusState{},
				sm.ClientState(),
				fmt.Errorf("proto: wrong wireType = 0 for field TypeUrl"),
			},
			{
				"invalid consensus state: consensus state does not match consensus state in client",
				malleatedConsensus,
				sm.ClientState(),
				clienttypes.ErrInvalidConsensus,
			},
			{
				"invalid client state: sequence is zero",
				sm.ConsensusState(),
				solomachine.NewClientState(0, sm.ConsensusState()),
				clienttypes.ErrInvalidClient,
			},
			{
				"invalid client state: Tendermint client state",
				sm.ConsensusState(),
				&ibctm.ClientState{},
				fmt.Errorf("proto: wrong wireType = 2 for field IsFrozen"),
			},
		}

		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {
				suite.SetupTest()

				clientStateBz := suite.chainA.Codec.MustMarshal(tc.clientState)
				consStateBz := suite.chainA.Codec.MustMarshal(tc.consState)

				clientID := suite.chainA.App.GetIBCKeeper().ClientKeeper.GenerateClientIdentifier(suite.chainA.GetContext(), exported.Solomachine)

				lcm, found := suite.chainA.GetSimApp().IBCKeeper.ClientKeeper.Route(clientID)
				suite.Require().True(found)

				err := lcm.Initialize(suite.chainA.GetContext(), clientID, clientStateBz, consStateBz)
				store := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), clientID)

				expPass := tc.expErr == nil
				if expPass {
					suite.Require().NoError(err, "valid testcase: %s failed", tc.name)
					suite.Require().True(store.Has(host.ClientStateKey()))
				} else {
					suite.Require().ErrorContains(err, tc.expErr.Error())
					suite.Require().False(store.Has(host.ClientStateKey()))
				}
			})
		}
	}
}

func (suite *SoloMachineTestSuite) TestVerifyMembership() {
	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {

		var (
			clientState *solomachine.ClientState
			path        exported.Path
			proof       []byte
			testingPath *ibctesting.Path
			signBytes   solomachine.SignBytes
			err         error
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
				"success: client state verification",
				func() {
					clientState = sm.ClientState()
					clientStateBz, err := suite.chainA.Codec.Marshal(clientState)
					suite.Require().NoError(err)

					path = sm.GetClientStatePath(counterpartyClientIdentifier)
					merklePath, ok := path.(commitmenttypes.MerklePath)
					suite.Require().True(ok)
					key, err := merklePath.GetKey(1) // in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
					suite.Require().NoError(err)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.GetHeight().GetRevisionHeight(),
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        key,
						Data:        clientStateBz,
					}

					signBz, err := suite.chainA.Codec.Marshal(&signBytes)
					suite.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)

					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: sig,
						Timestamp:     sm.Time,
					}

					proof, err = suite.chainA.Codec.Marshal(signatureDoc)
					suite.Require().NoError(err)
				},
				nil,
			},
			{
				"success: consensus state verification",
				func() {
					clientState = sm.ClientState()
					consensusState := clientState.ConsensusState
					consensusStateBz, err := suite.chainA.Codec.Marshal(consensusState)
					suite.Require().NoError(err)

					path = sm.GetConsensusStatePath(counterpartyClientIdentifier, clienttypes.NewHeight(0, 1))
					merklePath, ok := path.(commitmenttypes.MerklePath)
					suite.Require().True(ok)
					key, err := merklePath.GetKey(1) // in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
					suite.Require().NoError(err)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        key,
						Data:        consensusStateBz,
					}

					signBz, err := suite.chainA.Codec.Marshal(&signBytes)
					suite.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)

					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: sig,
						Timestamp:     sm.Time,
					}

					proof, err = suite.chainA.Codec.Marshal(signatureDoc)
					suite.Require().NoError(err)
				},
				nil,
			},
			{
				"success: connection state verification",
				func() {
					testingPath.SetupConnections()

					connectionEnd, found := suite.chainA.GetSimApp().IBCKeeper.ConnectionKeeper.GetConnection(suite.chainA.GetContext(), ibctesting.FirstConnectionID)
					suite.Require().True(found)

					connectionEndBz, err := suite.chainA.Codec.Marshal(&connectionEnd)
					suite.Require().NoError(err)

					path = sm.GetConnectionStatePath(ibctesting.FirstConnectionID)
					merklePath, ok := path.(commitmenttypes.MerklePath)
					suite.Require().True(ok)
					key, err := merklePath.GetKey(1) // in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
					suite.Require().NoError(err)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        key,
						Data:        connectionEndBz,
					}

					signBz, err := suite.chainA.Codec.Marshal(&signBytes)
					suite.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)

					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: sig,
						Timestamp:     sm.Time,
					}

					proof, err = suite.chainA.Codec.Marshal(signatureDoc)
					suite.Require().NoError(err)
				},
				nil,
			},
			{
				"success: channel state verification",
				func() {
					testingPath.SetupConnections()
					suite.coordinator.CreateMockChannels(testingPath)

					channelEnd, found := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetChannel(suite.chainA.GetContext(), ibctesting.MockPort, ibctesting.FirstChannelID)
					suite.Require().True(found)

					channelEndBz, err := suite.chainA.Codec.Marshal(&channelEnd)
					suite.Require().NoError(err)

					path = sm.GetChannelStatePath(ibctesting.MockPort, ibctesting.FirstChannelID)
					merklePath, ok := path.(commitmenttypes.MerklePath)
					suite.Require().True(ok)
					key, err := merklePath.GetKey(1) // in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
					suite.Require().NoError(err)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        key,
						Data:        channelEndBz,
					}

					signBz, err := suite.chainA.Codec.Marshal(&signBytes)
					suite.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)

					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: sig,
						Timestamp:     sm.Time,
					}

					proof, err = suite.chainA.Codec.Marshal(signatureDoc)
					suite.Require().NoError(err)
				},
				nil,
			},
			{
				"success: next sequence recv verification",
				func() {
					testingPath.SetupConnections()
					suite.coordinator.CreateMockChannels(testingPath)

					nextSeqRecv, found := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetNextSequenceRecv(suite.chainA.GetContext(), ibctesting.MockPort, ibctesting.FirstChannelID)
					suite.Require().True(found)

					path = sm.GetNextSequenceRecvPath(ibctesting.MockPort, ibctesting.FirstChannelID)
					merklePath, ok := path.(commitmenttypes.MerklePath)
					suite.Require().True(ok)
					key, err := merklePath.GetKey(1) // in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
					suite.Require().NoError(err)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        key,
						Data:        sdk.Uint64ToBigEndian(nextSeqRecv),
					}

					signBz, err := suite.chainA.Codec.Marshal(&signBytes)
					suite.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)

					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: sig,
						Timestamp:     sm.Time,
					}

					proof, err = suite.chainA.Codec.Marshal(signatureDoc)
					suite.Require().NoError(err)
				},
				nil,
			},
			{
				"success: packet commitment verification",
				func() {
					packet := channeltypes.NewPacket(
						ibctesting.MockPacketData,
						1,
						ibctesting.MockPort,
						ibctesting.FirstChannelID,
						ibctesting.MockPort,
						ibctesting.FirstChannelID,
						clienttypes.NewHeight(0, 10),
						0,
					)

					commitmentBz := channeltypes.CommitPacket(suite.chainA.Codec, packet)
					path = sm.GetPacketCommitmentPath(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
					merklePath, ok := path.(commitmenttypes.MerklePath)
					suite.Require().True(ok)
					key, err := merklePath.GetKey(1) // in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
					suite.Require().NoError(err)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        key,
						Data:        commitmentBz,
					}

					signBz, err := suite.chainA.Codec.Marshal(&signBytes)
					suite.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)

					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: sig,
						Timestamp:     sm.Time,
					}

					proof, err = suite.chainA.Codec.Marshal(signatureDoc)
					suite.Require().NoError(err)
				},
				nil,
			},
			{
				"success: packet acknowledgement verification",
				func() {
					path = sm.GetPacketAcknowledgementPath(ibctesting.MockPort, ibctesting.FirstChannelID, 1)
					merklePath, ok := path.(commitmenttypes.MerklePath)
					suite.Require().True(ok)
					key, err := merklePath.GetKey(1) // index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
					suite.Require().NoError(err)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        key,
						Data:        ibctesting.MockAcknowledgement,
					}

					signBz, err := suite.chainA.Codec.Marshal(&signBytes)
					suite.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)

					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: sig,
						Timestamp:     sm.Time,
					}

					proof, err = suite.chainA.Codec.Marshal(signatureDoc)
					suite.Require().NoError(err)
				},
				nil,
			},
			{
				"success: packet receipt verification",
				func() {
					path = sm.GetPacketReceiptPath(ibctesting.MockPort, ibctesting.FirstChannelID, 1)
					merklePath, ok := path.(commitmenttypes.MerklePath)
					suite.Require().True(ok)
					key, err := merklePath.GetKey(1) // in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
					suite.Require().NoError(err)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        key,
						Data:        []byte{byte(1)}, // packet receipt is stored as a single byte
					}

					signBz, err := suite.chainA.Codec.Marshal(&signBytes)
					suite.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)

					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: sig,
						Timestamp:     sm.Time,
					}

					proof, err = suite.chainA.Codec.Marshal(signatureDoc)
					suite.Require().NoError(err)
				},
				nil,
			},
			// TODO: Cov missing, need a more extensive refactor of tests in order to cov it.
			// {
			// 	"client not found",
			// 	func() {},
			// 	clienttypes.ErrClientNotFound,
			// },
			{
				"invalid path type - empty",
				func() {
					path = ibcmock.KeyPath{}
				},
				ibcerrors.ErrInvalidType,
			},
			{
				"malformed proof fails to unmarshal",
				func() {
					path = sm.GetClientStatePath(counterpartyClientIdentifier)
					proof = []byte("invalid proof")
				},
				fmt.Errorf("failed to unmarshal proof into type"),
			},
			{
				"consensus state timestamp is greater than signature",
				func() {
					consensusState := &solomachine.ConsensusState{
						Timestamp: sm.Time + 1,
						PublicKey: sm.ConsensusState().PublicKey,
					}

					clientState = solomachine.NewClientState(sm.Sequence, consensusState)
				},
				fmt.Errorf("the consensus state timestamp is greater than the signature timestamp (11 >= 10): %s", solomachine.ErrInvalidProof),
			},
			{
				"signature data is nil",
				func() {
					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: nil,
						Timestamp:     sm.Time,
					}

					proof, err = suite.chainA.Codec.Marshal(signatureDoc)
					suite.Require().NoError(err)
				},
				fmt.Errorf("signature data cannot be empty: %s", solomachine.ErrInvalidProof),
			},
			{
				"consensus state public key is nil",
				func() {
					clientState.ConsensusState.PublicKey = nil
				},
				fmt.Errorf("consensus state PublicKey cannot be nil: %s", clienttypes.ErrInvalidConsensus),
			},
			{
				"malformed signature data fails to unmarshal",
				func() {
					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: []byte("invalid signature data"),
						Timestamp:     sm.Time,
					}

					proof, err = suite.chainA.Codec.Marshal(signatureDoc)
					suite.Require().NoError(err)
				},
				fmt.Errorf("failed to unmarshal proof into type"),
			},
			{
				"proof is nil",
				func() {
					proof = nil
				},
				fmt.Errorf("proof cannot be empty: %s", solomachine.ErrInvalidProof),
			},
			{
				"proof verification failed",
				func() {
					signBytes.Data = []byte("invalid membership data value")
				},
				solomachine.ErrSignatureVerificationFailed,
			},
			{
				"empty path",
				func() {
					path = commitmenttypes.MerklePath{}
				},
				fmt.Errorf("path must be of length 2: []: %s", host.ErrInvalidPath),
			},
		}

		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {
				suite.SetupTest()
				testingPath = ibctesting.NewPath(suite.chainA, suite.chainB)

				clientState = sm.ClientState()

				path = commitmenttypes.NewMerklePath("ibc", "solomachine")
				merklePath, ok := path.(commitmenttypes.MerklePath)
				suite.Require().True(ok)
				key, err := merklePath.GetKey(1) // in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
				suite.Require().NoError(err)
				signBytes = solomachine.SignBytes{
					Sequence:    sm.GetHeight().GetRevisionHeight(),
					Timestamp:   sm.Time,
					Diversifier: sm.Diversifier,
					Path:        key,
					Data:        []byte("solomachine"),
				}

				signBz, err := suite.chainA.Codec.Marshal(&signBytes)
				suite.Require().NoError(err)

				sig := sm.GenerateSignature(signBz)

				signatureDoc := &solomachine.TimestampedSignatureData{
					SignatureData: sig,
					Timestamp:     sm.Time,
				}

				proof, err = suite.chainA.Codec.Marshal(signatureDoc)
				suite.Require().NoError(err)

				tc.malleate()

				// Generate clientID
				clientID := sm.ClientID

				// Set the client state in the store for light client call to find.
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, clientState)

				var expSeq uint64
				if clientState.ConsensusState != nil {
					expSeq = clientState.Sequence + 1
				}

				lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(smClientID)
				suite.Require().True(found)

				// Verify the membership proof
				err = lightClientModule.VerifyMembership(
					suite.chainA.GetContext(), clientID, clienttypes.ZeroHeight(),
					0, 0, proof, path, signBytes.Data,
				)

				// Grab fresh client state after updates.
				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), clientID)
				suite.Require().True(found)
				clientState = cs.(*solomachine.ClientState)

				expPass := tc.expErr == nil
				if expPass {
					suite.Require().NoError(err)
					// clientState.Sequence is the most recent view of state.
					suite.Require().Equal(expSeq, clientState.Sequence)
				} else {
					suite.Require().Error(err)
					suite.Require().ErrorContains(err, tc.expErr.Error())
				}
			})
		}
	}
}

func (suite *SoloMachineTestSuite) TestVerifyNonMembership() {
	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {

		var (
			clientState *solomachine.ClientState
			path        exported.Path
			proof       []byte
			signBytes   solomachine.SignBytes
			err         error
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
				"success: packet receipt absence verification",
				func() {
					path = sm.GetPacketReceiptPath(ibctesting.MockPort, ibctesting.FirstChannelID, 1)
					merklePath, ok := path.(commitmenttypes.MerklePath)
					suite.Require().True(ok)
					key, err := merklePath.GetKey(1) // in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
					suite.Require().NoError(err)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.GetHeight().GetRevisionHeight(),
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        key,
						Data:        nil,
					}

					signBz, err := suite.chainA.Codec.Marshal(&signBytes)
					suite.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)

					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: sig,
						Timestamp:     sm.Time,
					}

					proof, err = suite.chainA.Codec.Marshal(signatureDoc)
					suite.Require().NoError(err)
				},
				nil,
			},
			// TODO: Cov missing, need a more extensive refactor of tests in order to cov it.
			// {
			// 	"client not found",
			// 	func() {},
			// 	clienttypes.ErrClientNotFound,
			// },
			{
				"invalid path type",
				func() {
					path = ibcmock.KeyPath{}
				},
				ibcerrors.ErrInvalidType,
			},
			{
				"malformed proof fails to unmarshal",
				func() {
					path = sm.GetClientStatePath(counterpartyClientIdentifier)
					proof = []byte("invalid proof")
				},
				fmt.Errorf("failed to unmarshal proof into type"),
			},
			{
				"consensus state timestamp is greater than signature",
				func() {
					consensusState := &solomachine.ConsensusState{
						Timestamp: sm.Time + 1,
						PublicKey: sm.ConsensusState().PublicKey,
					}

					clientState = solomachine.NewClientState(sm.Sequence, consensusState)
				},
				fmt.Errorf("the consensus state timestamp is greater than the signature timestamp (11 >= 10): %s", solomachine.ErrInvalidProof),
			},
			{
				"signature data is nil",
				func() {
					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: nil,
						Timestamp:     sm.Time,
					}

					proof, err = suite.chainA.Codec.Marshal(signatureDoc)
					suite.Require().NoError(err)
				},
				fmt.Errorf("signature data cannot be empty: %s", solomachine.ErrInvalidProof),
			},
			{
				"consensus state public key is nil",
				func() {
					clientState.ConsensusState.PublicKey = nil
				},
				fmt.Errorf("consensus state PublicKey cannot be nil: %s", clienttypes.ErrInvalidConsensus),
			},
			{
				"malformed signature data fails to unmarshal",
				func() {
					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: []byte("invalid signature data"),
						Timestamp:     sm.Time,
					}

					proof, err = suite.chainA.Codec.Marshal(signatureDoc)
					suite.Require().NoError(err)
				},
				fmt.Errorf("failed to unmarshal proof into type"),
			},
			{
				"proof is nil",
				func() {
					proof = nil
				},
				fmt.Errorf("proof cannot be empty: %s", solomachine.ErrInvalidProof),
			},
			{
				"proof verification failed",
				func() {
					signBytes.Data = []byte("invalid non-membership data value")

					signBz, err := suite.chainA.Codec.Marshal(&signBytes)
					suite.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)

					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: sig,
						Timestamp:     sm.Time,
					}

					proof, err = suite.chainA.Codec.Marshal(signatureDoc)
					suite.Require().NoError(err)
				},
				solomachine.ErrSignatureVerificationFailed,
			},
		}

		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {
				clientState = sm.ClientState()

				path = commitmenttypes.NewMerklePath("ibc", "solomachine")
				merklePath, ok := path.(commitmenttypes.MerklePath)
				suite.Require().True(ok)
				key, err := merklePath.GetKey(1) // in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
				suite.Require().NoError(err)
				signBytes = solomachine.SignBytes{
					Sequence:    sm.GetHeight().GetRevisionHeight(),
					Timestamp:   sm.Time,
					Diversifier: sm.Diversifier,
					Path:        key,
					Data:        nil,
				}

				signBz, err := suite.chainA.Codec.Marshal(&signBytes)
				suite.Require().NoError(err)

				sig := sm.GenerateSignature(signBz)

				signatureDoc := &solomachine.TimestampedSignatureData{
					SignatureData: sig,
					Timestamp:     sm.Time,
				}

				proof, err = suite.chainA.Codec.Marshal(signatureDoc)
				suite.Require().NoError(err)

				tc.malleate()

				var expSeq uint64
				if clientState.ConsensusState != nil {
					expSeq = clientState.Sequence + 1
				}

				// Generate clientID
				clientID := sm.ClientID

				// Set the client state in the store for light client call to find.
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, clientState)

				lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(smClientID)
				suite.Require().True(found)

				// Verify the membership proof
				err = lightClientModule.VerifyNonMembership(
					suite.chainA.GetContext(), clientID, clienttypes.ZeroHeight(),
					0, 0, proof, path,
				)

				// Grab fresh client state after updates.
				cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), clientID)
				suite.Require().True(found)
				clientState = cs.(*solomachine.ClientState)

				expPass := tc.expErr == nil
				if expPass {
					suite.Require().NoError(err)
					suite.Require().Equal(expSeq, clientState.Sequence)
				} else {
					suite.Require().Error(err)
					suite.Require().ErrorContains(err, tc.expErr.Error())
				}
			})
		}
	}
}

func (suite *SoloMachineTestSuite) TestRecoverClient() {
	var (
		subjectClientID, substituteClientID       string
		subjectClientState, substituteClientState *solomachine.ClientState
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
			"cannot parse malformed substitute client ID",
			func() {
				substituteClientID = ibctesting.InvalidID
			},
			host.ErrInvalidID,
		},
		{
			"substitute client ID does not contain 06-solomachine prefix",
			func() {
				substituteClientID = wasmClientID
			},
			clienttypes.ErrInvalidClientType,
		},
		{
			"cannot find subject client state",
			func() {
				subjectClientID = smClientID
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"cannot find substitute client state",
			func() {
				substituteClientID = smClientID
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			cdc := suite.chainA.Codec
			ctx := suite.chainA.GetContext()

			subjectClientID = suite.chainA.App.GetIBCKeeper().ClientKeeper.GenerateClientIdentifier(ctx, exported.Solomachine)
			subject := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, substituteClientID, "testing", 1)
			subjectClientState = subject.ClientState()

			substituteClientID = suite.chainA.App.GetIBCKeeper().ClientKeeper.GenerateClientIdentifier(ctx, exported.Solomachine)
			substitute := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, substituteClientID, "testing", 1)
			substitute.Sequence++ // increase sequence so that latest height of substitute is > than subject's latest height
			substituteClientState = substitute.ClientState()

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, substituteClientID)
			clientStore.Get(host.ClientStateKey())
			bz := clienttypes.MustMarshalClientState(cdc, substituteClientState)
			clientStore.Set(host.ClientStateKey(), bz)

			subjectClientState.IsFrozen = true
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(ctx, subjectClientID, subjectClientState)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(subjectClientID)
			suite.Require().True(found)

			tc.malleate()

			err := lightClientModule.RecoverClient(ctx, subjectClientID, substituteClientID)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				// assert that status of subject client is now Active
				clientStore = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, subjectClientID)
				bz = clientStore.Get(host.ClientStateKey())
				smClientState := clienttypes.MustUnmarshalClientState(cdc, bz).(*solomachine.ClientState)

				suite.Require().Equal(substituteClientState.ConsensusState, smClientState.ConsensusState)
				suite.Require().Equal(substituteClientState.Sequence, smClientState.Sequence)
				suite.Require().Equal(exported.Active, lightClientModule.Status(ctx, subjectClientID))
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *SoloMachineTestSuite) TestVerifyUpgradeAndUpdateState() {
	clientID := suite.solomachine.ClientID

	lightClientModule, found := suite.chainA.GetSimApp().IBCKeeper.ClientKeeper.Route(clientID)
	suite.Require().True(found)

	err := lightClientModule.VerifyUpgradeAndUpdateState(suite.chainA.GetContext(), clientID, nil, nil, nil, nil)
	suite.Require().Error(err)
}
