package solomachine_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testifysuite "github.com/stretchr/testify/suite"

	storetypes "cosmossdk.io/store/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v10/modules/light-clients/06-solomachine"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/cosmos/ibc-go/v10/testing/mock"
)

var channelIDSolomachine = "channel-on-solomachine" // channelID generated on solo machine side

type SoloMachineTestSuite struct {
	testifysuite.Suite

	solomachine      *ibctesting.Solomachine // singlesig public key
	solomachineMulti *ibctesting.Solomachine // multisig public key
	coordinator      *ibctesting.Coordinator

	// testing chain used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	store storetypes.KVStore
}

func (s *SoloMachineTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))

	s.solomachine = ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "06-solomachine-0", "testing", 1)
	s.solomachineMulti = ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "06-solomachine-1", "testing", 4)

	s.store = s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), exported.Solomachine)
}

func TestSoloMachineTestSuite(t *testing.T) {
	testifysuite.Run(t, new(SoloMachineTestSuite))
}

func (s *SoloMachineTestSuite) SetupSolomachine() string {
	clientID := s.solomachine.CreateClient(s.chainA)

	connectionID := s.solomachine.ConnOpenInit(s.chainA, clientID)

	// open try is not necessary as the solo machine implementation is mocked

	s.solomachine.ConnOpenAck(s.chainA, clientID, connectionID)

	// open confirm is not necessary as the solo machine implementation is mocked

	channelID := s.solomachine.ChanOpenInit(s.chainA, connectionID)

	// open try is not necessary as the solo machine implementation is mocked

	s.solomachine.ChanOpenAck(s.chainA, channelID)

	// open confirm is not necessary as the solo machine implementation is mocked

	return channelID
}

func (s *SoloMachineTestSuite) TestRecvPacket() {
	channelID := s.SetupSolomachine()
	packet := channeltypes.NewPacket(
		mock.MockPacketData,
		1,
		transfertypes.PortID,
		channelIDSolomachine,
		transfertypes.PortID,
		channelID,
		clienttypes.ZeroHeight(),
		uint64(s.chainA.GetContext().BlockTime().Add(time.Hour).UnixNano()),
	)

	// send packet is not necessary as the solo machine implementation is mocked

	s.solomachine.RecvPacket(s.chainA, packet)

	// close init is not necessary as the solomachine implementation is mocked

	s.solomachine.ChanCloseConfirm(s.chainA, transfertypes.PortID, channelID)
}

func (s *SoloMachineTestSuite) TestAcknowledgePacket() {
	channelID := s.SetupSolomachine()

	packet := s.solomachine.SendTransfer(s.chainA, transfertypes.PortID, channelID)

	// recv packet is not necessary as the solo machine implementation is mocked

	s.solomachine.AcknowledgePacket(s.chainA, packet)

	// close init is not necessary as the solomachine implementation is mocked

	s.solomachine.ChanCloseConfirm(s.chainA, transfertypes.PortID, channelID)
}

func (s *SoloMachineTestSuite) TestTimeout() {
	channelID := s.SetupSolomachine()
	packet := s.solomachine.SendTransfer(s.chainA, transfertypes.PortID, channelID, func(msg *transfertypes.MsgTransfer) {
		msg.TimeoutTimestamp = s.solomachine.Time + 1
	})

	// simulate solomachine time increment
	s.solomachine.Time++

	s.solomachine.UpdateClient(s.chainA, ibctesting.DefaultSolomachineClientID)

	s.solomachine.TimeoutPacket(s.chainA, packet)

	s.solomachine.ChanCloseConfirm(s.chainA, transfertypes.PortID, channelID)
}

func (s *SoloMachineTestSuite) TestTimeoutOnClose() {
	channelID := s.SetupSolomachine()

	packet := s.solomachine.SendTransfer(s.chainA, transfertypes.PortID, channelID)

	s.solomachine.TimeoutPacketOnClose(s.chainA, packet, channelID)
}

func (s *SoloMachineTestSuite) GetSequenceFromStore() uint64 {
	bz := s.store.Get(host.ClientStateKey())
	s.Require().NotNil(bz)

	var clientState exported.ClientState
	err := s.chainA.Codec.UnmarshalInterface(bz, &clientState)
	s.Require().NoError(err)

	smClientState, ok := clientState.(*solomachine.ClientState)
	s.Require().True(ok)

	return smClientState.Sequence
}

func (s *SoloMachineTestSuite) GetInvalidProof() []byte {
	invalidProof, err := s.chainA.Codec.Marshal(&solomachine.TimestampedSignatureData{Timestamp: s.solomachine.Time})
	s.Require().NoError(err)

	return invalidProof
}

func TestUnpackInterfaces_Header(t *testing.T) {
	registry := testdata.NewTestInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)

	pk := secp256k1.GenPrivKey().PubKey()
	protoAny, err := codectypes.NewAnyWithValue(pk)
	require.NoError(t, err)

	header := solomachine.Header{
		NewPublicKey: protoAny,
	}
	bz, err := header.Marshal()
	require.NoError(t, err)

	var header2 solomachine.Header
	err = header2.Unmarshal(bz)
	require.NoError(t, err)

	err = codectypes.UnpackInterfaces(header2, registry)
	require.NoError(t, err)

	require.Equal(t, pk, header2.NewPublicKey.GetCachedValue())
}

func TestUnpackInterfaces_HeaderData(t *testing.T) {
	registry := testdata.NewTestInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)

	pk := secp256k1.GenPrivKey().PubKey()
	protoAny, err := codectypes.NewAnyWithValue(pk)
	require.NoError(t, err)

	hd := solomachine.HeaderData{
		NewPubKey: protoAny,
	}
	bz, err := hd.Marshal()
	require.NoError(t, err)

	var hd2 solomachine.HeaderData
	err = hd2.Unmarshal(bz)
	require.NoError(t, err)

	err = codectypes.UnpackInterfaces(hd2, registry)
	require.NoError(t, err)

	require.Equal(t, pk, hd2.NewPubKey.GetCachedValue())
}
