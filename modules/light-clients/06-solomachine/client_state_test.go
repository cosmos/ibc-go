package solomachine_test

import (
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v3/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v3/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v3/modules/light-clients/06-solomachine"
	ibctmtypes "github.com/cosmos/ibc-go/v3/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

const (
	counterpartyClientIdentifier = "chainA"
	testConnectionID             = "connectionid"
	testChannelID                = "testchannelid"
	testPortID                   = "testportid"
)

var (
	prefix = &commitmenttypes.MerklePrefix{
		KeyPrefix: []byte("ibc"),
	}
	consensusHeight = clienttypes.ZeroHeight()
)

func (suite *SoloMachineTestSuite) TestStatus() {
	clientState := suite.solomachine.ClientState()
	// solo machine discards arguments
	status := clientState.Status(suite.chainA.GetContext(), nil, nil)
	suite.Require().Equal(exported.Active, status)

	// freeze solo machine
	clientState.IsFrozen = true
	status = clientState.Status(suite.chainA.GetContext(), nil, nil)
	suite.Require().Equal(exported.Frozen, status)
}

func (suite *SoloMachineTestSuite) TestClientStateValidateBasic() {
	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {

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
				solomachine.NewClientState(0, &solomachine.ConsensusState{sm.ConsensusState().PublicKey, sm.Diversifier, sm.Time}, false),
				false,
			},
			{
				"timestamp is zero",
				solomachine.NewClientState(1, &solomachine.ConsensusState{sm.ConsensusState().PublicKey, sm.Diversifier, 0}, false),
				false,
			},
			{
				"diversifier is blank",
				solomachine.NewClientState(1, &solomachine.ConsensusState{sm.ConsensusState().PublicKey, "  ", 1}, false),
				false,
			},
			{
				"pubkey is empty",
				solomachine.NewClientState(1, &solomachine.ConsensusState{nil, sm.Diversifier, sm.Time}, false),
				false,
			},
		}

		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {

				err := tc.clientState.Validate()

				if tc.expPass {
					suite.Require().NoError(err)
				} else {
					suite.Require().Error(err)
				}
			})
		}
	}
}

func (suite *SoloMachineTestSuite) TestInitialize() {
	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {
		malleatedConsensus := sm.ClientState().ConsensusState
		malleatedConsensus.Timestamp = malleatedConsensus.Timestamp + 10

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
				&ibctmtypes.ConsensusState{},
				false,
			},
			{
				"invalid consensus state: consensus state does not match consensus state in client",
				malleatedConsensus,
				false,
			},
		}

		for _, tc := range testCases {
			err := sm.ClientState().Initialize(
				suite.chainA.GetContext(), suite.chainA.Codec,
				suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), "solomachine"),
				tc.consState,
			)

			if tc.expPass {
				suite.Require().NoError(err, "valid testcase: %s failed", tc.name)
			} else {
				suite.Require().Error(err, "invalid testcase: %s passed", tc.name)
			}
		}
	}
}

func (suite *SoloMachineTestSuite) TestVerifyClientState() {
	// create client for tendermint so we can use client state for verification
	tmPath := ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupClients(tmPath)
	clientState := suite.chainA.GetClientState(tmPath.EndpointA.ClientID)
	path := suite.solomachine.GetClientStatePath(counterpartyClientIdentifier)

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {

		value, err := solomachine.ClientStateSignBytes(suite.chainA.Codec, sm.Sequence, sm.Time, sm.Diversifier, path, clientState)
		suite.Require().NoError(err)

		sig := sm.GenerateSignature(value)

		signatureDoc := &solomachine.TimestampedSignatureData{
			SignatureData: sig,
			Timestamp:     sm.Time,
		}

		proof, err := suite.chainA.Codec.Marshal(signatureDoc)
		suite.Require().NoError(err)

		testCases := []struct {
			name        string
			clientState *solomachine.ClientState
			prefix      exported.Prefix
			proof       []byte
			expPass     bool
		}{
			{
				"successful verification",
				sm.ClientState(),
				prefix,
				proof,
				true,
			},
			{
				"ApplyPrefix failed",
				sm.ClientState(),
				nil,
				proof,
				false,
			},
			{
				"consensus state in client state is nil",
				solomachine.NewClientState(1, nil, false),
				prefix,
				proof,
				false,
			},
			{
				"client state latest height is less than sequence",
				solomachine.NewClientState(sm.Sequence-1,
					&solomachine.ConsensusState{
						Timestamp: sm.Time,
						PublicKey: sm.ConsensusState().PublicKey,
					}, false),
				prefix,
				proof,
				false,
			},
			{
				"consensus state timestamp is greater than signature",
				solomachine.NewClientState(sm.Sequence,
					&solomachine.ConsensusState{
						Timestamp: sm.Time + 1,
						PublicKey: sm.ConsensusState().PublicKey,
					}, false),
				prefix,
				proof,
				false,
			},

			{
				"proof is nil",
				sm.ClientState(),
				prefix,
				nil,
				false,
			},
			{
				"proof verification failed",
				sm.ClientState(),
				prefix,
				suite.GetInvalidProof(),
				false,
			},
		}

		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {

				var expSeq uint64
				if tc.clientState.ConsensusState != nil {
					expSeq = tc.clientState.Sequence + 1
				}

				// NOTE: to replicate the ordering of connection handshake, we must decrement proof height by 1
				height := clienttypes.NewHeight(sm.GetHeight().GetRevisionNumber(), sm.GetHeight().GetRevisionHeight()-1)

				err := tc.clientState.VerifyClientState(
					suite.store, suite.chainA.Codec, height, tc.prefix, counterpartyClientIdentifier, tc.proof, clientState,
				)

				if tc.expPass {
					suite.Require().NoError(err)
					suite.Require().Equal(expSeq, tc.clientState.Sequence)
					suite.Require().Equal(expSeq, suite.GetSequenceFromStore(), "sequence not updated in the store (%d) on valid test case %s", suite.GetSequenceFromStore(), tc.name)
				} else {
					suite.Require().Error(err)
				}
			})
		}
	}
}

func (suite *SoloMachineTestSuite) TestVerifyClientConsensusState() {
	// create client for tendermint so we can use consensus state for verification
	tmPath := ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupClients(tmPath)
	clientState := suite.chainA.GetClientState(tmPath.EndpointA.ClientID)
	consensusState, found := suite.chainA.GetConsensusState(tmPath.EndpointA.ClientID, clientState.GetLatestHeight())
	suite.Require().True(found)

	path := suite.solomachine.GetConsensusStatePath(counterpartyClientIdentifier, consensusHeight)

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {

		value, err := solomachine.ConsensusStateSignBytes(suite.chainA.Codec, sm.Sequence, sm.Time, sm.Diversifier, path, consensusState)
		suite.Require().NoError(err)

		sig := sm.GenerateSignature(value)
		signatureDoc := &solomachine.TimestampedSignatureData{
			SignatureData: sig,
			Timestamp:     sm.Time,
		}

		proof, err := suite.chainA.Codec.Marshal(signatureDoc)
		suite.Require().NoError(err)

		testCases := []struct {
			name        string
			clientState *solomachine.ClientState
			prefix      exported.Prefix
			proof       []byte
			expPass     bool
		}{
			{
				"successful verification",
				sm.ClientState(),
				prefix,
				proof,
				true,
			},
			{
				"ApplyPrefix failed",
				sm.ClientState(),
				nil,
				proof,
				false,
			},
			{
				"consensus state in client state is nil",
				solomachine.NewClientState(1, nil, false),
				prefix,
				proof,
				false,
			},
			{
				"client state latest height is less than sequence",
				solomachine.NewClientState(sm.Sequence-1,
					&solomachine.ConsensusState{
						Timestamp: sm.Time,
						PublicKey: sm.ConsensusState().PublicKey,
					}, false),
				prefix,
				proof,
				false,
			},
			{
				"consensus state timestamp is greater than signature",
				solomachine.NewClientState(sm.Sequence,
					&solomachine.ConsensusState{
						Timestamp: sm.Time + 1,
						PublicKey: sm.ConsensusState().PublicKey,
					}, false),
				prefix,
				proof,
				false,
			},

			{
				"proof is nil",
				sm.ClientState(),
				prefix,
				nil,
				false,
			},
			{
				"proof verification failed",
				sm.ClientState(),
				prefix,
				suite.GetInvalidProof(),
				false,
			},
		}

		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {

				var expSeq uint64
				if tc.clientState.ConsensusState != nil {
					expSeq = tc.clientState.Sequence + 1
				}

				// NOTE: to replicate the ordering of connection handshake, we must decrement proof height by 1
				height := clienttypes.NewHeight(sm.GetHeight().GetRevisionNumber(), sm.GetHeight().GetRevisionHeight()-2)

				err := tc.clientState.VerifyClientConsensusState(
					suite.store, suite.chainA.Codec, height, counterpartyClientIdentifier, consensusHeight, tc.prefix, tc.proof, consensusState,
				)

				if tc.expPass {
					suite.Require().NoError(err)
					suite.Require().Equal(expSeq, tc.clientState.Sequence)
					suite.Require().Equal(expSeq, suite.GetSequenceFromStore(), "sequence not updated in the store (%d) on valid test case %s", suite.GetSequenceFromStore(), tc.name)
				} else {
					suite.Require().Error(err)
				}
			})
		}
	}
}

func (suite *SoloMachineTestSuite) TestVerifyConnectionState() {
	counterparty := connectiontypes.NewCounterparty("clientB", testConnectionID, *prefix)
	conn := connectiontypes.NewConnectionEnd(connectiontypes.OPEN, "clientA", counterparty, connectiontypes.ExportedVersionsToProto(connectiontypes.GetCompatibleVersions()), 0)

	path := suite.solomachine.GetConnectionStatePath(testConnectionID)

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {

		value, err := solomachine.ConnectionStateSignBytes(suite.chainA.Codec, sm.Sequence, sm.Time, sm.Diversifier, path, conn)
		suite.Require().NoError(err)

		sig := sm.GenerateSignature(value)
		signatureDoc := &solomachine.TimestampedSignatureData{
			SignatureData: sig,
			Timestamp:     sm.Time,
		}

		proof, err := suite.chainA.Codec.Marshal(signatureDoc)
		suite.Require().NoError(err)

		testCases := []struct {
			name        string
			clientState *solomachine.ClientState
			prefix      exported.Prefix
			proof       []byte
			expPass     bool
		}{
			{
				"successful verification",
				sm.ClientState(),
				prefix,
				proof,
				true,
			},
			{
				"ApplyPrefix failed",
				sm.ClientState(),
				commitmenttypes.NewMerklePrefix([]byte{}),
				proof,
				false,
			},
			{
				"proof is nil",
				sm.ClientState(),
				prefix,
				nil,
				false,
			},
			{
				"proof verification failed",
				sm.ClientState(),
				prefix,
				suite.GetInvalidProof(),
				false,
			},
		}

		for i, tc := range testCases {
			tc := tc

			expSeq := tc.clientState.Sequence + 1

			err := tc.clientState.VerifyConnectionState(
				suite.store, suite.chainA.Codec, sm.GetHeight(), tc.prefix, tc.proof, testConnectionID, conn,
			)

			if tc.expPass {
				suite.Require().NoError(err, "valid test case %d failed: %s", i, tc.name)
				suite.Require().Equal(expSeq, tc.clientState.Sequence)
				suite.Require().Equal(expSeq, suite.GetSequenceFromStore(), "sequence not updated in the store (%d) on valid test case %d: %s", suite.GetSequenceFromStore(), i, tc.name)
			} else {
				suite.Require().Error(err, "invalid test case %d passed: %s", i, tc.name)
			}
		}
	}
}

func (suite *SoloMachineTestSuite) TestVerifyChannelState() {
	counterparty := channeltypes.NewCounterparty(testPortID, testChannelID)
	ch := channeltypes.NewChannel(channeltypes.OPEN, channeltypes.ORDERED, counterparty, []string{testConnectionID}, "1.0.0")

	path := suite.solomachine.GetChannelStatePath(testPortID, testChannelID)

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {

		value, err := solomachine.ChannelStateSignBytes(suite.chainA.Codec, sm.Sequence, sm.Time, sm.Diversifier, path, ch)
		suite.Require().NoError(err)

		sig := sm.GenerateSignature(value)
		signatureDoc := &solomachine.TimestampedSignatureData{
			SignatureData: sig,
			Timestamp:     sm.Time,
		}

		proof, err := suite.chainA.Codec.Marshal(signatureDoc)
		suite.Require().NoError(err)

		testCases := []struct {
			name        string
			clientState *solomachine.ClientState
			prefix      exported.Prefix
			proof       []byte
			expPass     bool
		}{
			{
				"successful verification",
				sm.ClientState(),
				prefix,
				proof,
				true,
			},
			{
				"ApplyPrefix failed",
				sm.ClientState(),
				nil,
				proof,
				false,
			},
			{
				"proof is nil",
				sm.ClientState(),
				prefix,
				nil,
				false,
			},
			{
				"proof verification failed",
				sm.ClientState(),
				prefix,
				suite.GetInvalidProof(),
				false,
			},
		}

		for i, tc := range testCases {
			tc := tc

			expSeq := tc.clientState.Sequence + 1

			err := tc.clientState.VerifyChannelState(
				suite.store, suite.chainA.Codec, sm.GetHeight(), tc.prefix, tc.proof, testPortID, testChannelID, ch,
			)

			if tc.expPass {
				suite.Require().NoError(err, "valid test case %d failed: %s", i, tc.name)
				suite.Require().Equal(expSeq, tc.clientState.Sequence)
				suite.Require().Equal(expSeq, suite.GetSequenceFromStore(), "sequence not updated in the store (%d) on valid test case %d: %s", suite.GetSequenceFromStore(), i, tc.name)
			} else {
				suite.Require().Error(err, "invalid test case %d passed: %s", i, tc.name)
			}
		}
	}
}

func (suite *SoloMachineTestSuite) TestVerifyPacketCommitment() {
	commitmentBytes := []byte("COMMITMENT BYTES")

	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {

		path := sm.GetPacketCommitmentPath(testPortID, testChannelID)

		value, err := solomachine.PacketCommitmentSignBytes(suite.chainA.Codec, sm.Sequence, sm.Time, sm.Diversifier, path, commitmentBytes)
		suite.Require().NoError(err)

		sig := sm.GenerateSignature(value)
		signatureDoc := &solomachine.TimestampedSignatureData{
			SignatureData: sig,
			Timestamp:     sm.Time,
		}

		proof, err := suite.chainA.Codec.Marshal(signatureDoc)
		suite.Require().NoError(err)

		testCases := []struct {
			name        string
			clientState *solomachine.ClientState
			prefix      exported.Prefix
			proof       []byte
			expPass     bool
		}{
			{
				"successful verification",
				sm.ClientState(),
				prefix,
				proof,
				true,
			},
			{
				"ApplyPrefix failed",
				sm.ClientState(),
				commitmenttypes.NewMerklePrefix([]byte{}),
				proof,
				false,
			},
			{
				"proof is nil",
				sm.ClientState(),
				prefix,
				nil,
				false,
			},
			{
				"proof verification failed",
				sm.ClientState(),
				prefix,
				suite.GetInvalidProof(),
				false,
			},
		}

		for i, tc := range testCases {
			tc := tc

			expSeq := tc.clientState.Sequence + 1
			ctx := suite.chainA.GetContext()

			err := tc.clientState.VerifyPacketCommitment(
				ctx, suite.store, suite.chainA.Codec, sm.GetHeight(), 0, 0, tc.prefix, tc.proof, testPortID, testChannelID, sm.Sequence, commitmentBytes,
			)

			if tc.expPass {
				suite.Require().NoError(err, "valid test case %d failed: %s", i, tc.name)
				suite.Require().Equal(expSeq, suite.GetSequenceFromStore(), "sequence not updated in the store (%d) on valid test case %d: %s", suite.GetSequenceFromStore(), i, tc.name)
			} else {
				suite.Require().Error(err, "invalid test case %d passed: %s", i, tc.name)
			}
		}
	}
}

func (suite *SoloMachineTestSuite) TestVerifyPacketAcknowledgement() {
	ack := []byte("ACK")
	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {

		path := sm.GetPacketAcknowledgementPath(testPortID, testChannelID)

		value, err := solomachine.PacketAcknowledgementSignBytes(suite.chainA.Codec, sm.Sequence, sm.Time, sm.Diversifier, path, ack)
		suite.Require().NoError(err)

		sig := sm.GenerateSignature(value)
		signatureDoc := &solomachine.TimestampedSignatureData{
			SignatureData: sig,
			Timestamp:     sm.Time,
		}

		proof, err := suite.chainA.Codec.Marshal(signatureDoc)
		suite.Require().NoError(err)

		testCases := []struct {
			name        string
			clientState *solomachine.ClientState
			prefix      exported.Prefix
			proof       []byte
			expPass     bool
		}{
			{
				"successful verification",
				sm.ClientState(),
				prefix,
				proof,
				true,
			},
			{
				"ApplyPrefix failed",
				sm.ClientState(),
				commitmenttypes.NewMerklePrefix([]byte{}),
				proof,
				false,
			},
			{
				"proof is nil",
				sm.ClientState(),
				prefix,
				nil,
				false,
			},
			{
				"proof verification failed",
				sm.ClientState(),
				prefix,
				suite.GetInvalidProof(),
				false,
			},
		}

		for i, tc := range testCases {
			tc := tc

			expSeq := tc.clientState.Sequence + 1
			ctx := suite.chainA.GetContext()

			err := tc.clientState.VerifyPacketAcknowledgement(
				ctx, suite.store, suite.chainA.Codec, sm.GetHeight(), 0, 0, tc.prefix, tc.proof, testPortID, testChannelID, sm.Sequence, ack,
			)

			if tc.expPass {
				suite.Require().NoError(err, "valid test case %d failed: %s", i, tc.name)
				suite.Require().Equal(expSeq, suite.GetSequenceFromStore(), "sequence not updated in the store (%d) on valid test case %d: %s", suite.GetSequenceFromStore(), i, tc.name)
			} else {
				suite.Require().Error(err, "invalid test case %d passed: %s", i, tc.name)
			}
		}
	}
}

func (suite *SoloMachineTestSuite) TestVerifyPacketReceiptAbsence() {
	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {

		// absence uses receipt path as well
		path := sm.GetPacketReceiptPath(testPortID, testChannelID)

		value, err := solomachine.PacketReceiptAbsenceSignBytes(suite.chainA.Codec, sm.Sequence, sm.Time, sm.Diversifier, path)
		suite.Require().NoError(err)

		sig := sm.GenerateSignature(value)
		signatureDoc := &solomachine.TimestampedSignatureData{
			SignatureData: sig,
			Timestamp:     sm.Time,
		}

		proof, err := suite.chainA.Codec.Marshal(signatureDoc)
		suite.Require().NoError(err)

		testCases := []struct {
			name        string
			clientState *solomachine.ClientState
			prefix      exported.Prefix
			proof       []byte
			expPass     bool
		}{
			{
				"successful verification",
				sm.ClientState(),
				prefix,
				proof,
				true,
			},
			{
				"ApplyPrefix failed",
				sm.ClientState(),
				commitmenttypes.NewMerklePrefix([]byte{}),
				proof,
				false,
			},
			{
				"proof is nil",
				sm.ClientState(),
				prefix,
				nil,
				false,
			},
			{
				"proof verification failed",
				sm.ClientState(),
				prefix,
				suite.GetInvalidProof(),
				false,
			},
		}

		for i, tc := range testCases {
			tc := tc

			expSeq := tc.clientState.Sequence + 1
			ctx := suite.chainA.GetContext()

			err := tc.clientState.VerifyPacketReceiptAbsence(
				ctx, suite.store, suite.chainA.Codec, sm.GetHeight(), 0, 0, tc.prefix, tc.proof, testPortID, testChannelID, sm.Sequence,
			)

			if tc.expPass {
				suite.Require().NoError(err, "valid test case %d failed: %s", i, tc.name)
				suite.Require().Equal(expSeq, suite.GetSequenceFromStore(), "sequence not updated in the store (%d) on valid test case %d: %s", suite.GetSequenceFromStore(), i, tc.name)
			} else {
				suite.Require().Error(err, "invalid test case %d passed: %s", i, tc.name)
			}
		}
	}
}

func (suite *SoloMachineTestSuite) TestVerifyNextSeqRecv() {
	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {

		nextSeqRecv := sm.Sequence + 1
		path := sm.GetNextSequenceRecvPath(testPortID, testChannelID)

		value, err := solomachine.NextSequenceRecvSignBytes(suite.chainA.Codec, sm.Sequence, sm.Time, sm.Diversifier, path, nextSeqRecv)
		suite.Require().NoError(err)

		sig := sm.GenerateSignature(value)
		signatureDoc := &solomachine.TimestampedSignatureData{
			SignatureData: sig,
			Timestamp:     sm.Time,
		}

		proof, err := suite.chainA.Codec.Marshal(signatureDoc)
		suite.Require().NoError(err)

		testCases := []struct {
			name        string
			clientState *solomachine.ClientState
			prefix      exported.Prefix
			proof       []byte
			expPass     bool
		}{
			{
				"successful verification",
				sm.ClientState(),
				prefix,
				proof,
				true,
			},
			{
				"ApplyPrefix failed",
				sm.ClientState(),
				commitmenttypes.NewMerklePrefix([]byte{}),
				proof,
				false,
			},
			{
				"proof is nil",
				sm.ClientState(),
				prefix,
				nil,
				false,
			},
			{
				"proof verification failed",
				sm.ClientState(),
				prefix,
				suite.GetInvalidProof(),
				false,
			},
		}

		for i, tc := range testCases {
			tc := tc

			expSeq := tc.clientState.Sequence + 1
			ctx := suite.chainA.GetContext()

			err := tc.clientState.VerifyNextSequenceRecv(
				ctx, suite.store, suite.chainA.Codec, sm.GetHeight(), 0, 0, tc.prefix, tc.proof, testPortID, testChannelID, nextSeqRecv,
			)

			if tc.expPass {
				suite.Require().NoError(err, "valid test case %d failed: %s", i, tc.name)
				suite.Require().Equal(expSeq, tc.clientState.Sequence)
				suite.Require().Equal(expSeq, suite.GetSequenceFromStore(), "sequence not updated in the store (%d) on valid test case %d: %s", suite.GetSequenceFromStore(), i, tc.name)
			} else {
				suite.Require().Error(err, "invalid test case %d passed: %s", i, tc.name)
			}
		}
	}
}

func (suite *SoloMachineTestSuite) TestGetTimestampAtHeight() {
	tmPath := ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupClients(tmPath)
	// Single setup for all test cases.
	suite.SetupTest()

	testCases := []struct {
		name        string
		clientState *solomachine.ClientState
		height      exported.Height
		expValue    uint64
		expPass     bool
	}{
		{
			name:        "get timestamp at height exists",
			clientState: suite.solomachine.ClientState(),
			height:      suite.solomachine.ClientState().GetLatestHeight(),
			expValue:    suite.solomachine.ClientState().ConsensusState.Timestamp,
			expPass:     true,
		},
		{
			name:        "get timestamp at height not exists",
			clientState: suite.solomachine.ClientState(),
			height:      suite.solomachine.ClientState().GetLatestHeight().Increment(),
		},
	}

	for i, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			ctx := suite.chainA.GetContext()

			ts, err := tc.clientState.GetTimestampAtHeight(
				ctx, suite.store, suite.chainA.Codec, tc.height,
			)

			suite.Require().Equal(tc.expValue, ts)

			if tc.expPass {
				suite.Require().NoError(err, "valid test case %d failed: %s", i, tc.name)
			} else {
				suite.Require().Error(err, "invalid test case %d passed: %s", i, tc.name)
			}
		})
	}
}
