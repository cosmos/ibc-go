package solomachine_test

import (
	"errors"
	"fmt"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types/v2"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v10/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	ibcmock "github.com/cosmos/ibc-go/v10/testing/mock"
)

const (
	unusedSmClientID = "06-solomachine-999"
	wasmClientID     = "08-wasm-0"
)

func (s *SoloMachineTestSuite) TestStatus() {
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
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), clientID, clientState)
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
		s.Run(tc.name, func() {
			clientID = s.solomachine.ClientID

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
			s.Require().NoError(err)

			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), clientID, s.solomachine.ClientState())

			tc.malleate()

			status := lightClientModule.Status(s.chainA.GetContext(), clientID)
			s.Require().Equal(tc.expStatus, status)
		})
	}
}

func (s *SoloMachineTestSuite) TestGetTimestampAtHeight() {
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
			s.solomachine.ClientState().ConsensusState.Timestamp,
			nil,
		},
		{
			"success: modified height",
			func() {
				height = clienttypes.ZeroHeight()
			},
			// Timestamp should be the same.
			s.solomachine.ClientState().ConsensusState.Timestamp,
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
		s.Run(tc.name, func() {
			clientID = s.solomachine.ClientID
			clientState := s.solomachine.ClientState()
			height = clienttypes.NewHeight(0, s.solomachine.ClientState().Sequence)

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
			s.Require().NoError(err)

			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), clientID, clientState)

			tc.malleate()

			ts, err := lightClientModule.TimestampAtHeight(s.chainA.GetContext(), clientID, height)

			s.Require().Equal(tc.expValue, ts)
			s.Require().ErrorIs(err, tc.expErr)
		})
	}
}

func (s *SoloMachineTestSuite) TestInitialize() {
	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{s.solomachine, s.solomachineMulti} {
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
				errors.New("proto: wrong wireType = 0 for field TypeUrl"),
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
				errors.New("proto: wrong wireType = 2 for field IsFrozen"),
			},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				s.SetupTest()
				clientID := sm.ClientID

				clientStateBz := s.chainA.Codec.MustMarshal(tc.clientState)
				consStateBz := s.chainA.Codec.MustMarshal(tc.consState)

				lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
				s.Require().NoError(err)

				err = lightClientModule.Initialize(s.chainA.GetContext(), clientID, clientStateBz, consStateBz)
				store := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), clientID)

				if tc.expErr == nil {
					s.Require().NoError(err)
					s.Require().True(store.Has(host.ClientStateKey()))
				} else {
					s.Require().ErrorContains(err, tc.expErr.Error())
					s.Require().False(store.Has(host.ClientStateKey()))
				}
			})
		}
	}
}

func (s *SoloMachineTestSuite) TestVerifyMembership() {
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
	for _, sm := range []*ibctesting.Solomachine{s.solomachine, s.solomachineMulti} {
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
					clientStateBz, err := s.chainA.Codec.MarshalInterface(clientState)
					s.Require().NoError(err)

					path = sm.GetClientStatePath(counterpartyClientIdentifier)
					merklePath, ok := path.(commitmenttypesv2.MerklePath)
					s.Require().True(ok)
					key, err := merklePath.GetKey(1) // in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
					s.Require().NoError(err)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.GetHeight().GetRevisionHeight(),
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        key,
						Data:        clientStateBz,
					}

					signBz, err := s.chainA.Codec.Marshal(&signBytes)
					s.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)

					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: sig,
						Timestamp:     sm.Time,
					}

					proof, err = s.chainA.Codec.Marshal(signatureDoc)
					s.Require().NoError(err)
				},
				nil,
			},
			{
				"success: consensus state verification",
				func() {
					clientState = sm.ClientState()
					consensusState := clientState.ConsensusState
					consensusStateBz, err := s.chainA.Codec.MarshalInterface(consensusState)
					s.Require().NoError(err)

					path = sm.GetConsensusStatePath(counterpartyClientIdentifier, clienttypes.NewHeight(0, 1))
					merklePath, ok := path.(commitmenttypesv2.MerklePath)
					s.Require().True(ok)
					key, err := merklePath.GetKey(1) // in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
					s.Require().NoError(err)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        key,
						Data:        consensusStateBz,
					}

					signBz, err := s.chainA.Codec.Marshal(&signBytes)
					s.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)

					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: sig,
						Timestamp:     sm.Time,
					}

					proof, err = s.chainA.Codec.Marshal(signatureDoc)
					s.Require().NoError(err)
				},
				nil,
			},
			{
				"success: connection state verification",
				func() {
					testingPath.SetupConnections()

					connectionEnd, found := s.chainA.GetSimApp().IBCKeeper.ConnectionKeeper.GetConnection(s.chainA.GetContext(), ibctesting.FirstConnectionID)
					s.Require().True(found)

					connectionEndBz, err := s.chainA.Codec.Marshal(&connectionEnd)
					s.Require().NoError(err)

					path = sm.GetConnectionStatePath(ibctesting.FirstConnectionID)
					merklePath, ok := path.(commitmenttypesv2.MerklePath)
					s.Require().True(ok)
					key, err := merklePath.GetKey(1) // in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
					s.Require().NoError(err)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        key,
						Data:        connectionEndBz,
					}

					signBz, err := s.chainA.Codec.Marshal(&signBytes)
					s.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)

					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: sig,
						Timestamp:     sm.Time,
					}

					proof, err = s.chainA.Codec.Marshal(signatureDoc)
					s.Require().NoError(err)
				},
				nil,
			},
			{
				"success: channel state verification",
				func() {
					testingPath.SetupConnections()
					s.coordinator.CreateMockChannels(testingPath)

					channelEnd, found := s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetChannel(s.chainA.GetContext(), ibctesting.MockPort, testingPath.EndpointA.ChannelID)
					s.Require().True(found)

					channelEndBz, err := s.chainA.Codec.Marshal(&channelEnd)
					s.Require().NoError(err)

					path = sm.GetChannelStatePath(ibctesting.MockPort, ibctesting.FirstChannelID)
					merklePath, ok := path.(commitmenttypesv2.MerklePath)
					s.Require().True(ok)
					key, err := merklePath.GetKey(1) // in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
					s.Require().NoError(err)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        key,
						Data:        channelEndBz,
					}

					signBz, err := s.chainA.Codec.Marshal(&signBytes)
					s.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)

					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: sig,
						Timestamp:     sm.Time,
					}

					proof, err = s.chainA.Codec.Marshal(signatureDoc)
					s.Require().NoError(err)
				},
				nil,
			},
			{
				"success: next sequence recv verification",
				func() {
					testingPath.SetupConnections()
					s.coordinator.CreateMockChannels(testingPath)

					nextSeqRecv, found := s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetNextSequenceRecv(s.chainA.GetContext(), ibctesting.MockPort, testingPath.EndpointA.ChannelID)
					s.Require().True(found)

					path = sm.GetNextSequenceRecvPath(ibctesting.MockPort, ibctesting.FirstChannelID)
					merklePath, ok := path.(commitmenttypesv2.MerklePath)
					s.Require().True(ok)
					key, err := merklePath.GetKey(1) // in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
					s.Require().NoError(err)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        key,
						Data:        sdk.Uint64ToBigEndian(nextSeqRecv),
					}

					signBz, err := s.chainA.Codec.Marshal(&signBytes)
					s.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)

					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: sig,
						Timestamp:     sm.Time,
					}

					proof, err = s.chainA.Codec.Marshal(signatureDoc)
					s.Require().NoError(err)
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

					commitmentBz := channeltypes.CommitPacket(packet)
					path = sm.GetPacketCommitmentPath(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
					merklePath, ok := path.(commitmenttypesv2.MerklePath)
					s.Require().True(ok)
					key, err := merklePath.GetKey(1) // in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
					s.Require().NoError(err)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        key,
						Data:        commitmentBz,
					}

					signBz, err := s.chainA.Codec.Marshal(&signBytes)
					s.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)

					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: sig,
						Timestamp:     sm.Time,
					}

					proof, err = s.chainA.Codec.Marshal(signatureDoc)
					s.Require().NoError(err)
				},
				nil,
			},
			{
				"success: packet acknowledgement verification",
				func() {
					path = sm.GetPacketAcknowledgementPath(ibctesting.MockPort, ibctesting.FirstChannelID, 1)
					merklePath, ok := path.(commitmenttypesv2.MerklePath)
					s.Require().True(ok)
					key, err := merklePath.GetKey(1) // index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
					s.Require().NoError(err)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        key,
						Data:        ibctesting.MockAcknowledgement,
					}

					signBz, err := s.chainA.Codec.Marshal(&signBytes)
					s.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)

					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: sig,
						Timestamp:     sm.Time,
					}

					proof, err = s.chainA.Codec.Marshal(signatureDoc)
					s.Require().NoError(err)
				},
				nil,
			},
			{
				"success: packet receipt verification",
				func() {
					path = sm.GetPacketReceiptPath(ibctesting.MockPort, ibctesting.FirstChannelID, 1)
					merklePath, ok := path.(commitmenttypesv2.MerklePath)
					s.Require().True(ok)
					key, err := merklePath.GetKey(1) // in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
					s.Require().NoError(err)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        key,
						Data:        []byte{byte(1)}, // packet receipt is stored as a single byte
					}

					signBz, err := s.chainA.Codec.Marshal(&signBytes)
					s.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)

					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: sig,
						Timestamp:     sm.Time,
					}

					proof, err = s.chainA.Codec.Marshal(signatureDoc)
					s.Require().NoError(err)
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
				errors.New("failed to unmarshal proof into type"),
			},
			{
				"failure: consensus state timestamp is greater than signature",
				func() {
					consensusState := &solomachine.ConsensusState{
						Timestamp: sm.Time + 1,
						PublicKey: sm.ConsensusState().PublicKey,
					}

					clientState = solomachine.NewClientState(sm.Sequence, consensusState)
					s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), clientID, clientState)
				},
				fmt.Errorf("the consensus state timestamp is greater than the signature timestamp (11 >= 10): %w", solomachine.ErrInvalidProof),
			},
			{
				"failure: signature data is nil",
				func() {
					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: nil,
						Timestamp:     sm.Time,
					}

					proof, err = s.chainA.Codec.Marshal(signatureDoc)
					s.Require().NoError(err)
				},
				fmt.Errorf("signature data cannot be empty: %w", solomachine.ErrInvalidProof),
			},
			{
				"failure: consensus state public key is nil",
				func() {
					clientState.ConsensusState.PublicKey = nil
					s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), clientID, clientState)
				},
				fmt.Errorf("consensus state PublicKey cannot be nil: %w", clienttypes.ErrInvalidConsensus),
			},
			{
				"failure: malformed signature data fails to unmarshal",
				func() {
					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: []byte("invalid signature data"),
						Timestamp:     sm.Time,
					}

					proof, err = s.chainA.Codec.Marshal(signatureDoc)
					s.Require().NoError(err)
				},
				errors.New("failed to unmarshal proof into type"),
			},
			{
				"failure: proof is nil",
				func() {
					proof = nil
				},
				fmt.Errorf("proof cannot be empty: %w", solomachine.ErrInvalidProof),
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
					path = commitmenttypesv2.MerklePath{}
				},
				fmt.Errorf("path must be of length 2: []: %w", host.ErrInvalidPath),
			},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				s.SetupTest()
				testingPath = ibctesting.NewPath(s.chainA, s.chainB)

				clientID = sm.ClientID
				clientState = sm.ClientState()

				path = commitmenttypesv2.NewMerklePath([]byte("ibc"), []byte("solomachine"))
				merklePath, ok := path.(commitmenttypesv2.MerklePath)
				s.Require().True(ok)
				key, err := merklePath.GetKey(1) // in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
				s.Require().NoError(err)
				signBytes = solomachine.SignBytes{
					Sequence:    sm.GetHeight().GetRevisionHeight(),
					Timestamp:   sm.Time,
					Diversifier: sm.Diversifier,
					Path:        key,
					Data:        []byte("solomachine"),
				}

				signBz, err := s.chainA.Codec.Marshal(&signBytes)
				s.Require().NoError(err)

				sig := sm.GenerateSignature(signBz)

				signatureDoc := &solomachine.TimestampedSignatureData{
					SignatureData: sig,
					Timestamp:     sm.Time,
				}

				proof, err = s.chainA.Codec.Marshal(signatureDoc)
				s.Require().NoError(err)

				lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
				s.Require().NoError(err)

				// Set the client state in the store for light client call to find.
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), clientID, clientState)

				tc.malleate()

				var expSeq uint64
				if clientState.ConsensusState != nil {
					expSeq = clientState.Sequence + 1
				}

				// Verify the membership proof
				err = lightClientModule.VerifyMembership(
					s.chainA.GetContext(), clientID, clienttypes.ZeroHeight(),
					0, 0, proof, path, signBytes.Data,
				)

				if tc.expErr == nil {
					// Grab fresh client state after updates.
					cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), clientID)
					s.Require().True(found)
					clientState, ok = cs.(*solomachine.ClientState)
					s.Require().True(ok)

					s.Require().NoError(err)
					// clientState.Sequence is the most recent view of state.
					s.Require().Equal(expSeq, clientState.Sequence)
				} else {
					s.Require().Error(err)
					s.Require().ErrorContains(err, tc.expErr.Error())
				}
			})
		}
	}
}

func (s *SoloMachineTestSuite) TestVerifyNonMembership() {
	var (
		clientState *solomachine.ClientState
		path        exported.Path
		proof       []byte
		signBytes   solomachine.SignBytes
		err         error
		clientID    string
	)

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{s.solomachine, s.solomachineMulti} {
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
					merklePath, ok := path.(commitmenttypesv2.MerklePath)
					s.Require().True(ok)
					key, err := merklePath.GetKey(1) // in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
					s.Require().NoError(err)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.GetHeight().GetRevisionHeight(),
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        key,
						Data:        nil,
					}

					signBz, err := s.chainA.Codec.Marshal(&signBytes)
					s.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)

					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: sig,
						Timestamp:     sm.Time,
					}

					proof, err = s.chainA.Codec.Marshal(signatureDoc)
					s.Require().NoError(err)
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
				errors.New("failed to unmarshal proof into type"),
			},
			{
				"failure: consensus state timestamp is greater than signature",
				func() {
					consensusState := &solomachine.ConsensusState{
						Timestamp: sm.Time + 1,
						PublicKey: sm.ConsensusState().PublicKey,
					}

					clientState = solomachine.NewClientState(sm.Sequence, consensusState)
					s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), clientID, clientState)
				},
				fmt.Errorf("the consensus state timestamp is greater than the signature timestamp (11 >= 10): %w", solomachine.ErrInvalidProof),
			},
			{
				"failure: signature data is nil",
				func() {
					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: nil,
						Timestamp:     sm.Time,
					}

					proof, err = s.chainA.Codec.Marshal(signatureDoc)
					s.Require().NoError(err)
				},
				fmt.Errorf("signature data cannot be empty: %w", solomachine.ErrInvalidProof),
			},
			{
				"failure: consensus state public key is nil",
				func() {
					clientState.ConsensusState.PublicKey = nil
					s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), clientID, clientState)
				},
				fmt.Errorf("consensus state PublicKey cannot be nil: %w", clienttypes.ErrInvalidConsensus),
			},
			{
				"failure: malformed signature data fails to unmarshal",
				func() {
					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: []byte("invalid signature data"),
						Timestamp:     sm.Time,
					}

					proof, err = s.chainA.Codec.Marshal(signatureDoc)
					s.Require().NoError(err)
				},
				errors.New("failed to unmarshal proof into type"),
			},
			{
				"failure: proof is nil",
				func() {
					proof = nil
				},
				fmt.Errorf("proof cannot be empty: %w", solomachine.ErrInvalidProof),
			},
			{
				"failure: proof verification failed",
				func() {
					signBytes.Data = []byte("invalid non-membership data value")

					signBz, err := s.chainA.Codec.Marshal(&signBytes)
					s.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)

					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: sig,
						Timestamp:     sm.Time,
					}

					proof, err = s.chainA.Codec.Marshal(signatureDoc)
					s.Require().NoError(err)
				},
				solomachine.ErrSignatureVerificationFailed,
			},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				s.SetupTest()

				clientState = sm.ClientState()
				clientID = sm.ClientID

				path = commitmenttypesv2.NewMerklePath([]byte("ibc"), []byte("solomachine"))
				merklePath, ok := path.(commitmenttypesv2.MerklePath)
				s.Require().True(ok)
				key, err := merklePath.GetKey(1) // in a multistore context: index 0 is the key for the IBC store in the multistore, index 1 is the key in the IBC store
				s.Require().NoError(err)
				signBytes = solomachine.SignBytes{
					Sequence:    sm.GetHeight().GetRevisionHeight(),
					Timestamp:   sm.Time,
					Diversifier: sm.Diversifier,
					Path:        key,
					Data:        nil,
				}

				signBz, err := s.chainA.Codec.Marshal(&signBytes)
				s.Require().NoError(err)

				sig := sm.GenerateSignature(signBz)

				signatureDoc := &solomachine.TimestampedSignatureData{
					SignatureData: sig,
					Timestamp:     sm.Time,
				}

				proof, err = s.chainA.Codec.Marshal(signatureDoc)
				s.Require().NoError(err)

				lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
				s.Require().NoError(err)

				// Set the client state in the store for light client call to find.
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), clientID, clientState)

				tc.malleate()

				var expSeq uint64
				if clientState.ConsensusState != nil {
					expSeq = clientState.Sequence + 1
				}

				// Verify the membership proof
				err = lightClientModule.VerifyNonMembership(
					s.chainA.GetContext(), clientID, clienttypes.ZeroHeight(),
					0, 0, proof, path,
				)

				if tc.expErr == nil {
					// Grab fresh client state after updates.
					cs, found := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), clientID)
					s.Require().True(found)
					clientState, ok = cs.(*solomachine.ClientState)
					s.Require().True(ok)

					s.Require().NoError(err)
					s.Require().Equal(expSeq, clientState.Sequence)
				} else {
					s.Require().Error(err)
					s.Require().ErrorContains(err, tc.expErr.Error())
				}
			})
		}
	}
}

func (s *SoloMachineTestSuite) TestRecoverClient() {
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
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			ctx := s.chainA.GetContext()

			subjectClientID = s.chainA.App.GetIBCKeeper().ClientKeeper.GenerateClientIdentifier(ctx, exported.Solomachine)
			subject := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, substituteClientID, "testing", 1)
			subjectClientState = subject.ClientState()

			substituteClientID = s.chainA.App.GetIBCKeeper().ClientKeeper.GenerateClientIdentifier(ctx, exported.Solomachine)
			substitute := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, substituteClientID, "testing", 1)
			substitute.Sequence++ // increase sequence so that latest height of substitute is > than subject's latest height
			substituteClientState = substitute.ClientState()

			clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, substituteClientID)
			clientStore.Get(host.ClientStateKey())
			bz := clienttypes.MustMarshalClientState(s.chainA.Codec, substituteClientState)
			clientStore.Set(host.ClientStateKey(), bz)

			subjectClientState.IsFrozen = true
			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(ctx, subjectClientID, subjectClientState)

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), subjectClientID)
			s.Require().NoError(err)

			tc.malleate()

			err = lightClientModule.RecoverClient(ctx, subjectClientID, substituteClientID)

			if tc.expErr == nil {
				s.Require().NoError(err)

				// assert that status of subject client is now Active
				clientStore = s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, subjectClientID)
				bz = clientStore.Get(host.ClientStateKey())
				smClientState, ok := clienttypes.MustUnmarshalClientState(s.chainA.Codec, bz).(*solomachine.ClientState)
				s.Require().True(ok)

				s.Require().Equal(substituteClientState.ConsensusState, smClientState.ConsensusState)
				s.Require().Equal(substituteClientState.Sequence, smClientState.Sequence)
				s.Require().Equal(exported.Active, lightClientModule.Status(ctx, subjectClientID))
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (s *SoloMachineTestSuite) TestUpdateState() {
	var (
		clientState *solomachine.ClientState
		clientMsg   exported.ClientMessage
		clientID    string
	)

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{s.solomachine, s.solomachineMulti} {
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
				"invalid type misbehaviour no-ops",
				func() {
					clientState = sm.ClientState()
					clientMsg = sm.CreateMisbehaviour()
					s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), clientID, clientState)
				},
				nil,
			},
			{
				"failure: cannot find client state",
				func() {
					clientID = unusedSmClientID
				},
				fmt.Errorf("%s: %w", unusedSmClientID, clienttypes.ErrClientNotFound),
			},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				s.SetupTest()

				clientID = sm.ClientID
				clientState = sm.ClientState()
				clientMsg = sm.CreateHeader(sm.Diversifier)

				lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
				s.Require().NoError(err)

				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), clientID, clientState)

				tc.malleate() // setup test

				store := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), clientID)

				var consensusHeights []exported.Height
				updateStateFunc := func() {
					consensusHeights = lightClientModule.UpdateState(s.chainA.GetContext(), clientID, clientMsg)
				}

				if tc.expPanic == nil {
					updateStateFunc()

					clientStateBz := store.Get(host.ClientStateKey())
					s.Require().NotEmpty(clientStateBz)

					newClientState := clienttypes.MustUnmarshalClientState(s.chainA.Codec, clientStateBz)

					if len(consensusHeights) == 0 {
						s.Require().Equal(clientState, newClientState)
						return
					}

					s.Require().Len(consensusHeights, 1)
					s.Require().Equal(uint64(0), consensusHeights[0].GetRevisionNumber())
					s.Require().Equal(newClientState.(*solomachine.ClientState).Sequence, consensusHeights[0].GetRevisionHeight())

					s.Require().False(newClientState.(*solomachine.ClientState).IsFrozen)
					s.Require().Equal(clientMsg.(*solomachine.Header).NewPublicKey, newClientState.(*solomachine.ClientState).ConsensusState.PublicKey)
					s.Require().Equal(clientMsg.(*solomachine.Header).NewDiversifier, newClientState.(*solomachine.ClientState).ConsensusState.Diversifier)
					s.Require().Equal(clientMsg.(*solomachine.Header).Timestamp, newClientState.(*solomachine.ClientState).ConsensusState.Timestamp)
				} else {
					s.Require().PanicsWithError(tc.expPanic.Error(), updateStateFunc)
				}
			})
		}
	}
}

func (s *SoloMachineTestSuite) TestCheckForMisbehaviour() {
	var (
		clientMsg exported.ClientMessage
		clientID  string
	)

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{s.solomachine, s.solomachineMulti} {
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
				fmt.Errorf("%s: %w", unusedSmClientID, clienttypes.ErrClientNotFound),
			},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				s.SetupTest()

				clientID = sm.ClientID

				lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
				s.Require().NoError(err)

				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), clientID, sm.ClientState())

				tc.malleate()

				var foundMisbehaviour bool
				foundMisbehaviourFunc := func() {
					foundMisbehaviour = lightClientModule.CheckForMisbehaviour(s.chainA.GetContext(), clientID, clientMsg)
				}

				if tc.expPanic == nil {
					foundMisbehaviourFunc()

					s.Require().Equal(tc.foundMisbehaviour, foundMisbehaviour)
				} else {
					s.Require().PanicsWithError(tc.expPanic.Error(), foundMisbehaviourFunc)
					s.Require().False(foundMisbehaviour)
				}
			})
		}
	}
}

func (s *SoloMachineTestSuite) TestUpdateStateOnMisbehaviour() {
	var clientID string

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{s.solomachine, s.solomachineMulti} {
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
				fmt.Errorf("%s: %w", unusedSmClientID, clienttypes.ErrClientNotFound),
			},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				s.SetupTest()
				clientID = sm.ClientID

				lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
				s.Require().NoError(err)

				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), clientID, sm.ClientState())

				tc.malleate()

				updateOnMisbehaviourFunc := func() {
					lightClientModule.UpdateStateOnMisbehaviour(s.chainA.GetContext(), clientID, nil)
				}

				if tc.expPanic == nil {
					updateOnMisbehaviourFunc()

					store := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), clientID)

					clientStateBz := store.Get(host.ClientStateKey())
					s.Require().NotEmpty(clientStateBz)

					newClientState := clienttypes.MustUnmarshalClientState(s.chainA.Codec, clientStateBz)

					s.Require().True(newClientState.(*solomachine.ClientState).IsFrozen)
				} else {
					s.Require().PanicsWithError(tc.expPanic.Error(), updateOnMisbehaviourFunc)
				}
			})
		}
	}
}

func (s *SoloMachineTestSuite) TestVerifyClientMessageHeader() {
	var (
		clientID  string
		clientMsg exported.ClientMessage
	)

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{s.solomachine, s.solomachineMulti} {
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
					h.Signature = s.GetInvalidProof()
					clientMsg = h
				}, errors.New("proto: wrong wireType = 0 for field Multi"),
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
					s.Require().NoError(err)

					data := &solomachine.HeaderData{
						NewPubKey:      publicKey,
						NewDiversifier: h.NewDiversifier,
					}

					dataBz, err := s.chainA.Codec.Marshal(data)
					s.Require().NoError(err)

					// generate invalid signature
					signBytes := &solomachine.SignBytes{
						Sequence:    cs.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        []byte("invalid signature data"),
						Data:        dataBz,
					}

					signBz, err := s.chainA.Codec.Marshal(signBytes)
					s.Require().NoError(err)

					sig := sm.GenerateSignature(signBz)
					s.Require().NoError(err)
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
				fmt.Errorf("%s: %w", unusedSmClientID, clienttypes.ErrClientNotFound),
			},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				s.SetupTest()
				clientID = sm.ClientID

				lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
				s.Require().NoError(err)

				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), clientID, sm.ClientState())

				tc.malleate()

				err = lightClientModule.VerifyClientMessage(s.chainA.GetContext(), clientID, clientMsg)

				if tc.expErr == nil {
					s.Require().NoError(err)
				} else {
					s.Require().ErrorContains(err, tc.expErr.Error())
				}
			})
		}
	}
}

func (s *SoloMachineTestSuite) TestVerifyClientMessageMisbehaviour() {
	var (
		clientMsg   exported.ClientMessage
		clientState *solomachine.ClientState
		clientID    string
	)

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{s.solomachine, s.solomachineMulti} {
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
					s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), clientID, clientState)
				},
				clienttypes.ErrInvalidConsensus,
			},
			{
				"failure: invalid SignatureOne SignatureData",
				func() {
					m := sm.CreateMisbehaviour()

					m.SignatureOne.Signature = s.GetInvalidProof()
					clientMsg = m
				}, errors.New("proto: wrong wireType = 0 for field Multi"),
			},
			{
				"failure: invalid SignatureTwo SignatureData",
				func() {
					m := sm.CreateMisbehaviour()

					m.SignatureTwo.Signature = s.GetInvalidProof()
					clientMsg = m
				}, errors.New("proto: wrong wireType = 0 for field Multi"),
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

					data, err := s.chainA.Codec.Marshal(signBytes)
					s.Require().NoError(err)

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

					data, err := s.chainA.Codec.Marshal(signBytes)
					s.Require().NoError(err)

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

					data, err := s.chainA.Codec.Marshal(signBytes)
					s.Require().NoError(err)

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
					data, err = s.chainA.Codec.Marshal(signBytes)
					s.Require().NoError(err)

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
				fmt.Errorf("%s: %w", unusedSmClientID, clienttypes.ErrClientNotFound),
			},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				s.SetupTest()
				clientID = sm.ClientID

				lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
				s.Require().NoError(err)

				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), clientID, sm.ClientState())

				tc.malleate()

				err = lightClientModule.VerifyClientMessage(s.chainA.GetContext(), clientID, clientMsg)

				if tc.expErr == nil {
					s.Require().NoError(err)
				} else {
					s.Require().ErrorContains(err, tc.expErr.Error())
				}
			})
		}
	}
}

func (s *SoloMachineTestSuite) TestVerifyUpgradeAndUpdateState() {
	clientID := s.solomachine.ClientID

	lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
	s.Require().NoError(err)

	err = lightClientModule.VerifyUpgradeAndUpdateState(s.chainA.GetContext(), clientID, nil, nil, nil, nil)
	s.Require().Error(err)
}

func (s *SoloMachineTestSuite) TestLatestHeight() {
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
		s.Run(tc.name, func() {
			s.SetupTest()
			clientID = s.solomachine.ClientID
			clientState := s.solomachine.ClientState()

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
			s.Require().NoError(err)

			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), clientID, clientState)

			tc.malleate()

			height := lightClientModule.LatestHeight(s.chainA.GetContext(), clientID)

			s.Require().Equal(tc.expHeight, height)
		})
	}
}
