package solomachine_test

import (
	"bytes"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v7/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
)

const (
	counterpartyClientIdentifier = "chainA"
	testConnectionID             = "connectionid"
	testChannelID                = "testchannelid"
	testPortID                   = "testportid"
)

func (s *SoloMachineTestSuite) TestStatus() {
	clientState := s.solomachine.ClientState()
	// solo machine discards arguments
	status := clientState.Status(s.chainA.GetContext(), nil, nil)
	s.Require().Equal(exported.Active, status)

	// freeze solo machine
	clientState.IsFrozen = true
	status = clientState.Status(s.chainA.GetContext(), nil, nil)
	s.Require().Equal(exported.Frozen, status)
}

func (s *SoloMachineTestSuite) TestClientStateValidateBasic() {
	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{s.solomachine, s.solomachineMulti} {

		testCases := []struct {
			name        string
			clientState *solomachine.ClientState
			expPass     bool
		}{
			{
				"valid client state",
				sm.ClientState(),
				true,
			},
			{
				"empty ClientState",
				&solomachine.ClientState{},
				false,
			},
			{
				"sequence is zero",
				solomachine.NewClientState(0, &solomachine.ConsensusState{sm.ConsensusState().PublicKey, sm.Diversifier, sm.Time}),
				false,
			},
			{
				"timestamp is zero",
				solomachine.NewClientState(1, &solomachine.ConsensusState{sm.ConsensusState().PublicKey, sm.Diversifier, 0}),
				false,
			},
			{
				"diversifier is blank",
				solomachine.NewClientState(1, &solomachine.ConsensusState{sm.ConsensusState().PublicKey, "  ", 1}),
				false,
			},
			{
				"pubkey is empty",
				solomachine.NewClientState(1, &solomachine.ConsensusState{nil, sm.Diversifier, sm.Time}),
				false,
			},
		}

		for _, tc := range testCases {
			tc := tc

			s.Run(tc.name, func() {
				err := tc.clientState.Validate()

				if tc.expPass {
					s.Require().NoError(err)
				} else {
					s.Require().Error(err)
				}
			})
		}
	}
}

func (s *SoloMachineTestSuite) TestInitialize() {
	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{s.solomachine, s.solomachineMulti} {
		malleatedConsensus := sm.ClientState().ConsensusState
		malleatedConsensus.Timestamp += 10

		testCases := []struct {
			name      string
			consState exported.ConsensusState
			expPass   bool
		}{
			{
				"valid consensus state",
				sm.ConsensusState(),
				true,
			},
			{
				"nil consensus state",
				nil,
				false,
			},
			{
				"invalid consensus state: Tendermint consensus state",
				&ibctm.ConsensusState{},
				false,
			},
			{
				"invalid consensus state: consensus state does not match consensus state in client",
				malleatedConsensus,
				false,
			},
		}

		for _, tc := range testCases {
			s.SetupTest()

			store := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), "solomachine")
			err := sm.ClientState().Initialize(
				s.chainA.GetContext(), s.chainA.Codec,
				store, tc.consState,
			)

			if tc.expPass {
				s.Require().NoError(err, "valid testcase: %s failed", tc.name)
				s.Require().True(store.Has(host.ClientStateKey()))
			} else {
				s.Require().Error(err, "invalid testcase: %s passed", tc.name)
				s.Require().False(store.Has(host.ClientStateKey()))
			}
		}
	}
}

func (s *SoloMachineTestSuite) TestVerifyMembership() {
	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{s.solomachine, s.solomachineMulti} {

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
			expPass  bool
		}{
			{
				"success",
				func() {},
				true,
			},
			{
				"success: client state verification",
				func() {
					clientState = sm.ClientState()
					clientStateBz, err := s.chainA.Codec.Marshal(clientState)
					s.Require().NoError(err)

					path = s.solomachine.GetClientStatePath(counterpartyClientIdentifier)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.GetHeight().GetRevisionHeight(),
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        []byte(path.String()),
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
				true,
			},
			{
				"success: consensus state verification",
				func() {
					clientState = sm.ClientState()
					consensusState := clientState.ConsensusState
					consensusStateBz, err := s.chainA.Codec.Marshal(consensusState)
					s.Require().NoError(err)

					path = sm.GetConsensusStatePath(counterpartyClientIdentifier, clienttypes.NewHeight(0, 1))
					signBytes = solomachine.SignBytes{
						Sequence:    sm.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        []byte(path.String()),
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
				true,
			},
			{
				"success: connection state verification",
				func() {
					s.coordinator.SetupConnections(testingPath)

					connectionEnd, found := s.chainA.GetSimApp().IBCKeeper.ConnectionKeeper.GetConnection(s.chainA.GetContext(), ibctesting.FirstConnectionID)
					s.Require().True(found)

					connectionEndBz, err := s.chainA.Codec.Marshal(&connectionEnd)
					s.Require().NoError(err)

					path = sm.GetConnectionStatePath(ibctesting.FirstConnectionID)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        []byte(path.String()),
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
				true,
			},
			{
				"success: channel state verification",
				func() {
					s.coordinator.SetupConnections(testingPath)
					s.coordinator.CreateMockChannels(testingPath)

					channelEnd, found := s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetChannel(s.chainA.GetContext(), ibctesting.MockPort, ibctesting.FirstChannelID)
					s.Require().True(found)

					channelEndBz, err := s.chainA.Codec.Marshal(&channelEnd)
					s.Require().NoError(err)

					path = sm.GetChannelStatePath(ibctesting.MockPort, ibctesting.FirstChannelID)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        []byte(path.String()),
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
				true,
			},
			{
				"success: next sequence recv verification",
				func() {
					s.coordinator.SetupConnections(testingPath)
					s.coordinator.CreateMockChannels(testingPath)

					nextSeqRecv, found := s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetNextSequenceRecv(s.chainA.GetContext(), ibctesting.MockPort, ibctesting.FirstChannelID)
					s.Require().True(found)

					path = sm.GetNextSequenceRecvPath(ibctesting.MockPort, ibctesting.FirstChannelID)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        []byte(path.String()),
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
				true,
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

					commitmentBz := channeltypes.CommitPacket(s.chainA.Codec, packet)
					path = sm.GetPacketCommitmentPath(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
					signBytes = solomachine.SignBytes{
						Sequence:    sm.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        []byte(path.String()),
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
				true,
			},
			{
				"success: packet acknowledgement verification",
				func() {
					path = sm.GetPacketAcknowledgementPath(ibctesting.MockPort, ibctesting.FirstChannelID, 1)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        []byte(path.String()),
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
				true,
			},
			{
				"success: packet receipt verification",
				func() {
					path = sm.GetPacketReceiptPath(ibctesting.MockPort, ibctesting.FirstChannelID, 1)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.Sequence,
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        []byte(path.String()),
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
				true,
			},
			{
				"invalid path type",
				func() {
					path = ibcmock.KeyPath{}
				},
				false,
			},
			{
				"malformed proof fails to unmarshal",
				func() {
					path = s.solomachine.GetClientStatePath(counterpartyClientIdentifier)
					proof = []byte("invalid proof")
				},
				false,
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
				false,
			},
			{
				"signature data is nil",
				func() {
					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: nil,
						Timestamp:     sm.Time,
					}

					proof, err = s.chainA.Codec.Marshal(signatureDoc)
					s.Require().NoError(err)
				},
				false,
			},
			{
				"consensus state public key is nil",
				func() {
					clientState.ConsensusState.PublicKey = nil
				},
				false,
			},
			{
				"malformed signature data fails to unmarshal",
				func() {
					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: []byte("invalid signature data"),
						Timestamp:     sm.Time,
					}

					proof, err = s.chainA.Codec.Marshal(signatureDoc)
					s.Require().NoError(err)
				},
				false,
			},
			{
				"proof is nil",
				func() {
					proof = nil
				},
				false,
			},
			{
				"proof verification failed",
				func() {
					signBytes.Data = []byte("invalid membership data value")
				},
				false,
			},
			{
				"empty path",
				func() {
					path = commitmenttypes.MerklePath{}
				},
				false,
			},
		}

		for _, tc := range testCases {
			tc := tc

			s.Run(tc.name, func() {
				s.SetupTest()
				testingPath = ibctesting.NewPath(s.chainA, s.chainB)

				clientState = sm.ClientState()

				path = commitmenttypes.NewMerklePath("ibc", "solomachine")
				signBytes = solomachine.SignBytes{
					Sequence:    sm.GetHeight().GetRevisionHeight(),
					Timestamp:   sm.Time,
					Diversifier: sm.Diversifier,
					Path:        []byte(path.String()),
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

				tc.malleate()

				var expSeq uint64
				if clientState.ConsensusState != nil {
					expSeq = clientState.Sequence + 1
				}

				err = clientState.VerifyMembership(
					s.chainA.GetContext(), s.store, s.chainA.Codec,
					clienttypes.ZeroHeight(), 0, 0, // solomachine does not check delay periods
					proof, path, signBytes.Data,
				)

				if tc.expPass {
					s.Require().NoError(err)
					s.Require().Equal(expSeq, clientState.Sequence)
					s.Require().Equal(expSeq, s.GetSequenceFromStore(), "sequence not updated in the store (%d) on valid test case %s", s.GetSequenceFromStore(), tc.name)
				} else {
					s.Require().Error(err)
				}
			})
		}
	}
}

func (s *SoloMachineTestSuite) TestSignBytesMarshalling() {
	sm := s.solomachine
	merklePath := commitmenttypes.NewMerklePath("ibc", "solomachine")
	signBytesNilData := solomachine.SignBytes{
		Sequence:    sm.GetHeight().GetRevisionHeight(),
		Timestamp:   sm.Time,
		Diversifier: sm.Diversifier,
		Path:        []byte(merklePath.String()),
		Data:        nil,
	}

	signBytesEmptyArray := solomachine.SignBytes{
		Sequence:    sm.GetHeight().GetRevisionHeight(),
		Timestamp:   sm.Time,
		Diversifier: sm.Diversifier,
		Path:        []byte(merklePath.String()),
		Data:        []byte{},
	}

	signBzNil, err := s.chainA.Codec.Marshal(&signBytesNilData)
	s.Require().NoError(err)

	signBzEmptyArray, err := s.chainA.Codec.Marshal(&signBytesEmptyArray)
	s.Require().NoError(err)

	s.Require().True(bytes.Equal(signBzNil, signBzEmptyArray))
}

func (s *SoloMachineTestSuite) TestVerifyNonMembership() {
	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{s.solomachine, s.solomachineMulti} {

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
			expPass  bool
		}{
			{
				"success",
				func() {},
				true,
			},
			{
				"success: packet receipt absence verification",
				func() {
					path = s.solomachine.GetPacketReceiptPath(ibctesting.MockPort, ibctesting.FirstChannelID, 1)
					signBytes = solomachine.SignBytes{
						Sequence:    sm.GetHeight().GetRevisionHeight(),
						Timestamp:   sm.Time,
						Diversifier: sm.Diversifier,
						Path:        []byte(path.String()),
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
				true,
			},
			{
				"invalid path type",
				func() {
					path = ibcmock.KeyPath{}
				},
				false,
			},
			{
				"malformed proof fails to unmarshal",
				func() {
					path = s.solomachine.GetClientStatePath(counterpartyClientIdentifier)
					proof = []byte("invalid proof")
				},
				false,
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
				false,
			},
			{
				"signature data is nil",
				func() {
					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: nil,
						Timestamp:     sm.Time,
					}

					proof, err = s.chainA.Codec.Marshal(signatureDoc)
					s.Require().NoError(err)
				},
				false,
			},
			{
				"consensus state public key is nil",
				func() {
					clientState.ConsensusState.PublicKey = nil
				},
				false,
			},
			{
				"malformed signature data fails to unmarshal",
				func() {
					signatureDoc := &solomachine.TimestampedSignatureData{
						SignatureData: []byte("invalid signature data"),
						Timestamp:     sm.Time,
					}

					proof, err = s.chainA.Codec.Marshal(signatureDoc)
					s.Require().NoError(err)
				},
				false,
			},
			{
				"proof is nil",
				func() {
					proof = nil
				},
				false,
			},
			{
				"proof verification failed",
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
				false,
			},
		}

		for _, tc := range testCases {
			tc := tc

			s.Run(tc.name, func() {
				clientState = sm.ClientState()

				path = commitmenttypes.NewMerklePath("ibc", "solomachine")
				signBytes = solomachine.SignBytes{
					Sequence:    sm.GetHeight().GetRevisionHeight(),
					Timestamp:   sm.Time,
					Diversifier: sm.Diversifier,
					Path:        []byte(path.String()),
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

				tc.malleate()

				var expSeq uint64
				if clientState.ConsensusState != nil {
					expSeq = clientState.Sequence + 1
				}

				err = clientState.VerifyNonMembership(
					s.chainA.GetContext(), s.store, s.chainA.Codec,
					clienttypes.ZeroHeight(), 0, 0, // solomachine does not check delay periods
					proof, path,
				)

				if tc.expPass {
					s.Require().NoError(err)
					s.Require().Equal(expSeq, clientState.Sequence)
					s.Require().Equal(expSeq, s.GetSequenceFromStore(), "sequence not updated in the store (%d) on valid test case %s", s.GetSequenceFromStore(), tc.name)
				} else {
					s.Require().Error(err)
				}
			})
		}
	}
}

func (s *SoloMachineTestSuite) TestGetTimestampAtHeight() {
	tmPath := ibctesting.NewPath(s.chainA, s.chainB)
	s.coordinator.SetupClients(tmPath)
	// Single setup for all test cases.
	s.SetupTest()

	testCases := []struct {
		name        string
		clientState *solomachine.ClientState
		height      exported.Height
		expValue    uint64
		expPass     bool
	}{
		{
			name:        "get timestamp at height exists",
			clientState: s.solomachine.ClientState(),
			height:      s.solomachine.ClientState().GetLatestHeight(),
			expValue:    s.solomachine.ClientState().ConsensusState.Timestamp,
			expPass:     true,
		},
	}

	for i, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			ctx := s.chainA.GetContext()

			ts, err := tc.clientState.GetTimestampAtHeight(
				ctx, s.store, s.chainA.Codec, tc.height,
			)

			s.Require().Equal(tc.expValue, ts)

			if tc.expPass {
				s.Require().NoError(err, "valid test case %d failed: %s", i, tc.name)
			} else {
				s.Require().Error(err, "invalid test case %d passed: %s", i, tc.name)
			}
		})
	}
}
