package ibctesting

import (
	"github.com/cosmos/cosmos-sdk/codec"
	clienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v6/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v6/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	"github.com/cosmos/ibc-go/v6/modules/core/exported"
	"github.com/cosmos/ibc-go/v6/modules/core/multihop"
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
	// eg. for chains {A,B,C,D} the linked paths would be Link{AB, BC, CD}
	paths     LinkedPaths
	mChanPath multihop.ChanPath
}

// NewEndpointM constructs a new EndpointM without the counterparty.
// CONTRACT: the counterparty EndpointM must be set by the caller.
func NewEndpointMFromLinkedPaths(path LinkedPaths) (A, Z EndpointM) {
	A.paths = path
	A.Endpoint = A.paths.A()
	A.Counterparty = &Z

	Z.paths = path.Reverse()
	Z.Endpoint = Z.paths.A()
	Z.Counterparty = &A

	// create multihop channel paths
	A.mChanPath = A.paths.ToMultihopChanPath()
	Z.mChanPath = Z.paths.ToMultihopChanPath()
	return A, Z
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
func (ep *EndpointM) ChanOpenTry() error {
	// propogate client state updates from A to Z
	err := ep.UpdateAllClients()
	if err != nil {
		return err
	}

	proof := ep.Counterparty.QueryChannelProof()
	unusedProofHeight := ep.GetClientState().GetLatestHeight().(clienttypes.Height)

	msg := channeltypes.NewMsgChannelOpenTry(
		ep.ChannelConfig.PortID, ep.ChannelConfig.Version, ep.ChannelConfig.Order, ep.GetConnectionHops(),
		ep.Counterparty.ChannelConfig.PortID, ep.Counterparty.ChannelID, ep.Counterparty.ChannelConfig.Version,
		proof, unusedProofHeight,
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
func (ep *EndpointM) ChanOpenAck() error {
	// propogate client state updates from Z to A
	err := ep.UpdateAllClients()
	if err != nil {
		return err
	}

	proof := ep.Counterparty.QueryChannelProof()
	unusedProofHeight := ep.GetClientState().GetLatestHeight().(clienttypes.Height)

	msg := channeltypes.NewMsgChannelOpenAck(
		ep.ChannelConfig.PortID, ep.ChannelID,
		ep.Counterparty.ChannelID, ep.Counterparty.ChannelConfig.Version,
		proof, unusedProofHeight,
		ep.Chain.SenderAccount.GetAddress().String(),
	)
	if _, err = ep.Chain.SendMsgs(msg); err != nil {
		return err
	}

	ep.ChannelConfig.Version = ep.GetChannel().Version

	return nil
}

// ChanOpenConfirm will construct and execute a MsgChannelOpenConfirm on the associated EndpointM.
func (ep *EndpointM) ChanOpenConfirm() error {
	// propogate client state updates from Z to A
	err := ep.UpdateAllClients()
	if err != nil {
		return err
	}

	proof := ep.Counterparty.QueryChannelProof()
	unusedProofHeight := ep.GetClientState().GetLatestHeight().(clienttypes.Height)

	msg := channeltypes.NewMsgChannelOpenConfirm(
		ep.ChannelConfig.PortID, ep.ChannelID,
		proof, unusedProofHeight,
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
) (*channeltypes.Packet, error) {
	portId, channelId := ep.ChannelConfig.PortID, ep.ChannelID
	channelCap := ep.Chain.GetChannelCapability(portId, channelId)

	seq, err := ep.Chain.App.GetIBCKeeper().ChannelKeeper.SendPacket(
		ep.Chain.GetContext(),
		channelCap,
		portId, channelId,
		timeoutHeight,
		timeoutTimestamp,
		data,
	)
	if err != nil {
		return nil, err
	}
	ep.Chain.Coordinator.CommitBlock(ep.Chain)
	require.NoError(ep.Chain.T, ep.Counterparty.UpdateAllClients())

	packet := channeltypes.NewPacket(data, seq, portId, channelId,
		ep.Counterparty.ChannelConfig.PortID, ep.Counterparty.ChannelID,
		timeoutHeight, timeoutTimestamp,
	)
	return &packet, nil
}

// RecvPacket receives a packet on the associated EndpointM.
// The counterparty and all intermediate chains' clients are updated.
func (ep *EndpointM) RecvPacket(packet *channeltypes.Packet) error {
	proof := ep.Counterparty.QueryPacketProof(packet)

	recvMsg := channeltypes.NewMsgRecvPacket(
		*packet,
		proof,
		ep.ProofHeight(),
		ep.Chain.SenderAccount.GetAddress().String(),
	)
	_, err := ep.Chain.SendMsgs(recvMsg)
	if err != nil {
		return err
	}

	require.NoError(ep.Chain.T, ep.Counterparty.UpdateAllClients())
	return nil
}

// SetChannelClosed sets a channel state to CLOSED.
func (ep *EndpointM) SetChannelClosed() error {
	channel := ep.GetChannel()

	channel.State = channeltypes.CLOSED
	ep.Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(
		ep.Chain.GetContext(),
		ep.ChannelConfig.PortID,
		ep.ChannelID,
		channel,
	)

	ep.Chain.Coordinator.CommitBlock(ep.Chain)

	return ep.Counterparty.UpdateClient()
}

// UpdateAllClients updates all client states starting from the first single-hop path to the last.
// ie. self's client state is propogated from the counterparty chain following the multihop channel path.
// This should be called on the chain that's about to receive a Msg with a proof.
func (ep *EndpointM) UpdateAllClients() error {
	return ep.Counterparty.mChanPath.UpdateClient()
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
func (ep *EndpointM) QueryChannelProof() []byte {
	channelKey := host.ChannelKey(ep.ChannelConfig.PortID, ep.ChannelID)
	return ep.QueryMultihopProof(channelKey)
}

// QueryPacketProof queries the multihop packet proof on the endpoint chain.
func (ep *EndpointM) QueryPacketProof(packet *channeltypes.Packet) []byte {
	packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	return ep.QueryMultihopProof(packetKey)
}

// QueryPacketAcknowledgementProof queries the multihop packet acknowledgement proof on the endpoint chain.
func (ep *EndpointM) QueryPacketAcknowledgementProof(packet *channeltypes.Packet) []byte {
	packetKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	return ep.QueryMultihopProof(packetKey)
}

// QueryMultihopProof queries the proof for a key/value on this endpoint, which is verified on the counterparty chain.
func (ep *EndpointM) QueryMultihopProof(key []byte) []byte {
	proof, err := ep.mChanPath.GenerateProof(key, nil, false)
	require.NoError(
		ep.Chain.T,
		err,
		"could not generate proof for key [%s] on chain [%s]",
		key,
		ep.Chain.ChainID,
	)
	return ep.Chain.Codec.MustMarshal(proof)
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

// Codec implements multihop.Endpoint
func (mep multihopEndpoint) Codec() codec.BinaryCodec {
	return mep.testEndpoint.Chain.Codec
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
func (mep multihopEndpoint) GetConsensusState(height exported.Height) (exported.ConsensusState, error) {
	return mep.testEndpoint.GetConsensusState(height), nil
}

// GetMerklePath implements multihop.Endpoint
func (mep multihopEndpoint) GetMerklePath(path string) (commitmenttypes.MerklePath, error) {
	return commitmenttypes.ApplyPrefix(
		mep.testEndpoint.Chain.GetPrefix(),
		commitmenttypes.NewMerklePath(path),
	)
}

// QueryProofAtHeight implements multihop.Endpoint
func (mep multihopEndpoint) QueryProofAtHeight(key []byte, height int64) ([]byte, clienttypes.Height, error) {
	proof, proofHeight := mep.testEndpoint.Chain.QueryProofAtHeight(key, height)
	return proof, proofHeight, nil
}

// UpdateClient implements multihop.Endpoint
func (mep multihopEndpoint) UpdateClient() error {
	return mep.testEndpoint.UpdateClient()
}
