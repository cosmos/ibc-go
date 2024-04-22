package solomachine_test

import (
	"fmt"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
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
	unusedSmClientID = "06-solomachine-999"
	wasmClientID     = "08-wasm-0"
)

func (suite *SoloMachineTestSuite) TestStatus() {
	var (
		clientState *solomachine.ClientState
		clientID    string
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
				clientState = solomachine.NewClientState(0, &solomachine.ConsensusState{})
				clientState.IsFrozen = true
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, clientState)
			},
			exported.Frozen,
		},
		{
			"failure: cannot find client state",
			func() {
				clientID = unusedSmClientID
			},
			exported.Unknown,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			clientID = suite.solomachine.ClientID

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, suite.solomachine.ClientState())

			tc.malleate()

			status := lightClientModule.Status(suite.chainA.GetContext(), clientID)
			suite.Require().Equal(tc.expStatus, status)
		})
	}
}

func (suite *SoloMachineTestSuite) TestGetTimestampAtHeight() {
	var (
		clientID string
		height   exported.Height
	)

	testCases := []struct {
		name     string
		malleate func()
		expValue uint64
		expErr   error
	}{
		{
			"success: get timestamp at height exists",
			func() {},
			suite.solomachine.ClientState().ConsensusState.Timestamp,
			nil,
		},
		{
			"success: modified height",
			func() {
				height = clienttypes.ZeroHeight()
			},
			// Timestamp should be the same.
			suite.solomachine.ClientState().ConsensusState.Timestamp,
			nil,
		},
		{
			"failure: cannot find client state",
			func() {
				clientID = unusedSmClientID
			},
			0,
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			clientID = suite.solomachine.ClientID
			clientState := suite.solomachine.ClientState()
			height = clienttypes.NewHeight(0, suite.solomachine.ClientState().Sequence)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, clientState)

			tc.malleate()

			ts, err := lightClientModule.TimestampAtHeight(suite.chainA.GetContext(), clientID, height)

			suite.Require().Equal(tc.expValue, ts)
			suite.Require().ErrorIs(err, tc.expErr)
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
				"success: valid consensus state",
				sm.ConsensusState(),
				sm.ClientState(),
				nil,
			},
			{
				"failure: nil consensus state",
				nil,
				sm.ClientState(),
				clienttypes.ErrInvalidConsensus,
			},
			{
				"failure: invalid consensus state: Tendermint consensus state",
				&ibctm.ConsensusState{},
				sm.ClientState(),
				fmt.Errorf("proto: wrong wireType = 0 for field TypeUrl"),
			},
			{
				"failure: invalid consensus state: consensus state does not match consensus state in client",
				malleatedConsensus,
				sm.ClientState(),
				clienttypes.ErrInvalidConsensus,
			},
			{
				"failure: invalid client state: sequence is zero",
				sm.ConsensusState(),
				solomachine.NewClientState(0, sm.ConsensusState()),
				clienttypes.ErrInvalidClient,
			},
			{
				"failure: invalid client state: Tendermint client state",
				sm.ConsensusState(),
				&ibctm.ClientState{},
				fmt.Errorf("proto: wrong wireType = 2 for field IsFrozen"),
			},
		}

		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {
				suite.SetupTest()
				clientID := sm.ClientID

				clientStateBz := suite.chainA.Codec.MustMarshal(tc.clientState)
				consStateBz := suite.chainA.Codec.MustMarshal(tc.consState)

				lcm, found := suite.chainA.GetSimApp().IBCKeeper.ClientKeeper.Route(clientID)
				suite.Require().True(found)

				err := lcm.Initialize(suite.chainA.GetContext(), clientID, clientStateBz, consStateBz)
				store := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), clientID)

				expPass := tc.expErr == nil
				if expPass {
					suite.Require().NoError(err)
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
	var (
		clientState *solomachine.ClientState
		path        exported.Path
		proof       []byte
		testingPath *ibctesting.Path
		signBytes   solomachine.SignBytes
		err         error
		clientID    string
	)

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {

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
					clientStateBz, err := suite.chainA.Codec.MarshalInterface(clientState)
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
					consensusStateBz, err := suite.chainA.Codec.MarshalInterface(consensusState)
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
			{
				"failure: cannot find client state",
				func() {
					clientID = unusedSmClientID
				},
				clienttypes.ErrClientNotFound,
			},
			{
				"failure: invalid path type - empty",
				func() {
					path = ibcmock.KeyPath{}
				},
				ibcerrors.ErrInvalidType,
			},
			{
				"failure: malformed proof fails to unmarshal",
				func() {
					path = sm.GetClientStatePath(counterpartyClientIdentifier)
					proof = []byte("invalid proof")
				},
				fmt.Errorf("failed to unmarshal proof into type"),
			},
			{
				"failure: consensus state timestamp is greater than signature",
				func() {
					consensusState := &solomachine.ConsensusState{
						Timestamp: sm.Time + 1,
						PublicKey: sm.ConsensusState().PublicKey,
					}

					clientState = solomachine.NewClientState(sm.Sequence, consensusState)
					suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, clientState)
				},
				fmt.Errorf("the consensus state timestamp is greater than the signature timestamp (11 >= 10): %s", solomachine.ErrInvalidProof),
			},
			{
				"failure: signature data is nil",
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
				"failure: consensus state public key is nil",
				func() {
					clientState.ConsensusState.PublicKey = nil
					suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, clientState)
				},
				fmt.Errorf("consensus state PublicKey cannot be nil: %s", clienttypes.ErrInvalidConsensus),
			},
			{
				"failure: malformed signature data fails to unmarshal",
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
				"failure: proof is nil",
				func() {
					proof = nil
				},
				fmt.Errorf("proof cannot be empty: %s", solomachine.ErrInvalidProof),
			},
			{
				"failure: proof verification failed",
				func() {
					signBytes.Data = []byte("invalid membership data value")
				},
				solomachine.ErrSignatureVerificationFailed,
			},
			{
				"failure: empty path",
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

				clientID = sm.ClientID
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

				lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
				suite.Require().True(found)

				// Set the client state in the store for light client call to find.
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, clientState)

				tc.malleate()

				var expSeq uint64
				if clientState.ConsensusState != nil {
					expSeq = clientState.Sequence + 1
				}

				// Verify the membership proof
				err = lightClientModule.VerifyMembership(
					suite.chainA.GetContext(), clientID, clienttypes.ZeroHeight(),
					0, 0, proof, path, signBytes.Data,
				)

				expPass := tc.expErr == nil
				if expPass {
					// Grab fresh client state after updates.
					cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), clientID)
					suite.Require().True(found)
					clientState, ok = cs.(*solomachine.ClientState)
					suite.Require().True(ok)

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
	var (
		clientState *solomachine.ClientState
		path        exported.Path
		proof       []byte
		signBytes   solomachine.SignBytes
		err         error
		clientID    string
	)

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {
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
			{
				"failure: cannot find client state",
				func() {
					clientID = unusedSmClientID
				},
				clienttypes.ErrClientNotFound,
			},
			{
				"failure: invalid path type",
				func() {
					path = ibcmock.KeyPath{}
				},
				ibcerrors.ErrInvalidType,
			},
			{
				"failure: malformed proof fails to unmarshal",
				func() {
					path = sm.GetClientStatePath(counterpartyClientIdentifier)
					proof = []byte("invalid proof")
				},
				fmt.Errorf("failed to unmarshal proof into type"),
			},
			{
				"failure: consensus state timestamp is greater than signature",
				func() {
					consensusState := &solomachine.ConsensusState{
						Timestamp: sm.Time + 1,
						PublicKey: sm.ConsensusState().PublicKey,
					}

					clientState = solomachine.NewClientState(sm.Sequence, consensusState)
					suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, clientState)
				},
				fmt.Errorf("the consensus state timestamp is greater than the signature timestamp (11 >= 10): %s", solomachine.ErrInvalidProof),
			},
			{
				"failure: signature data is nil",
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
				"failure: consensus state public key is nil",
				func() {
					clientState.ConsensusState.PublicKey = nil
					suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, clientState)
				},
				fmt.Errorf("consensus state PublicKey cannot be nil: %s", clienttypes.ErrInvalidConsensus),
			},
			{
				"failure: malformed signature data fails to unmarshal",
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
				"failure: proof is nil",
				func() {
					proof = nil
				},
				fmt.Errorf("proof cannot be empty: %s", solomachine.ErrInvalidProof),
			},
			{
				"failure: proof verification failed",
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
				suite.SetupTest()

				clientState = sm.ClientState()
				clientID = sm.ClientID

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

				lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
				suite.Require().True(found)

				// Set the client state in the store for light client call to find.
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, clientState)

				tc.malleate()

				var expSeq uint64
				if clientState.ConsensusState != nil {
					expSeq = clientState.Sequence + 1
				}

				// Verify the membership proof
				err = lightClientModule.VerifyNonMembership(
					suite.chainA.GetContext(), clientID, clienttypes.ZeroHeight(),
					0, 0, proof, path,
				)

				expPass := tc.expErr == nil
				if expPass {
					// Grab fresh client state after updates.
					cs, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), clientID)
					suite.Require().True(found)
					clientState, ok = cs.(*solomachine.ClientState)
					suite.Require().True(ok)

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
			"failure: cannot parse malformed substitute client ID",
			func() {
				substituteClientID = ibctesting.InvalidID
			},
			host.ErrInvalidID,
		},
		{
			"failure: substitute client ID does not contain 06-solomachine prefix",
			func() {
				substituteClientID = wasmClientID
			},
			clienttypes.ErrInvalidClientType,
		},
		{
			"failure: cannot find subject client state",
			func() {
				subjectClientID = unusedSmClientID
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"failure: cannot find substitute client state",
			func() {
				substituteClientID = unusedSmClientID
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

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
			bz := clienttypes.MustMarshalClientState(suite.chainA.Codec, substituteClientState)
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
				smClientState, ok := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, bz).(*solomachine.ClientState)
				suite.Require().True(ok)

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

func (suite *SoloMachineTestSuite) TestUpdateState() {
	var (
		clientState *solomachine.ClientState
		clientMsg   exported.ClientMessage
		clientID    string
	)

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {

		testCases := []struct {
			name     string
			malleate func()
			expPanic error
		}{
			{
				"successful update",
				func() {},
				nil,
			},
			{
				"failure: invalid type misbehaviour",
				func() {
					clientState = sm.ClientState()
					clientMsg = sm.CreateMisbehaviour()
					suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, clientState)
				},
				fmt.Errorf("unsupported ClientMessage: %T", sm.CreateMisbehaviour()),
			},
			{
				"failure: cannot find client state",
				func() {
					clientID = unusedSmClientID
				},
				fmt.Errorf("%s: %s", unusedSmClientID, clienttypes.ErrClientNotFound),
			},
		}

		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {
				suite.SetupTest()

				clientID = sm.ClientID
				clientState = sm.ClientState()
				clientMsg = sm.CreateHeader(sm.Diversifier)

				lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
				suite.Require().True(found)

				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, clientState)

				tc.malleate() // setup test

				store := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), clientID)

				var consensusHeights []exported.Height
				updateStateFunc := func() {
					consensusHeights = lightClientModule.UpdateState(suite.chainA.GetContext(), clientID, clientMsg)
				}

				expPass := tc.expPanic == nil
				if expPass {
					updateStateFunc()

					clientStateBz := store.Get(host.ClientStateKey())
					suite.Require().NotEmpty(clientStateBz)

					newClientState := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, clientStateBz)

					suite.Require().Len(consensusHeights, 1)
					suite.Require().Equal(uint64(0), consensusHeights[0].GetRevisionNumber())
					suite.Require().Equal(newClientState.(*solomachine.ClientState).Sequence, consensusHeights[0].GetRevisionHeight())

					suite.Require().False(newClientState.(*solomachine.ClientState).IsFrozen)
					suite.Require().Equal(clientMsg.(*solomachine.Header).NewPublicKey, newClientState.(*solomachine.ClientState).ConsensusState.PublicKey)
					suite.Require().Equal(clientMsg.(*solomachine.Header).NewDiversifier, newClientState.(*solomachine.ClientState).ConsensusState.Diversifier)
					suite.Require().Equal(clientMsg.(*solomachine.Header).Timestamp, newClientState.(*solomachine.ClientState).ConsensusState.Timestamp)
				} else {
					suite.Require().PanicsWithError(tc.expPanic.Error(), updateStateFunc)
				}
			})
		}
	}
}

func (suite *SoloMachineTestSuite) TestCheckForMisbehaviour() {
	var (
		clientMsg exported.ClientMessage
		clientID  string
	)

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {
		testCases := []struct {
			name              string
			malleate          func()
			foundMisbehaviour bool
			expPanic          error
		}{
			{
				"success",
				func() {
					clientMsg = sm.CreateMisbehaviour()
				},
				true,
				nil,
			},
			{
				"failure: normal header returns false",
				func() {
					clientMsg = sm.CreateHeader(sm.Diversifier)
				},
				false,
				nil,
			},
			{
				"failure: cannot find client state",
				func() {
					clientID = unusedSmClientID
				},
				false,
				fmt.Errorf("%s: %s", unusedSmClientID, clienttypes.ErrClientNotFound),
			},
		}

		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {
				suite.SetupTest()

				clientID = sm.ClientID

				lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
				suite.Require().True(found)

				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, sm.ClientState())

				tc.malleate()

				var foundMisbehaviour bool
				foundMisbehaviourFunc := func() {
					foundMisbehaviour = lightClientModule.CheckForMisbehaviour(suite.chainA.GetContext(), clientID, clientMsg)
				}

				expPass := tc.expPanic == nil
				if expPass {
					foundMisbehaviourFunc()

					suite.Require().Equal(tc.foundMisbehaviour, foundMisbehaviour)
				} else {
					suite.Require().PanicsWithError(tc.expPanic.Error(), foundMisbehaviourFunc)
					suite.Require().False(foundMisbehaviour)
				}
			})
		}
	}
}

func (suite *SoloMachineTestSuite) TestUpdateStateOnMisbehaviour() {
	var clientID string

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {
		testCases := []struct {
			name     string
			malleate func()
			expPanic error
		}{
			{
				"success",
				func() {},
				nil,
			},
			{
				"failure: cannot find client state",
				func() {
					clientID = unusedSmClientID
				},
				fmt.Errorf("%s: %s", unusedSmClientID, clienttypes.ErrClientNotFound),
			},
		}

		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {
				suite.SetupTest()
				clientID = sm.ClientID

				lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
				suite.Require().True(found)

				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, sm.ClientState())

				tc.malleate()

				updateOnMisbehaviourFunc := func() {
					lightClientModule.UpdateStateOnMisbehaviour(suite.chainA.GetContext(), clientID, nil)
				}

				expPass := tc.expPanic == nil
				if expPass {
					updateOnMisbehaviourFunc()

					store := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), clientID)

					clientStateBz := store.Get(host.ClientStateKey())
					suite.Require().NotEmpty(clientStateBz)

					newClientState := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, clientStateBz)

					suite.Require().True(newClientState.(*solomachine.ClientState).IsFrozen)
				} else {
					suite.Require().PanicsWithError(tc.expPanic.Error(), updateOnMisbehaviourFunc)
				}
			})
		}
	}
}

func (suite *SoloMachineTestSuite) TestVerifyClientMessageHeader() {
	var (
		clientID  string
		clientMsg exported.ClientMessage
	)

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {

		testCases := []struct {
			name     string
			malleate func()
			expErr   error
		}{
			{
				"success: successful header",
				func() {
					clientMsg = sm.CreateHeader(sm.Diversifier)
				},
				nil,
			},
			{
				"success: successful header with new diversifier",
				func() {
					clientMsg = sm.CreateHeader(sm.Diversifier + "0")
				},
				nil,
			},
			{
				"success: successful misbehaviour",
				func() {
					clientMsg = sm.CreateMisbehaviour()
				},
				nil,
			},
			{
				"failure: invalid client message type",
				func() {
					clientMsg = &ibctm.Header{}
				},
				clienttypes.ErrInvalidClientType,
			},
			{
				"failure: invalid header Signature",
				func() {
					h := sm.CreateHeader(sm.Diversifier)
					h.Signature = suite.GetInvalidProof()
					clientMsg = h
				}, fmt.Errorf("proto: wrong wireType = 0 for field Multi"),
			},
			{
				"failure: invalid timestamp in header",
				func() {
					h := sm.CreateHeader(sm.Diversifier)
					h.Timestamp--
					clientMsg = h
				}, clienttypes.ErrInvalidHeader,
			},
			{
				"failure: signature uses wrong sequence",
				func() {
					sm.Sequence++
					clientMsg = sm.CreateHeader(sm.Diversifier)
				},
				solomachine.ErrSignatureVerificationFailed,
			},
			{
				"signature uses new pubkey to sign",
				func() {
					// store in temp before assigning to interface type
					cs := sm.ClientState()
					h := sm.CreateHeader(sm.Diversifier)

					publicKey, err := codectypes.NewAnyWithValue(sm.PublicKey)
					suite.NoError(err)

					data := &solomachine.HeaderData{
						NewPubKey:      publicKey,
						NewDiversifier: h.NewDiversifier,
					}

					dataBz, err := suite.chainA.Codec.Marshal(data)
					suite.Require().NoError(err)

					// generate invalid signature
					signBytes := &solomachine.SignBytes{
						Sequence:    cs.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        []byte("invalid signature data"),
						Data:        dataBz,
					}

					signBz, err := suite.chainA.Codec.Marshal(signBytes)
					suite.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)
					suite.Require().NoError(err)
					h.Signature = sig

					clientMsg = h
				},
				solomachine.ErrSignatureVerificationFailed,
			},
			{
				"failure: signature signs over old pubkey",
				func() {
					// store in temp before assigning to interface type
					cs := sm.ClientState()

					oldPubKey := sm.PublicKey
					h := sm.CreateHeader(sm.Diversifier)

					// generate invalid signature
					data := append(sdk.Uint64ToBigEndian(cs.Sequence), oldPubKey.Bytes()...)
					sig := sm.GenerateSignature(data)
					h.Signature = sig

					clientMsg = h
				},
				solomachine.ErrSignatureVerificationFailed,
			},
			{
				"failure: consensus state public key is nil - header",
				func() {
					h := sm.CreateHeader(sm.Diversifier)
					h.NewPublicKey = nil
					clientMsg = h
				},
				solomachine.ErrSignatureVerificationFailed,
			},
			{
				"failure: cannot find client state",
				func() {
					clientID = unusedSmClientID
				},
				fmt.Errorf("%s: %s", unusedSmClientID, clienttypes.ErrClientNotFound),
			},
		}

		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {
				suite.SetupTest()
				clientID = sm.ClientID

				lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
				suite.Require().True(found)

				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, sm.ClientState())

				tc.malleate()

				err := lightClientModule.VerifyClientMessage(suite.chainA.GetContext(), clientID, clientMsg)

				expPass := tc.expErr == nil
				if expPass {
					suite.Require().NoError(err)
				} else {
					suite.Require().ErrorContains(err, tc.expErr.Error())
				}
			})
		}
	}
}

func (suite *SoloMachineTestSuite) TestVerifyClientMessageMisbehaviour() {
	var (
		clientMsg   exported.ClientMessage
		clientState *solomachine.ClientState
		clientID    string
	)

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {

		testCases := []struct {
			name     string
			malleate func()
			expErr   error
		}{
			{
				"success: successful misbehaviour",
				func() {
					clientMsg = sm.CreateMisbehaviour()
				},
				nil,
			},
			{
				"success: old misbehaviour is successful (timestamp is less than current consensus state)",
				func() {
					clientState = sm.ClientState()
					sm.Time -= 5
					clientMsg = sm.CreateMisbehaviour()
				}, nil,
			},
			{
				"failure: invalid client message type",
				func() {
					clientMsg = &ibctm.Header{}
				},
				clienttypes.ErrInvalidClientType,
			},
			{
				"failure: consensus state pubkey is nil",
				func() {
					clientState.ConsensusState.PublicKey = nil
					clientMsg = sm.CreateMisbehaviour()
					suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, clientState)
				},
				clienttypes.ErrInvalidConsensus,
			},
			{
				"failure: invalid SignatureOne SignatureData",
				func() {
					m := sm.CreateMisbehaviour()

					m.SignatureOne.Signature = suite.GetInvalidProof()
					clientMsg = m
				}, fmt.Errorf("proto: wrong wireType = 0 for field Multi"),
			},
			{
				"failure: invalid SignatureTwo SignatureData",
				func() {
					m := sm.CreateMisbehaviour()

					m.SignatureTwo.Signature = suite.GetInvalidProof()
					clientMsg = m
				}, fmt.Errorf("proto: wrong wireType = 0 for field Multi"),
			},
			{
				"failure: invalid SignatureOne timestamp",
				func() {
					m := sm.CreateMisbehaviour()

					m.SignatureOne.Timestamp = 1000000000000
					clientMsg = m
				}, solomachine.ErrSignatureVerificationFailed,
			},
			{
				"failure: invalid SignatureTwo timestamp",
				func() {
					m := sm.CreateMisbehaviour()

					m.SignatureTwo.Timestamp = 1000000000000
					clientMsg = m
				}, solomachine.ErrSignatureVerificationFailed,
			},
			{
				"failure: invalid first signature data",
				func() {
					// store in temp before assigning to interface type
					m := sm.CreateMisbehaviour()

					msg := []byte("DATA ONE")
					signBytes := &solomachine.SignBytes{
						Sequence:    sm.Sequence + 1,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        []byte("invalid signature data"),
						Data:        msg,
					}

					data, err := suite.chainA.Codec.Marshal(signBytes)
					suite.Require().NoError(err)

					sig := sm.GenerateSignature(data)

					m.SignatureOne.Signature = sig
					m.SignatureOne.Data = msg
					clientMsg = m
				},
				solomachine.ErrSignatureVerificationFailed,
			},
			{
				"failure: invalid second signature data",
				func() {
					// store in temp before assigning to interface type
					m := sm.CreateMisbehaviour()

					msg := []byte("DATA TWO")
					signBytes := &solomachine.SignBytes{
						Sequence:    sm.Sequence + 1,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        []byte("invalid signature data"),
						Data:        msg,
					}

					data, err := suite.chainA.Codec.Marshal(signBytes)
					suite.Require().NoError(err)

					sig := sm.GenerateSignature(data)

					m.SignatureTwo.Signature = sig
					m.SignatureTwo.Data = msg
					clientMsg = m
				},
				solomachine.ErrSignatureVerificationFailed,
			},
			{
				"failure: wrong pubkey generates first signature",
				func() {
					badMisbehaviour := sm.CreateMisbehaviour()

					// update public key to a new one
					sm.CreateHeader(sm.Diversifier)
					m := sm.CreateMisbehaviour()

					// set SignatureOne to use the wrong signature
					m.SignatureOne = badMisbehaviour.SignatureOne
					clientMsg = m
				}, solomachine.ErrSignatureVerificationFailed,
			},
			{
				"failure: wrong pubkey generates second signature",
				func() {
					badMisbehaviour := sm.CreateMisbehaviour()

					// update public key to a new one
					sm.CreateHeader(sm.Diversifier)
					m := sm.CreateMisbehaviour()

					// set SignatureTwo to use the wrong signature
					m.SignatureTwo = badMisbehaviour.SignatureTwo
					clientMsg = m
				}, solomachine.ErrSignatureVerificationFailed,
			},
			{
				"failure: signatures sign over different sequence",
				func() {
					// store in temp before assigning to interface type
					m := sm.CreateMisbehaviour()

					// Signature One
					msg := []byte("DATA ONE")
					// sequence used is plus 1
					signBytes := &solomachine.SignBytes{
						Sequence:    sm.Sequence + 1,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        []byte("invalid signature data"),
						Data:        msg,
					}

					data, err := suite.chainA.Codec.Marshal(signBytes)
					suite.Require().NoError(err)

					sig := sm.GenerateSignature(data)

					m.SignatureOne.Signature = sig
					m.SignatureOne.Data = msg

					// Signature Two
					msg = []byte("DATA TWO")
					// sequence used is minus 1

					signBytes = &solomachine.SignBytes{
						Sequence:    sm.Sequence - 1,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        []byte("invalid signature data"),
						Data:        msg,
					}
					data, err = suite.chainA.Codec.Marshal(signBytes)
					suite.Require().NoError(err)

					sig = sm.GenerateSignature(data)

					m.SignatureTwo.Signature = sig
					m.SignatureTwo.Data = msg

					clientMsg = m
				},
				solomachine.ErrSignatureVerificationFailed,
			},
			{
				"failure: cannot find client state",
				func() {
					clientID = unusedSmClientID
				},
				fmt.Errorf("%s: %s", unusedSmClientID, clienttypes.ErrClientNotFound),
			},
		}

		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {
				suite.SetupTest()
				clientID = sm.ClientID

				lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
				suite.Require().True(found)

				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, sm.ClientState())

				tc.malleate()

				err := lightClientModule.VerifyClientMessage(suite.chainA.GetContext(), clientID, clientMsg)

				expPass := tc.expErr == nil
				if expPass {
					suite.Require().NoError(err)
				} else {
					suite.Require().ErrorContains(err, tc.expErr.Error())
				}
			})
		}
	}
}

func (suite *SoloMachineTestSuite) TestVerifyUpgradeAndUpdateState() {
	clientID := suite.solomachine.ClientID

	lightClientModule, found := suite.chainA.GetSimApp().IBCKeeper.ClientKeeper.Route(clientID)
	suite.Require().True(found)

	err := lightClientModule.VerifyUpgradeAndUpdateState(suite.chainA.GetContext(), clientID, nil, nil, nil, nil)
	suite.Require().Error(err)
}

func (suite *SoloMachineTestSuite) TestLatestHeight() {
	var clientID string

	testCases := []struct {
		name      string
		malleate  func()
		expHeight clienttypes.Height
	}{
		{
			"success",
			func() {},
			// Default as returned by solomachine.ClientState()
			clienttypes.NewHeight(0, 1),
		},
		{
			"failure: cannot find client state",
			func() {
				clientID = unusedSmClientID
			},
			clienttypes.ZeroHeight(),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			clientID = suite.solomachine.ClientID
			clientState := suite.solomachine.ClientState()

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, clientState)

			tc.malleate()

			height := lightClientModule.LatestHeight(suite.chainA.GetContext(), clientID)

			suite.Require().Equal(tc.expHeight, height)
		})
	}
}
