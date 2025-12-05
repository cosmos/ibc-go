package attestations_test

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	"github.com/cosmos/ibc-go/v10/modules/light-clients/attestations"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

const testClientID = "attestations-0"

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
	data, err := stateAttestation.ABIEncode()
	s.Require().NoError(err)
	return data
}

// nolint:unparam
func (s *AttestationsTestSuite) createPacketAttestation(height uint64, packets []attestations.PacketCompact) []byte {
	packetAttestation := attestations.PacketAttestation{
		Height:  height,
		Packets: packets,
	}
	data, err := packetAttestation.ABIEncode()
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

func (*AttestationsTestSuite) createConsensusState(timestamp uint64) *attestations.ConsensusState {
	return &attestations.ConsensusState{
		Timestamp: timestamp,
	}
}

// nolint:unparam
func (s *AttestationsTestSuite) initializeClient(ctx sdk.Context, clientID string, initialHeight, initialTimestamp uint64) {
	clientState := s.createClientState(initialHeight)
	consensusState := s.createConsensusState(initialTimestamp)

	clientStateBz, err := s.chainA.App.AppCodec().Marshal(clientState)
	s.Require().NoError(err)

	consensusStateBz, err := s.chainA.App.AppCodec().Marshal(consensusState)
	s.Require().NoError(err)

	err = s.lightClientModule.Initialize(ctx, clientID, clientStateBz, consensusStateBz)
	s.Require().NoError(err)
}

// nolint:unparam
func (s *AttestationsTestSuite) freezeClient(ctx sdk.Context, clientID string) {
	clientStore := s.chainA.GetSimApp().GetIBCKeeper().ClientKeeper.ClientStore(ctx, clientID)
	cdc := s.chainA.App.AppCodec()

	clientStateBz := clientStore.Get(host.ClientStateKey())
	s.Require().NotEmpty(clientStateBz)

	clientStateI := clienttypes.MustUnmarshalClientState(cdc, clientStateBz)
	clientState, ok := clientStateI.(*attestations.ClientState)
	s.Require().True(ok)

	clientState.IsFrozen = true
	clientStore.Set(host.ClientStateKey(), clienttypes.MustMarshalClientState(cdc, clientState))
}

func (s *AttestationsTestSuite) updateClientState(ctx sdk.Context, clientID string, height, timestamp uint64) {
	stateAttestation := s.createStateAttestation(height, timestamp)
	signers := []int{0, 1, 2}
	proof := s.createAttestationProof(stateAttestation, signers)
	_ = s.lightClientModule.UpdateState(ctx, clientID, proof)
}

func (s *AttestationsTestSuite) TestInitialize() {
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

	status := s.lightClientModule.Status(ctx, clientID)
	s.Require().Equal(exported.Active, status)

	latestHeight := s.lightClientModule.LatestHeight(ctx, clientID)
	s.Require().Equal(initialHeight, latestHeight.GetRevisionHeight())

	timestamp, err := s.lightClientModule.TimestampAtHeight(ctx, clientID, latestHeight)
	s.Require().NoError(err)
	s.Require().Equal(uint64(time.Second.Nanoseconds()), timestamp)
}

func (s *AttestationsTestSuite) TestStatus() {
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

	status := s.lightClientModule.Status(ctx, clientID)
	s.Require().Equal(exported.Active, status)

	s.freezeClient(ctx, clientID)
	status = s.lightClientModule.Status(ctx, clientID)
	s.Require().Equal(exported.Frozen, status)
}

func (s *AttestationsTestSuite) TestRecoverClientNotSupported() {
	ctx := s.chainA.GetContext()
	err := s.lightClientModule.RecoverClient(ctx, testClientID, "attestations-1")
	s.Require().Error(err)
	s.Require().ErrorContains(err, "recoverClient is not supported")
}

func (s *AttestationsTestSuite) TestVerifyUpgradeAndUpdateStateNotSupported() {
	ctx := s.chainA.GetContext()
	err := s.lightClientModule.VerifyUpgradeAndUpdateState(ctx, testClientID, nil, nil, nil, nil)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "cannot upgrade attestations client")
}

func (s *AttestationsTestSuite) TestCheckForMisbehaviourReturnsFalse() {
	initialHeight := uint64(100)
	initialTimestamp := uint64(time.Second.Nanoseconds())

	ctx := s.chainA.GetContext()
	s.initializeClient(ctx, testClientID, initialHeight, initialTimestamp)

	stateAttestation := s.createStateAttestation(initialHeight+1, initialTimestamp+uint64(time.Second.Nanoseconds()))
	signers := []int{0, 1, 2}
	proof := s.createAttestationProof(stateAttestation, signers)

	foundMisbehaviour := s.lightClientModule.CheckForMisbehaviour(ctx, testClientID, proof)
	s.Require().False(foundMisbehaviour, "CheckForMisbehaviour should return false")
}
