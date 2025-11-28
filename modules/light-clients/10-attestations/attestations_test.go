package attestations_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/runtime"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	attestations "github.com/cosmos/ibc-go/v10/modules/light-clients/10-attestations"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

const testClientID = "10-attestations-0"

type AttestationsTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator
	chainA      *ibctesting.TestChain
	chainB      *ibctesting.TestChain

	attestors         []*ecdsa.PrivateKey
	attestorAddrs     []string
	minRequiredSigs   uint32
	lightClientModule attestations.LightClientModule
}

func TestAttestationsTestSuite(t *testing.T) {
	testifysuite.Run(t, new(AttestationsTestSuite))
}

func (s *AttestationsTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))

	attestations.RegisterInterfaces(s.chainA.GetSimApp().InterfaceRegistry())

	s.attestors = make([]*ecdsa.PrivateKey, 5)
	s.attestorAddrs = make([]string, 5)
	for i := range 5 {
		privKey, err := crypto.GenerateKey()
		s.Require().NoError(err)
		s.attestors[i] = privKey
		s.attestorAddrs[i] = crypto.PubkeyToAddress(privKey.PublicKey).Hex()
	}

	s.minRequiredSigs = 3

	cdc := s.chainA.App.AppCodec()
	storeKey := s.chainA.GetSimApp().GetKey(exported.StoreKey)
	storeProvider := clienttypes.NewStoreProvider(runtime.NewKVStoreService(storeKey))
	s.lightClientModule = attestations.NewLightClientModule(cdc, storeProvider)
}

func (s *AttestationsTestSuite) createAttestationProof(attestationData []byte, signers []int) *attestations.AttestationProof {
	hash := sha256.Sum256(attestationData)
	signatures := make([][]byte, 0, len(signers))

	for _, idx := range signers {
		if idx < 0 || idx >= len(s.attestors) {
			continue
		}
		sig, err := crypto.Sign(hash[:], s.attestors[idx])
		s.Require().NoError(err)
		signatures = append(signatures, sig)
	}

	return &attestations.AttestationProof{
		AttestationData: attestationData,
		Signatures:      signatures,
	}
}

func (s *AttestationsTestSuite) createStateAttestation(height, timestamp uint64) []byte {
	stateAttestation := attestations.StateAttestation{
		Height:    height,
		Timestamp: timestamp,
	}
	cdc := s.chainA.App.AppCodec()
	data, err := cdc.Marshal(&stateAttestation)
	s.Require().NoError(err)
	return data
}

func (s *AttestationsTestSuite) createPacketAttestation(height uint64, packets []attestations.PacketCompact) []byte {
	packetAttestation := attestations.PacketAttestation{
		Height:  height,
		Packets: packets,
	}
	cdc := s.chainA.App.AppCodec()
	data, err := cdc.Marshal(&packetAttestation)
	s.Require().NoError(err)
	return data
}

func (s *AttestationsTestSuite) marshalProof(proof *attestations.AttestationProof) []byte {
	cdc := s.chainA.App.AppCodec()
	data, err := cdc.Marshal(proof)
	s.Require().NoError(err)
	return data
}

func (s *AttestationsTestSuite) createClientState(initialHeight uint64) *attestations.ClientState {
	return attestations.NewClientState(
		s.attestorAddrs,
		s.minRequiredSigs,
		initialHeight,
	)
}

func (s *AttestationsTestSuite) createConsensusState(timestamp uint64) *attestations.ConsensusState {
	return &attestations.ConsensusState{
		Timestamp: timestamp,
	}
}

func (s *AttestationsTestSuite) TestClientStateValidate() {
	testCases := []struct {
		name        string
		clientState *attestations.ClientState
		expErr      bool
	}{
		{
			"valid client state",
			s.createClientState(1),
			false,
		},
		{
			"zero latest height",
			attestations.NewClientState(s.attestorAddrs, s.minRequiredSigs, 0),
			true,
		},
		{
			"empty attestor addresses",
			attestations.NewClientState([]string{}, 1, 1),
			true,
		},
		{
			"zero min required sigs",
			attestations.NewClientState(s.attestorAddrs, 0, 1),
			true,
		},
		{
			"min required sigs exceeds attestor count",
			attestations.NewClientState(s.attestorAddrs, 10, 1),
			true,
		},
		{
			"duplicate attestor address",
			attestations.NewClientState([]string{s.attestorAddrs[0], s.attestorAddrs[0]}, 1, 1),
			true,
		},
		{
			"empty attestor address",
			attestations.NewClientState([]string{""}, 1, 1),
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.clientState.Validate()
			if tc.expErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *AttestationsTestSuite) TestConsensusStateValidateBasic() {
	testCases := []struct {
		name           string
		consensusState *attestations.ConsensusState
		expErr         bool
	}{
		{
			"valid consensus state",
			s.createConsensusState(1000),
			false,
		},
		{
			"zero timestamp",
			s.createConsensusState(0),
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.consensusState.ValidateBasic()
			if tc.expErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *AttestationsTestSuite) TestInitialize() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(1000)

	clientState := s.createClientState(initialHeight)
	consensusState := s.createConsensusState(initialTimestamp)

	clientStateBz, err := s.chainA.App.AppCodec().Marshal(clientState)
	s.Require().NoError(err)

	consensusStateBz, err := s.chainA.App.AppCodec().Marshal(consensusState)
	s.Require().NoError(err)

	clientID := testClientID
	ctx := s.chainA.GetContext()

	err = s.lightClientModule.Initialize(ctx, clientID, clientStateBz, consensusStateBz)
	s.Require().NoError(err)

	status := s.lightClientModule.Status(ctx, clientID)
	s.Require().Equal(exported.Active, status)

	latestHeight := s.lightClientModule.LatestHeight(ctx, clientID)
	s.Require().Equal(initialHeight, latestHeight.GetRevisionHeight())

	timestamp, err := s.lightClientModule.TimestampAtHeight(ctx, clientID, latestHeight)
	s.Require().NoError(err)
	s.Require().Equal(initialTimestamp, timestamp)
}

func (s *AttestationsTestSuite) TestUpdateState() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(1000)

	clientState := s.createClientState(initialHeight)
	consensusState := s.createConsensusState(initialTimestamp)

	clientStateBz, err := s.chainA.App.AppCodec().Marshal(clientState)
	s.Require().NoError(err)

	consensusStateBz, err := s.chainA.App.AppCodec().Marshal(consensusState)
	s.Require().NoError(err)

	clientID := testClientID
	ctx := s.chainA.GetContext()

	err = s.lightClientModule.Initialize(ctx, clientID, clientStateBz, consensusStateBz)
	s.Require().NoError(err)

	newHeight := uint64(200)
	newTimestamp := uint64(2000)
	attestationData := s.createStateAttestation(newHeight, newTimestamp)

	signers := []int{0, 1, 2}
	proof := s.createAttestationProof(attestationData, signers)

	err = s.lightClientModule.VerifyClientMessage(ctx, clientID, proof)
	s.Require().NoError(err)

	heights := s.lightClientModule.UpdateState(ctx, clientID, proof)
	s.Require().Len(heights, 1)
	s.Require().Equal(newHeight, heights[0].GetRevisionHeight())

	latestHeight := s.lightClientModule.LatestHeight(ctx, clientID)
	s.Require().Equal(newHeight, latestHeight.GetRevisionHeight())

	timestamp, err := s.lightClientModule.TimestampAtHeight(ctx, clientID, latestHeight)
	s.Require().NoError(err)
	s.Require().Equal(newTimestamp, timestamp)
}

func (s *AttestationsTestSuite) TestUpdateStateInsufficientSignatures() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(1000)

	clientState := s.createClientState(initialHeight)
	consensusState := s.createConsensusState(initialTimestamp)

	clientStateBz, err := s.chainA.App.AppCodec().Marshal(clientState)
	s.Require().NoError(err)

	consensusStateBz, err := s.chainA.App.AppCodec().Marshal(consensusState)
	s.Require().NoError(err)

	clientID := testClientID
	ctx := s.chainA.GetContext()

	err = s.lightClientModule.Initialize(ctx, clientID, clientStateBz, consensusStateBz)
	s.Require().NoError(err)

	newHeight := uint64(200)
	newTimestamp := uint64(2000)
	attestationData := s.createStateAttestation(newHeight, newTimestamp)

	signers := []int{0, 1}
	proof := s.createAttestationProof(attestationData, signers)

	err = s.lightClientModule.VerifyClientMessage(ctx, clientID, proof)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "quorum")
}

func (s *AttestationsTestSuite) TestVerifyMembershipMatchingPathAndCommitment() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(time.Second.Nanoseconds())

	clientState := s.createClientState(initialHeight)
	consensusState := s.createConsensusState(initialTimestamp)

	clientStateBz, err := s.chainA.App.AppCodec().Marshal(clientState)
	s.Require().NoError(err)

	consensusStateBz, err := s.chainA.App.AppCodec().Marshal(consensusState)
	s.Require().NoError(err)

	clientID := testClientID
	ctx := s.chainA.GetContext()

	err = s.lightClientModule.Initialize(ctx, clientID, clientStateBz, consensusStateBz)
	s.Require().NoError(err)

	newHeight := uint64(200)
	newTimestamp := uint64(2 * time.Second.Nanoseconds())
	stateAttestation := s.createStateAttestation(newHeight, newTimestamp)
	signers := []int{0, 1, 2}
	updateProof := s.createAttestationProof(stateAttestation, signers)

	err = s.lightClientModule.VerifyClientMessage(ctx, clientID, updateProof)
	s.Require().NoError(err)

	heights := s.lightClientModule.UpdateState(ctx, clientID, updateProof)
	s.Require().Len(heights, 1)

	pathBytes := bytes.Repeat([]byte{0x01}, 32)
	commitment := bytes.Repeat([]byte{0xAB}, 32)
	packetAttestation := s.createPacketAttestation(newHeight, []attestations.PacketCompact{{Path: pathBytes, Commitment: commitment}})
	membershipProof := s.createAttestationProof(packetAttestation, signers)
	membershipProofBz := s.marshalProof(membershipProof)

	proofHeight := clienttypes.NewHeight(0, newHeight)
	err = s.lightClientModule.VerifyMembership(ctx, clientID, proofHeight, 0, 0, membershipProofBz, bytePath(pathBytes), commitment)
	s.Require().NoError(err)
}

func (s *AttestationsTestSuite) TestVerifyMembershipMismatchedPath() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(time.Second.Nanoseconds())

	clientState := s.createClientState(initialHeight)
	consensusState := s.createConsensusState(initialTimestamp)

	clientStateBz, err := s.chainA.App.AppCodec().Marshal(clientState)
	s.Require().NoError(err)

	consensusStateBz, err := s.chainA.App.AppCodec().Marshal(consensusState)
	s.Require().NoError(err)

	clientID := testClientID
	ctx := s.chainA.GetContext()

	err = s.lightClientModule.Initialize(ctx, clientID, clientStateBz, consensusStateBz)
	s.Require().NoError(err)

	newHeight := uint64(200)
	newTimestamp := uint64(2 * time.Second.Nanoseconds())
	stateAttestation := s.createStateAttestation(newHeight, newTimestamp)
	signers := []int{0, 1, 2}
	updateProof := s.createAttestationProof(stateAttestation, signers)

	err = s.lightClientModule.VerifyClientMessage(ctx, clientID, updateProof)
	s.Require().NoError(err)
	_ = s.lightClientModule.UpdateState(ctx, clientID, updateProof)

	attestedPath := bytes.Repeat([]byte{0x01}, 32)
	commitment := bytes.Repeat([]byte{0xCD}, 32)
	packetAttestation := s.createPacketAttestation(newHeight, []attestations.PacketCompact{{Path: attestedPath, Commitment: commitment}})
	membershipProof := s.createAttestationProof(packetAttestation, signers)
	membershipProofBz := s.marshalProof(membershipProof)

	proofHeight := clienttypes.NewHeight(0, newHeight)
	err = s.lightClientModule.VerifyMembership(ctx, clientID, proofHeight, 0, 0, membershipProofBz, bytePath(bytes.Repeat([]byte{0x02}, 32)), commitment)
	s.Require().ErrorIs(err, attestations.ErrNotMember)
}

func (s *AttestationsTestSuite) TestCheckForMisbehaviour() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(1000)

	clientState := s.createClientState(initialHeight)
	consensusState := s.createConsensusState(initialTimestamp)

	clientStateBz, err := s.chainA.App.AppCodec().Marshal(clientState)
	s.Require().NoError(err)

	consensusStateBz, err := s.chainA.App.AppCodec().Marshal(consensusState)
	s.Require().NoError(err)

	clientID := testClientID
	ctx := s.chainA.GetContext()

	err = s.lightClientModule.Initialize(ctx, clientID, clientStateBz, consensusStateBz)
	s.Require().NoError(err)

	newHeight := uint64(200)
	newTimestamp := uint64(2000)
	attestationData := s.createStateAttestation(newHeight, newTimestamp)

	signers := []int{0, 1, 2}
	proof := s.createAttestationProof(attestationData, signers)

	err = s.lightClientModule.VerifyClientMessage(ctx, clientID, proof)
	s.Require().NoError(err)

	heights := s.lightClientModule.UpdateState(ctx, clientID, proof)
	s.Require().Len(heights, 1)

	conflictingTimestamp := uint64(3000)
	conflictingAttestationData := s.createStateAttestation(newHeight, conflictingTimestamp)
	conflictingProof := s.createAttestationProof(conflictingAttestationData, signers)

	err = s.lightClientModule.VerifyClientMessage(ctx, clientID, conflictingProof)
	s.Require().NoError(err)

	hasMisbehaviour := s.lightClientModule.CheckForMisbehaviour(ctx, clientID, conflictingProof)
	s.Require().True(hasMisbehaviour)
}

func (s *AttestationsTestSuite) TestStatus() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(1000)

	clientState := s.createClientState(initialHeight)
	consensusState := s.createConsensusState(initialTimestamp)

	clientStateBz, err := s.chainA.App.AppCodec().Marshal(clientState)
	s.Require().NoError(err)

	consensusStateBz, err := s.chainA.App.AppCodec().Marshal(consensusState)
	s.Require().NoError(err)

	clientID := testClientID
	ctx := s.chainA.GetContext()

	err = s.lightClientModule.Initialize(ctx, clientID, clientStateBz, consensusStateBz)
	s.Require().NoError(err)

	status := s.lightClientModule.Status(ctx, clientID)
	s.Require().Equal(exported.Active, status)

	newHeight := uint64(200)
	newTimestamp := uint64(2000)
	attestationData := s.createStateAttestation(newHeight, newTimestamp)

	signers := []int{0, 1, 2}
	proof := s.createAttestationProof(attestationData, signers)

	err = s.lightClientModule.VerifyClientMessage(ctx, clientID, proof)
	s.Require().NoError(err)

	hasMisbehaviour := s.lightClientModule.CheckForMisbehaviour(ctx, clientID, proof)
	if hasMisbehaviour {
		s.lightClientModule.UpdateStateOnMisbehaviour(ctx, clientID, proof)
		status = s.lightClientModule.Status(ctx, clientID)
		s.Require().Equal(exported.Frozen, status)
	}
}

func (s *AttestationsTestSuite) TestClientStateValidateInvalidAddressFormat() {
	testCases := []struct {
		name        string
		clientState *attestations.ClientState
		expErr      bool
		errContains string
	}{
		{
			"invalid address format - not hex",
			attestations.NewClientState([]string{"not-a-valid-address"}, 1, 1),
			true,
			"invalid attestor address format",
		},
		{
			"invalid address format - too short",
			attestations.NewClientState([]string{"0x1234"}, 1, 1),
			true,
			"invalid attestor address format",
		},
		{
			"valid checksummed address",
			attestations.NewClientState([]string{s.attestorAddrs[0]}, 1, 1),
			false,
			"",
		},
		{
			"valid lowercase address",
			attestations.NewClientState([]string{strings.ToLower(s.attestorAddrs[0])}, 1, 1),
			false,
			"",
		},
		{
			"duplicate addresses with different case",
			attestations.NewClientState([]string{s.attestorAddrs[0], strings.ToLower(s.attestorAddrs[0])}, 1, 1),
			true,
			"duplicate attestor address",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.clientState.Validate()
			if tc.expErr {
				s.Require().Error(err)
				if tc.errContains != "" {
					s.Require().ErrorContains(err, tc.errContains)
				}
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *AttestationsTestSuite) TestVerifyClientMessageFrozenClient() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(1000)

	clientState := s.createClientState(initialHeight)
	consensusState := s.createConsensusState(initialTimestamp)

	clientStateBz, err := s.chainA.App.AppCodec().Marshal(clientState)
	s.Require().NoError(err)

	consensusStateBz, err := s.chainA.App.AppCodec().Marshal(consensusState)
	s.Require().NoError(err)

	clientID := testClientID
	ctx := s.chainA.GetContext()

	err = s.lightClientModule.Initialize(ctx, clientID, clientStateBz, consensusStateBz)
	s.Require().NoError(err)

	newHeight := uint64(200)
	newTimestamp := uint64(2000)
	attestationData := s.createStateAttestation(newHeight, newTimestamp)
	signers := []int{0, 1, 2}
	proof := s.createAttestationProof(attestationData, signers)

	err = s.lightClientModule.VerifyClientMessage(ctx, clientID, proof)
	s.Require().NoError(err)
	_ = s.lightClientModule.UpdateState(ctx, clientID, proof)

	conflictingTimestamp := uint64(3000)
	conflictingAttestationData := s.createStateAttestation(newHeight, conflictingTimestamp)
	conflictingProof := s.createAttestationProof(conflictingAttestationData, signers)

	err = s.lightClientModule.VerifyClientMessage(ctx, clientID, conflictingProof)
	s.Require().NoError(err)

	hasMisbehaviour := s.lightClientModule.CheckForMisbehaviour(ctx, clientID, conflictingProof)
	s.Require().True(hasMisbehaviour)

	s.lightClientModule.UpdateStateOnMisbehaviour(ctx, clientID, conflictingProof)

	status := s.lightClientModule.Status(ctx, clientID)
	s.Require().Equal(exported.Frozen, status)

	newProofData := s.createStateAttestation(uint64(300), uint64(4000))
	newProof := s.createAttestationProof(newProofData, signers)

	err = s.lightClientModule.VerifyClientMessage(ctx, clientID, newProof)
	s.Require().ErrorIs(err, attestations.ErrClientFrozen)
}

func (s *AttestationsTestSuite) TestVerifyMembershipFrozenClient() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(time.Second.Nanoseconds())

	clientState := s.createClientState(initialHeight)
	consensusState := s.createConsensusState(initialTimestamp)

	clientStateBz, err := s.chainA.App.AppCodec().Marshal(clientState)
	s.Require().NoError(err)

	consensusStateBz, err := s.chainA.App.AppCodec().Marshal(consensusState)
	s.Require().NoError(err)

	clientID := testClientID
	ctx := s.chainA.GetContext()

	err = s.lightClientModule.Initialize(ctx, clientID, clientStateBz, consensusStateBz)
	s.Require().NoError(err)

	newHeight := uint64(200)
	newTimestamp := uint64(2 * time.Second.Nanoseconds())
	stateAttestation := s.createStateAttestation(newHeight, newTimestamp)
	signers := []int{0, 1, 2}
	updateProof := s.createAttestationProof(stateAttestation, signers)

	err = s.lightClientModule.VerifyClientMessage(ctx, clientID, updateProof)
	s.Require().NoError(err)
	_ = s.lightClientModule.UpdateState(ctx, clientID, updateProof)

	conflictingTimestamp := uint64(3 * time.Second.Nanoseconds())
	conflictingData := s.createStateAttestation(newHeight, conflictingTimestamp)
	conflictingProof := s.createAttestationProof(conflictingData, signers)

	s.lightClientModule.UpdateStateOnMisbehaviour(ctx, clientID, conflictingProof)

	pathBytes := bytes.Repeat([]byte{0x01}, 32)
	commitment := bytes.Repeat([]byte{0xAB}, 32)
	packetAttestation := s.createPacketAttestation(newHeight, []attestations.PacketCompact{{Path: pathBytes, Commitment: commitment}})
	membershipProof := s.createAttestationProof(packetAttestation, signers)
	membershipProofBz := s.marshalProof(membershipProof)

	proofHeight := clienttypes.NewHeight(0, newHeight)
	err = s.lightClientModule.VerifyMembership(ctx, clientID, proofHeight, 0, 0, membershipProofBz, bytePath(pathBytes), commitment)
	s.Require().ErrorIs(err, attestations.ErrClientFrozen)
}

func (s *AttestationsTestSuite) TestVerifySignaturesInvalidLength() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(1000)

	clientState := s.createClientState(initialHeight)
	consensusState := s.createConsensusState(initialTimestamp)

	clientStateBz, err := s.chainA.App.AppCodec().Marshal(clientState)
	s.Require().NoError(err)

	consensusStateBz, err := s.chainA.App.AppCodec().Marshal(consensusState)
	s.Require().NoError(err)

	clientID := testClientID
	ctx := s.chainA.GetContext()

	err = s.lightClientModule.Initialize(ctx, clientID, clientStateBz, consensusStateBz)
	s.Require().NoError(err)

	newHeight := uint64(200)
	newTimestamp := uint64(2000)
	attestationData := s.createStateAttestation(newHeight, newTimestamp)

	proof := &attestations.AttestationProof{
		AttestationData: attestationData,
		Signatures:      [][]byte{bytes.Repeat([]byte{0x01}, 32)},
	}

	err = s.lightClientModule.VerifyClientMessage(ctx, clientID, proof)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "invalid length")
}

func (s *AttestationsTestSuite) TestVerifySignaturesUnknownSigner() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(1000)

	clientState := s.createClientState(initialHeight)
	consensusState := s.createConsensusState(initialTimestamp)

	clientStateBz, err := s.chainA.App.AppCodec().Marshal(clientState)
	s.Require().NoError(err)

	consensusStateBz, err := s.chainA.App.AppCodec().Marshal(consensusState)
	s.Require().NoError(err)

	clientID := testClientID
	ctx := s.chainA.GetContext()

	err = s.lightClientModule.Initialize(ctx, clientID, clientStateBz, consensusStateBz)
	s.Require().NoError(err)

	unknownKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	newHeight := uint64(200)
	newTimestamp := uint64(2000)
	attestationData := s.createStateAttestation(newHeight, newTimestamp)

	hash := sha256.Sum256(attestationData)
	unknownSig, err := crypto.Sign(hash[:], unknownKey)
	s.Require().NoError(err)

	proof := &attestations.AttestationProof{
		AttestationData: attestationData,
		Signatures:      [][]byte{unknownSig},
	}

	err = s.lightClientModule.VerifyClientMessage(ctx, clientID, proof)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "not in attestor set")
}

func (s *AttestationsTestSuite) TestVerifySignaturesDuplicateSigner() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(1000)

	clientState := s.createClientState(initialHeight)
	consensusState := s.createConsensusState(initialTimestamp)

	clientStateBz, err := s.chainA.App.AppCodec().Marshal(clientState)
	s.Require().NoError(err)

	consensusStateBz, err := s.chainA.App.AppCodec().Marshal(consensusState)
	s.Require().NoError(err)

	clientID := testClientID
	ctx := s.chainA.GetContext()

	err = s.lightClientModule.Initialize(ctx, clientID, clientStateBz, consensusStateBz)
	s.Require().NoError(err)

	newHeight := uint64(200)
	newTimestamp := uint64(2000)
	attestationData := s.createStateAttestation(newHeight, newTimestamp)

	hash := sha256.Sum256(attestationData)
	sig1, err := crypto.Sign(hash[:], s.attestors[0])
	s.Require().NoError(err)
	sig2, err := crypto.Sign(hash[:], s.attestors[0])
	s.Require().NoError(err)

	proof := &attestations.AttestationProof{
		AttestationData: attestationData,
		Signatures:      [][]byte{sig1, sig2},
	}

	err = s.lightClientModule.VerifyClientMessage(ctx, clientID, proof)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "duplicate signer")
}

func (s *AttestationsTestSuite) TestVerifySignaturesEmptySignatures() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(1000)

	clientState := s.createClientState(initialHeight)
	consensusState := s.createConsensusState(initialTimestamp)

	clientStateBz, err := s.chainA.App.AppCodec().Marshal(clientState)
	s.Require().NoError(err)

	consensusStateBz, err := s.chainA.App.AppCodec().Marshal(consensusState)
	s.Require().NoError(err)

	clientID := testClientID
	ctx := s.chainA.GetContext()

	err = s.lightClientModule.Initialize(ctx, clientID, clientStateBz, consensusStateBz)
	s.Require().NoError(err)

	newHeight := uint64(200)
	newTimestamp := uint64(2000)
	attestationData := s.createStateAttestation(newHeight, newTimestamp)

	proof := &attestations.AttestationProof{
		AttestationData: attestationData,
		Signatures:      [][]byte{},
	}

	err = s.lightClientModule.VerifyClientMessage(ctx, clientID, proof)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "signatures cannot be empty")
}

func (s *AttestationsTestSuite) TestUpdateStateOnMisbehaviourFreezesClient() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(1000)

	clientState := s.createClientState(initialHeight)
	consensusState := s.createConsensusState(initialTimestamp)

	clientStateBz, err := s.chainA.App.AppCodec().Marshal(clientState)
	s.Require().NoError(err)

	consensusStateBz, err := s.chainA.App.AppCodec().Marshal(consensusState)
	s.Require().NoError(err)

	clientID := testClientID
	ctx := s.chainA.GetContext()

	err = s.lightClientModule.Initialize(ctx, clientID, clientStateBz, consensusStateBz)
	s.Require().NoError(err)

	status := s.lightClientModule.Status(ctx, clientID)
	s.Require().Equal(exported.Active, status)

	newHeight := uint64(200)
	newTimestamp := uint64(2000)
	attestationData := s.createStateAttestation(newHeight, newTimestamp)
	signers := []int{0, 1, 2}
	proof := s.createAttestationProof(attestationData, signers)

	err = s.lightClientModule.VerifyClientMessage(ctx, clientID, proof)
	s.Require().NoError(err)
	_ = s.lightClientModule.UpdateState(ctx, clientID, proof)

	conflictingTimestamp := uint64(3000)
	conflictingAttestationData := s.createStateAttestation(newHeight, conflictingTimestamp)
	conflictingProof := s.createAttestationProof(conflictingAttestationData, signers)

	s.lightClientModule.UpdateStateOnMisbehaviour(ctx, clientID, conflictingProof)

	status = s.lightClientModule.Status(ctx, clientID)
	s.Require().Equal(exported.Frozen, status)
}

func (s *AttestationsTestSuite) TestVerifyNonMembershipNotSupported() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(time.Second.Nanoseconds())

	clientState := s.createClientState(initialHeight)
	consensusState := s.createConsensusState(initialTimestamp)

	clientStateBz, err := s.chainA.App.AppCodec().Marshal(clientState)
	s.Require().NoError(err)

	consensusStateBz, err := s.chainA.App.AppCodec().Marshal(consensusState)
	s.Require().NoError(err)

	clientID := testClientID
	ctx := s.chainA.GetContext()

	err = s.lightClientModule.Initialize(ctx, clientID, clientStateBz, consensusStateBz)
	s.Require().NoError(err)

	proofHeight := clienttypes.NewHeight(0, initialHeight)
	err = s.lightClientModule.VerifyNonMembership(ctx, clientID, proofHeight, 0, 0, []byte{}, bytePath([]byte("test")))
	s.Require().Error(err)
	s.Require().ErrorContains(err, "verifyNonMembership is not supported")
}

func (s *AttestationsTestSuite) TestRecoverClientNotSupported() {
	ctx := s.chainA.GetContext()
	err := s.lightClientModule.RecoverClient(ctx, testClientID, "10-attestations-1")
	s.Require().Error(err)
	s.Require().ErrorContains(err, "recoverClient is not supported")
}

func (s *AttestationsTestSuite) TestVerifyUpgradeAndUpdateStateNotSupported() {
	ctx := s.chainA.GetContext()
	err := s.lightClientModule.VerifyUpgradeAndUpdateState(ctx, testClientID, nil, nil, nil, nil)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "cannot upgrade attestations client")
}

func (s *AttestationsTestSuite) TestAddressCaseInsensitiveComparison() {
	privKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	addr := crypto.PubkeyToAddress(privKey.PublicKey).Hex()

	lowercaseAddrs := []string{strings.ToLower(addr)}
	clientState := attestations.NewClientState(lowercaseAddrs, 1, 100)
	consensusState := s.createConsensusState(1000)

	clientStateBz, err := s.chainA.App.AppCodec().Marshal(clientState)
	s.Require().NoError(err)

	consensusStateBz, err := s.chainA.App.AppCodec().Marshal(consensusState)
	s.Require().NoError(err)

	clientID := "10-attestations-case-test"
	ctx := s.chainA.GetContext()

	err = s.lightClientModule.Initialize(ctx, clientID, clientStateBz, consensusStateBz)
	s.Require().NoError(err)

	newHeight := uint64(200)
	newTimestamp := uint64(2000)
	attestationData := s.createStateAttestation(newHeight, newTimestamp)

	hash := sha256.Sum256(attestationData)
	sig, err := crypto.Sign(hash[:], privKey)
	s.Require().NoError(err)

	proof := &attestations.AttestationProof{
		AttestationData: attestationData,
		Signatures:      [][]byte{sig},
	}

	err = s.lightClientModule.VerifyClientMessage(ctx, clientID, proof)
	s.Require().NoError(err)
}

type bytePath []byte

func (p bytePath) Empty() bool {
	return len(p) == 0
}

func (p bytePath) Bytes() []byte {
	return p
}
