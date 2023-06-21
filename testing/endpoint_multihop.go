package ibctesting

import (
	"fmt"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	"github.com/cosmos/ibc-go/v7/modules/core/multihop"
	"github.com/stretchr/testify/require"
)

// EndpointM represents a multihop channel endpoint.
// It includes all intermediate endpoints in the linked paths.
// Invariants:
//   - paths[0].A == this.Endpoint
//   - paths[len(paths)-1].B == this.Counterparty
//   - self.paths.Reverse() == self.Counterparty.paths
//
// None of the fields should be changed after creation.
type EndpointM struct {
	*Endpoint
	Counterparty *EndpointM

	// a list of single-hop Paths that are linked together,
	// eg. for chains {A,B,C,D} the linked Paths would be Link{AB, BC, CD}
	Paths     LinkedPaths
	mChanPath multihop.ChanPath
}

// NewEndpointMFromLinkedPaths constructs a new EndpointM without the counterparty.
// CONTRACT: the counterparty EndpointM must be set by the caller.
func NewEndpointMFromLinkedPaths(path LinkedPaths) (a, z EndpointM) {
	a.Paths = path
	a.Endpoint = a.Paths.A()
	a.Counterparty = &z

	z.Paths = path.Reverse()
	z.Endpoint = z.Paths.A()
	z.Counterparty = &a

	// create multihop channel paths
	a.mChanPath = a.Paths.ToMultihopChanPath()
	z.mChanPath = z.Paths.ToMultihopChanPath()
	return
}

// ChanOpenInit will construct and execute a MsgChannelOpenInit on the associated EndpointM.
func (ep *EndpointM) ChanOpenInit() error {
	msg := channeltypes.NewMsgChannelOpenInit(
		ep.ChannelConfig.PortID, ep.ChannelConfig.Version, ep.ChannelConfig.Order, ep.GetConnectionHops(),
		ep.Counterparty.ChannelConfig.PortID,
		ep.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := ep.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	ep.ChannelID, err = ParseChannelIDFromEvents(res.GetEvents())
	require.NoError(ep.Chain.T, err, "could not retrieve channel id from event")

	// update version to selected app version
	// NOTE: this update must be performed after SendMsgs()
	ep.ChannelConfig.Version = ep.GetChannel().Version
	return nil
}

// ChanOpenTry will construct and execute a MsgChannelOpenTry on the associated EndpointM.
func (ep *EndpointM) ChanOpenTry(chanInitHeight exported.Height) error {

	proof, proofHeight, err := ep.Counterparty.QueryChannelProof(chanInitHeight)
	if err != nil {
		return err
	}

	msg := channeltypes.NewMsgChannelOpenTry(
		ep.ChannelConfig.PortID, ep.ChannelConfig.Version, ep.ChannelConfig.Order, ep.GetConnectionHops(),
		ep.Counterparty.ChannelConfig.PortID, ep.Counterparty.ChannelID, ep.Counterparty.ChannelConfig.Version,
		proof, proofHeight,
		ep.Chain.SenderAccount.GetAddress().String(),
	)

	res, err := ep.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	if ep.ChannelID == "" {
		ep.ChannelID, err = ParseChannelIDFromEvents(res.GetEvents())
		require.NoError(ep.Chain.T, err, "could not retrieve channel id from event on chain %s", ep.Chain.ChainID)
	}

	// update version to selected channel version. NOTE: this update must be performed after SendMsgs()
	ep.ChannelConfig.Version = ep.GetChannel().Version

	return nil
}

// ChanOpenAck will construct and execute a MsgChannelOpenAck on the associated EndpointM.
func (ep *EndpointM) ChanOpenAck(chanTryHeight exported.Height) error {

	proof, proofHeight, err := ep.Counterparty.QueryChannelProof(chanTryHeight)
	if err != nil {
		return err
	}

	msg := channeltypes.NewMsgChannelOpenAck(
		ep.ChannelConfig.PortID, ep.ChannelID,
		ep.Counterparty.ChannelID, ep.Counterparty.ChannelConfig.Version,
		proof, proofHeight,
		ep.Chain.SenderAccount.GetAddress().String(),
	)
	if _, err := ep.Chain.SendMsgs(msg); err != nil {
		return err
	}

	ep.ChannelConfig.Version = ep.GetChannel().Version

	return nil
}

// ChanOpenConfirm will construct and execute a MsgChannelOpenConfirm on the associated EndpointM.
func (ep *EndpointM) ChanOpenConfirm(chanAckHeight exported.Height) error {

	proof, proofHeight, err := ep.Counterparty.QueryChannelProof(chanAckHeight)
	if err != nil {
		return err
	}

	msg := channeltypes.NewMsgChannelOpenConfirm(
		ep.ChannelConfig.PortID, ep.ChannelID,
		proof, proofHeight,
		ep.Chain.SenderAccount.GetAddress().String(),
	)
	_, err = ep.Chain.SendMsgs(msg)
	return err
}

// ChanCloseInit will construct and execute a MsgChannelCloseInit on the associated EndpointM.
//
// NOTE: does not work with ibc-transfer module
func (ep *EndpointM) ChanCloseInit() error {
	return nil
}

// SendPacket sends a packet through the channel keeper using the associated EndpointM
// The counterparty client is updated so proofs can be sent to the counterparty chain.
// The packet sequence generated for the packet to be sent is returned. An error
// is returned if one occurs.
//
// The counterparty and all intermediate chains' clients are updated.
func (ep *EndpointM) SendPacket(
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (*channeltypes.Packet, exported.Height, error) {
	portID, channelID := ep.ChannelConfig.PortID, ep.ChannelID
	channelCap := ep.Chain.GetChannelCapability(portID, channelID)

	seq, err := ep.Chain.App.GetIBCKeeper().ChannelKeeper.SendPacket(
		ep.Chain.GetContext(),
		channelCap,
		portID, channelID,
		timeoutHeight,
		timeoutTimestamp,
		data,
	)
	if err != nil {
		return nil, nil, err
	}
	ep.Chain.Coordinator.CommitBlock(ep.Chain)

	packet := channeltypes.NewPacket(data, seq, portID, channelID,
		ep.Counterparty.ChannelConfig.PortID, ep.Counterparty.ChannelID,
		timeoutHeight, timeoutTimestamp,
	)
	return &packet, ep.Chain.LastHeader.GetHeight(), nil
}

// RecvPacket receives a packet on the associated EndpointM.
// The counterparty and all intermediate chains' clients are updated.
func (ep *EndpointM) RecvPacket(packet *channeltypes.Packet, packetHeight exported.Height) error {
	proof, proofHeight, err := ep.Counterparty.QueryPacketProof(packet, packetHeight)
	if err != nil {
		return err
	}

	recvMsg := channeltypes.NewMsgRecvPacket(
		*packet,
		proof,
		proofHeight,
		ep.Chain.SenderAccount.GetAddress().String(),
	)
	_, err = ep.Chain.SendMsgs(recvMsg)
	if err != nil {
		return err
	}

	return nil
}

// AcknowledgePacket sends a MsgAcknowledgement to the channel associated with the endpoint.
func (ep *EndpointM) AcknowledgePacket(packet channeltypes.Packet, ackHeight exported.Height, ack []byte) error {
	// get proof of acknowledgement on counterparty
	proof, proofHeight, err := ep.Counterparty.QueryPacketAcknowledgementProof(&packet, ackHeight)
	if err != nil {
		return err
	}

	ackMsg := channeltypes.NewMsgAcknowledgement(packet, ack, proof, proofHeight, ep.Chain.SenderAccount.GetAddress().String())

	return ep.Chain.sendMsgs(ackMsg)
}

// TimeoutPacket sends a MsgTimeout to the channel associated with the endpoint.
func (ep *EndpointM) TimeoutPacket(packet channeltypes.Packet, timeoutHeight exported.Height) error {
	// get proof for timeout based on channel order
	proof, proofHeight, err := ep.Counterparty.QueryPacketTimeoutProof(&packet, timeoutHeight)
	if err != nil {
		return err
	}

	nextSeqRecv, found := ep.Counterparty.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceRecv(ep.Counterparty.Chain.GetContext(), ep.ChannelConfig.PortID, ep.ChannelID)
	require.True(ep.Chain.T, found)

	timeoutMsg := channeltypes.NewMsgTimeout(
		packet, nextSeqRecv,
		proof, proofHeight, ep.Chain.SenderAccount.GetAddress().String(),
	)

	return ep.Chain.sendMsgs(timeoutMsg)
}

// SetChannelClosed sets a channel state to CLOSED.
func (ep *EndpointM) SetChannelClosed() {
	channel := ep.GetChannel()

	channel.State = channeltypes.CLOSED
	ep.Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(
		ep.Chain.GetContext(),
		ep.ChannelConfig.PortID,
		ep.ChannelID,
		channel,
	)

	ep.Chain.Coordinator.CommitBlock(ep.Chain)
}

// GetConnectionHops returns the connection hops for the multihop channel.
func (ep *EndpointM) GetConnectionHops() []string {
	return ep.mChanPath.GetConnectionHops()
}

// CounterpartyChannel returns the counterparty channel used in tx Msgs.
func (ep *EndpointM) CounterpartyChannel() channeltypes.Counterparty {
	return channeltypes.NewCounterparty(ep.Counterparty.ChannelConfig.PortID, ep.Counterparty.ChannelID)
}

// QueryChannelProof queries the multihop channel proof on the endpoint chain.
func (ep *EndpointM) QueryChannelProof(channelHeight exported.Height) ([]byte, clienttypes.Height, error) {
	channelKey := host.ChannelKey(ep.ChannelConfig.PortID, ep.ChannelID)
	return ep.QueryMultihopProof(channelKey, channelHeight)
}

// QueryFrozenClientProof queries proof of a frozen client in the multi-hop channel path.
func (ep *EndpointM) QueryFrozenClientProof(connectionID, clientID string, frozenHeight exported.Height) (proofConnection []byte, proofClientState []byte, proofHeight clienttypes.Height, err error) {
	connectionKey := host.ConnectionKey(connectionID)
	if proofConnection, proofHeight, err = ep.QueryMultihopProof(connectionKey, frozenHeight); err != nil {
		return
	}

	clientKey := host.FullClientStateKey(clientID)
	if proofClientState, _, err = ep.QueryMultihopProof(clientKey, frozenHeight); err != nil {
		return
	}
	return
}

// QueryPacketProof queries the multihop packet proof on the endpoint chain.
func (ep *EndpointM) QueryPacketProof(packet *channeltypes.Packet, packetHeight exported.Height) ([]byte, clienttypes.Height, error) {
	packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	return ep.QueryMultihopProof(packetKey, packetHeight)
}

// QueryPacketAcknowledgementProof queries the multihop packet acknowledgement proof on the endpoint chain.
func (ep *EndpointM) QueryPacketAcknowledgementProof(packet *channeltypes.Packet, ackHeight exported.Height) ([]byte, clienttypes.Height, error) {
	packetKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	return ep.QueryMultihopProof(packetKey, ackHeight)
}

// QueryPacketTimeoutProof queries the multihop packet timeout proof on the endpoint chain.
func (ep *EndpointM) QueryPacketTimeoutProof(packet *channeltypes.Packet, packetHeight exported.Height) ([]byte, clienttypes.Height, error) {
	var packetKey []byte

	switch ep.ChannelConfig.Order {
	case channeltypes.ORDERED:
		packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
	case channeltypes.UNORDERED:
		packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	default:
		return nil, packetHeight.(clienttypes.Height), fmt.Errorf("unsupported order type %s", ep.ChannelConfig.Order)
	}

	return ep.QueryMultihopProof(packetKey, packetHeight)
}

// QueryMultihopProof queries the proof for a key/value on this endpoint, which is verified on the counterparty chain.
func (ep *EndpointM) QueryMultihopProof(key []byte, keyHeight exported.Height) (proof []byte, proofHeight clienttypes.Height, err error) {

	multiHopProof, height, err := ep.mChanPath.QueryMultihopProof(key, keyHeight)
	if err != nil {
		return
	}

	proof = ep.Chain.Codec.MustMarshal(&multiHopProof)

	proofHeight, ok := height.Increment().(clienttypes.Height)
	if !ok {
		err = sdkerrors.Wrap(channeltypes.ErrMultihopProofGeneration, "height type conversion failed")
	}

	return
}

// ProofHeight returns the proof height passed to this endpoint where the proof is generated for the counterparty chain.
func (ep *EndpointM) ProofHeight() clienttypes.Height {
	return ep.GetClientState().GetLatestHeight().(clienttypes.Height)
}

// multihopEndpoint implements the multihop.Endpoint interface for a TestChain endpoint.
type multihopEndpoint struct {
	testEndpoint *Endpoint
}

// MultihopEndpoint returns a multihop.Endpoint implementation for the test endpoint.
func (tep *Endpoint) MultihopEndpoint() multihop.Endpoint {
	return multihopEndpoint{tep}
}

var _ multihop.Endpoint = multihopEndpoint{}

// ChainID implements multihop.Endpoint
func (mep multihopEndpoint) ChainID() string {
	return mep.testEndpoint.Chain.ChainID
}

// ClientID implements multihop.Endpoint
func (mep multihopEndpoint) ClientID() string {
	return mep.testEndpoint.ClientID
}

// ConnectionID implements multihop.Endpoint
func (mep multihopEndpoint) ConnectionID() string {
	return mep.testEndpoint.ConnectionID
}

// Counterparty implements multihop.Endpoint
func (mep multihopEndpoint) Counterparty() multihop.Endpoint {
	return mep.testEndpoint.Counterparty.MultihopEndpoint()
}

func (mep multihopEndpoint) GetLatestHeight() exported.Height {
	return mep.testEndpoint.Chain.LastHeader.GetHeight()
}

// GetClientState implements multihop.Endpoint
func (mep multihopEndpoint) GetClientState() exported.ClientState {
	return mep.testEndpoint.GetClientState()
}

// GetConnection implements multihop.Endpoint
func (mep multihopEndpoint) GetConnection() (*connectiontypes.ConnectionEnd, error) {
	conn := mep.testEndpoint.GetConnection()
	return &conn, nil
}

// GetConsensusState implements multihop.Endpoint
func (mep multihopEndpoint) GetConsensusState(height exported.Height) (exported.ConsensusState, bool) {
	return mep.testEndpoint.Chain.GetConsensusState(mep.testEndpoint.ClientID, height)
}

// GetMerklePath implements multihop.Endpoint
func (mep multihopEndpoint) GetMerklePath(path string) (commitmenttypes.MerklePath, error) {
	return commitmenttypes.ApplyPrefix(
		mep.testEndpoint.Chain.GetPrefix(),
		commitmenttypes.NewMerklePath(path),
	)
}

// QueryStateAtHeight implements multihop.Endpoint
func (mep multihopEndpoint) QueryStateAtHeight(key []byte, height int64, doProof bool) ([]byte, []byte, error) {
	return mep.testEndpoint.Chain.QueryStateAtHeight(key, height, doProof)
}

func (mep multihopEndpoint) QueryProcessedHeight(consensusHeight exported.Height) (exported.Height, error) {
	return mep.testEndpoint.Chain.QueryProcessedHeight(mep.testEndpoint.ClientID, consensusHeight)
}

// QueryMinimumConsensusHeight returns the minimum height within the provided range at which the consensusState exists (processedHeight)
// and the height of the corresponding consensus state (consensusHeight).
func (mep multihopEndpoint) QueryMinimumConsensusHeight(minConsensusHeight exported.Height, maxConsensusHeight exported.Height) (exported.Height, exported.Height, error) {
	return mep.testEndpoint.Chain.QueryMinimumConsensusHeight(mep.testEndpoint.ClientID, minConsensusHeight, maxConsensusHeight)
}

// QueryMaximumProofHeight returns the maxmimum height which can be used to prove a key/val pair by search consecutive heights
// to find the first point at which the value changes for the given key.
func (mep multihopEndpoint) QueryMaximumProofHeight(key []byte, minKeyHeight exported.Height, maxKeyHeightLimit exported.Height) exported.Height {
	return mep.testEndpoint.Chain.QueryMaximumProofHeight(key, minKeyHeight, maxKeyHeightLimit)
}

// UpdateClient updates the IBC client associated with the endpoint.
// Returns error for non-existent clients.
func (mep multihopEndpoint) UpdateClient() (err error) {

	_, found := mep.testEndpoint.Chain.App.GetIBCKeeper().ClientKeeper.GetClientState(mep.testEndpoint.Chain.GetContext(), mep.testEndpoint.ClientID)
	if !found {
		return fmt.Errorf("client=%s not found on chain=%s", mep.testEndpoint.ClientID, mep.testEndpoint.Chain.ChainID)
	}

	return mep.testEndpoint.UpdateClient()
}
