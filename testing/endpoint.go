package ibctesting

import (
	"errors"
	"fmt"
	"strings"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/baseapp"

	abci "github.com/cometbft/cometbft/abci/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types/v2"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
)

// Endpoint is a which represents a channel endpoint and its associated
// client and connections. It contains client, connection, and channel
// configuration parameters. Endpoint functions will utilize the parameters
// set in the configuration structs when executing IBC messages.
type Endpoint struct {
	Chain        *TestChain
	Counterparty *Endpoint
	ClientID     string
	ConnectionID string
	ChannelID    string

	ClientConfig     ClientConfig
	ConnectionConfig *ConnectionConfig
	ChannelConfig    *ChannelConfig

	MerklePathPrefix commitmenttypesv2.MerklePath
	// disableUniqueChannelIDs is used to enforce, in a test,
	// the old way to generate channel IDs (all channels are called channel-0)
	// It is used only by one test suite and should not be used for new tests.
	disableUniqueChannelIDs bool
}

// NewEndpoint constructs a new endpoint without the counterparty.
// CONTRACT: the counterparty endpoint must be set by the caller.
func NewEndpoint(
	chain *TestChain, clientConfig ClientConfig,
	connectionConfig *ConnectionConfig, channelConfig *ChannelConfig,
) *Endpoint {
	return &Endpoint{
		Chain:            chain,
		ClientConfig:     clientConfig,
		ConnectionConfig: connectionConfig,
		ChannelConfig:    channelConfig,
		MerklePathPrefix: MerklePath,
	}
}

// NewDefaultEndpoint constructs a new endpoint using default values.
// CONTRACT: the counterparty endpoitn must be set by the caller.
func NewDefaultEndpoint(chain *TestChain) *Endpoint {
	return &Endpoint{
		Chain:            chain,
		ClientConfig:     NewTendermintConfig(),
		ConnectionConfig: NewConnectionConfig(),
		ChannelConfig:    NewChannelConfig(),
		MerklePathPrefix: MerklePath,
	}
}

// QueryProof queries proof associated with this endpoint using the latest client state
// height on the counterparty chain.
func (ep *Endpoint) QueryProof(key []byte) ([]byte, clienttypes.Height) {
	// obtain the counterparty client height.
	latestCounterpartyHeight := ep.Counterparty.GetClientLatestHeight()
	// query proof on the counterparty using the latest height of the IBC client
	return ep.QueryProofAtHeight(key, latestCounterpartyHeight.GetRevisionHeight())
}

// QueryProofAtHeight queries proof associated with this endpoint using the proof height
// provided
func (ep *Endpoint) QueryProofAtHeight(key []byte, height uint64) ([]byte, clienttypes.Height) {
	// query proof on the counterparty using the latest height of the IBC client
	return ep.Chain.QueryProofAtHeight(key, int64(height))
}

// CreateClient creates an IBC client on the ep. It will update the
// clientID for the endpoint if the message is successfully executed.
// NOTE: a solo machine client will be created with an empty diversifier.
func (ep *Endpoint) CreateClient() error {
	// ensure counterparty has committed state
	ep.Counterparty.Chain.NextBlock()

	var (
		clientState    exported.ClientState
		consensusState exported.ConsensusState
	)

	switch ep.ClientConfig.GetClientType() {
	case exported.Tendermint:
		tmConfig, ok := ep.ClientConfig.(*TendermintConfig)
		require.True(ep.Chain.TB, ok)

		height, ok := ep.Counterparty.Chain.LatestCommittedHeader.GetHeight().(clienttypes.Height)
		require.True(ep.Chain.TB, ok)
		clientState = ibctm.NewClientState(
			ep.Counterparty.Chain.ChainID, tmConfig.TrustLevel, tmConfig.TrustingPeriod, tmConfig.UnbondingPeriod, tmConfig.MaxClockDrift,
			height, commitmenttypes.GetSDKSpecs(), UpgradePath)
		consensusState = ep.Counterparty.Chain.LatestCommittedHeader.ConsensusState()
	case exported.Solomachine:
		// TODO
		//		solo := NewSolomachine(ep.Chain.TB, ep.Chain.Codec, clientID, "", 1)
		//		clientState = solo.ClientState()
		//		consensusState = solo.ConsensusState()
	default:
		return fmt.Errorf("client type %s is not supported", ep.ClientConfig.GetClientType())
	}

	msg, err := clienttypes.NewMsgCreateClient(
		clientState, consensusState, ep.Chain.SenderAccount.GetAddress().String(),
	)
	require.NoError(ep.Chain.TB, err)

	res, err := ep.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	ep.ClientID, err = ParseClientIDFromEvents(res.Events)
	require.NoError(ep.Chain.TB, err)

	return nil
}

// UpdateClient updates the IBC client associated with the ep.
func (ep *Endpoint) UpdateClient() error {
	// ensure counterparty has committed state
	ep.Chain.Coordinator.CommitBlock(ep.Counterparty.Chain)

	var header exported.ClientMessage
	switch ep.ClientConfig.GetClientType() {
	case exported.Tendermint:
		trustedHeight, ok := ep.GetClientLatestHeight().(clienttypes.Height)
		require.True(ep.Chain.TB, ok)
		var err error
		header, err = ep.Counterparty.Chain.IBCClientHeader(ep.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("client type %s is not supported", ep.ClientConfig.GetClientType())
	}

	msg, err := clienttypes.NewMsgUpdateClient(
		ep.ClientID, header,
		ep.Chain.SenderAccount.GetAddress().String(),
	)
	require.NoError(ep.Chain.TB, err)

	return ep.Chain.sendMsgs(msg)
}

// FreezeClient freezes the IBC client associated with the ep.
func (ep *Endpoint) FreezeClient() {
	clientState := ep.Chain.GetClientState(ep.ClientID)
	tmClientState, ok := clientState.(*ibctm.ClientState)
	require.True(ep.Chain.TB, ok)

	tmClientState.FrozenHeight = clienttypes.NewHeight(0, 1)
	ep.Chain.App.GetIBCKeeper().ClientKeeper.SetClientState(ep.Chain.GetContext(), ep.ClientID, tmClientState)
}

// UpgradeChain will upgrade a chain's chainID to the next revision number.
// It will also update the counterparty client.
// TODO: implement actual upgrade chain functionality via scheduling an upgrade
// and upgrading the client via MsgUpgradeClient
// see reference https://github.com/cosmos/ibc-go/pull/1169
func (ep *Endpoint) UpgradeChain() error {
	if strings.TrimSpace(ep.Counterparty.ClientID) == "" {
		return errors.New("cannot upgrade chain if there is no counterparty client")
	}

	clientState := ep.Counterparty.GetClientState()
	tmClientState, ok := clientState.(*ibctm.ClientState)
	require.True(ep.Chain.TB, ok)

	// increment revision number in chainID
	oldChainID := tmClientState.ChainId
	if !clienttypes.IsRevisionFormat(oldChainID) {
		return fmt.Errorf("cannot upgrade chain which is not of revision format: %s", oldChainID)
	}

	revisionNumber := clienttypes.ParseChainID(oldChainID)
	newChainID, err := clienttypes.SetRevisionNumber(oldChainID, revisionNumber+1)
	if err != nil {
		return err
	}

	// update chain
	baseapp.SetChainID(newChainID)(ep.Chain.App.GetBaseApp())
	ep.Chain.ChainID = newChainID
	ep.Chain.ProposedHeader.ChainID = newChainID
	ep.Chain.NextBlock() // commit changes

	// update counterparty client manually
	tmClientState.ChainId = newChainID
	tmClientState.LatestHeight = clienttypes.NewHeight(revisionNumber+1, tmClientState.LatestHeight.GetRevisionHeight()+1)

	ep.Counterparty.SetClientState(clientState)

	tmConsensusState := &ibctm.ConsensusState{
		Timestamp:          ep.Chain.LatestCommittedHeader.GetTime(),
		Root:               commitmenttypes.NewMerkleRoot(ep.Chain.LatestCommittedHeader.Header.GetAppHash()),
		NextValidatorsHash: ep.Chain.LatestCommittedHeader.Header.NextValidatorsHash,
	}

	latestHeight := ep.Counterparty.GetClientLatestHeight()

	ep.Counterparty.SetConsensusState(tmConsensusState, latestHeight)

	// ensure the next update isn't identical to the one set in state
	ep.Chain.Coordinator.IncrementTime()
	ep.Chain.NextBlock()

	return ep.Counterparty.UpdateClient()
}

// ConnOpenInit will construct and execute a MsgConnectionOpenInit on the associated ep.
func (ep *Endpoint) ConnOpenInit() error {
	msg := connectiontypes.NewMsgConnectionOpenInit(
		ep.ClientID,
		ep.Counterparty.ClientID,
		ep.Counterparty.Chain.GetPrefix(), DefaultOpenInitVersion, ep.ConnectionConfig.DelayPeriod,
		ep.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := ep.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	ep.ConnectionID, err = ParseConnectionIDFromEvents(res.Events)
	require.NoError(ep.Chain.TB, err)

	return nil
}

// ConnOpenTry will construct and execute a MsgConnectionOpenTry on the associated ep.
func (ep *Endpoint) ConnOpenTry() error {
	err := ep.UpdateClient()
	require.NoError(ep.Chain.TB, err)

	initProof, proofHeight := ep.QueryConnectionHandshakeProof()

	msg := connectiontypes.NewMsgConnectionOpenTry(
		ep.ClientID, ep.Counterparty.ConnectionID, ep.Counterparty.ClientID,
		ep.Counterparty.Chain.GetPrefix(), []*connectiontypes.Version{ConnectionVersion},
		ep.ConnectionConfig.DelayPeriod, initProof, proofHeight,
		ep.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := ep.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	if ep.ConnectionID == "" {
		ep.ConnectionID, err = ParseConnectionIDFromEvents(res.Events)
		require.NoError(ep.Chain.TB, err)
	}

	return nil
}

// ConnOpenAck will construct and execute a MsgConnectionOpenAck on the associated ep.
func (ep *Endpoint) ConnOpenAck() error {
	err := ep.UpdateClient()
	require.NoError(ep.Chain.TB, err)

	tryProof, proofHeight := ep.QueryConnectionHandshakeProof()

	msg := connectiontypes.NewMsgConnectionOpenAck(
		ep.ConnectionID, ep.Counterparty.ConnectionID, // testing doesn't use flexible selection
		tryProof, proofHeight, ConnectionVersion,
		ep.Chain.SenderAccount.GetAddress().String(),
	)
	return ep.Chain.sendMsgs(msg)
}

// ConnOpenConfirm will construct and execute a MsgConnectionOpenConfirm on the associated ep.
func (ep *Endpoint) ConnOpenConfirm() error {
	err := ep.UpdateClient()
	require.NoError(ep.Chain.TB, err)

	connectionKey := host.ConnectionKey(ep.Counterparty.ConnectionID)
	proof, height := ep.Counterparty.Chain.QueryProof(connectionKey)

	msg := connectiontypes.NewMsgConnectionOpenConfirm(
		ep.ConnectionID,
		proof, height,
		ep.Chain.SenderAccount.GetAddress().String(),
	)
	return ep.Chain.sendMsgs(msg)
}

// QueryConnectionHandshakeProof returns all the proofs necessary to execute OpenTry or Open Ack of
// the connection handshakes. It returns the proof of the counterparty connection and the proof height.
func (ep *Endpoint) QueryConnectionHandshakeProof() (
	[]byte, clienttypes.Height,
) {
	// query proof for the connection on the counterparty
	connectionKey := host.ConnectionKey(ep.Counterparty.ConnectionID)
	return ep.Counterparty.QueryProof(connectionKey)
}

var sequenceNumber int

// IncrementNextChannelSequence incrementes the value "nextChannelSequence" in the store,
// which is used to determine the next channel ID.
// This guarantees that we'll have always different IDs while running tests.
func (ep *Endpoint) IncrementNextChannelSequence() {
	if ep.disableUniqueChannelIDs {
		return
	}
	sequenceNumber++
	ep.Chain.App.GetIBCKeeper().ChannelKeeper.SetNextChannelSequence(ep.Chain.GetContext(), uint64(sequenceNumber))
}

// ChanOpenInit will construct and execute a MsgChannelOpenInit on the associated ep.
func (ep *Endpoint) ChanOpenInit() error {
	ep.IncrementNextChannelSequence()
	msg := channeltypes.NewMsgChannelOpenInit(
		ep.ChannelConfig.PortID,
		ep.ChannelConfig.Version, ep.ChannelConfig.Order, []string{ep.ConnectionID},
		ep.Counterparty.ChannelConfig.PortID,
		ep.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := ep.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	ep.ChannelID, err = ParseChannelIDFromEvents(res.Events)
	require.NoError(ep.Chain.TB, err)

	// update version to selected app version
	// NOTE: this update must be performed after SendMsgs()
	ep.ChannelConfig.Version = ep.GetChannel().Version
	ep.Counterparty.ChannelConfig.Version = ep.GetChannel().Version

	return nil
}

// ChanOpenTry will construct and execute a MsgChannelOpenTry on the associated ep.
func (ep *Endpoint) ChanOpenTry() error {
	ep.IncrementNextChannelSequence()
	err := ep.UpdateClient()
	require.NoError(ep.Chain.TB, err)

	channelKey := host.ChannelKey(ep.Counterparty.ChannelConfig.PortID, ep.Counterparty.ChannelID)
	proof, height := ep.Counterparty.Chain.QueryProof(channelKey)

	msg := channeltypes.NewMsgChannelOpenTry(
		ep.ChannelConfig.PortID,
		ep.ChannelConfig.Version, ep.ChannelConfig.Order, []string{ep.ConnectionID},
		ep.Counterparty.ChannelConfig.PortID, ep.Counterparty.ChannelID, ep.Counterparty.ChannelConfig.Version,
		proof, height,
		ep.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := ep.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	if ep.ChannelID == "" {
		ep.ChannelID, err = ParseChannelIDFromEvents(res.Events)
		require.NoError(ep.Chain.TB, err)
	}

	// update version to selected app version
	// NOTE: this update must be performed after the endpoint channelID is set
	ep.ChannelConfig.Version = ep.GetChannel().Version
	ep.Counterparty.ChannelConfig.Version = ep.GetChannel().Version

	return nil
}

// ChanOpenAck will construct and execute a MsgChannelOpenAck on the associated ep.
func (ep *Endpoint) ChanOpenAck() error {
	err := ep.UpdateClient()
	require.NoError(ep.Chain.TB, err)

	channelKey := host.ChannelKey(ep.Counterparty.ChannelConfig.PortID, ep.Counterparty.ChannelID)
	proof, height := ep.Counterparty.Chain.QueryProof(channelKey)

	msg := channeltypes.NewMsgChannelOpenAck(
		ep.ChannelConfig.PortID, ep.ChannelID,
		ep.Counterparty.ChannelID, ep.Counterparty.ChannelConfig.Version, // testing doesn't use flexible selection
		proof, height,
		ep.Chain.SenderAccount.GetAddress().String(),
	)

	if err = ep.Chain.sendMsgs(msg); err != nil {
		return err
	}

	ep.ChannelConfig.Version = ep.GetChannel().Version

	return nil
}

// ChanOpenConfirm will construct and execute a MsgChannelOpenConfirm on the associated ep.
func (ep *Endpoint) ChanOpenConfirm() error {
	err := ep.UpdateClient()
	require.NoError(ep.Chain.TB, err)

	channelKey := host.ChannelKey(ep.Counterparty.ChannelConfig.PortID, ep.Counterparty.ChannelID)
	proof, height := ep.Counterparty.Chain.QueryProof(channelKey)

	msg := channeltypes.NewMsgChannelOpenConfirm(
		ep.ChannelConfig.PortID, ep.ChannelID,
		proof, height,
		ep.Chain.SenderAccount.GetAddress().String(),
	)
	return ep.Chain.sendMsgs(msg)
}

// ChanCloseInit will construct and execute a MsgChannelCloseInit on the associated ep.
//
// NOTE: does not work with ibc-transfer module
func (ep *Endpoint) ChanCloseInit() error {
	msg := channeltypes.NewMsgChannelCloseInit(
		ep.ChannelConfig.PortID, ep.ChannelID,
		ep.Chain.SenderAccount.GetAddress().String(),
	)
	return ep.Chain.sendMsgs(msg)
}

// SendPacket sends a packet through the channel keeper using the associated endpoint
// The counterparty client is updated so proofs can be sent to the counterparty chain.
// The packet sequence generated for the packet to be sent is returned. An error
// is returned if one occurs.
func (ep *Endpoint) SendPacket(
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (uint64, error) {
	// no need to send message, acting as a module
	sequence, err := ep.Chain.App.GetIBCKeeper().ChannelKeeper.SendPacket(ep.Chain.GetContext(), ep.ChannelConfig.PortID, ep.ChannelID, timeoutHeight, timeoutTimestamp, data)
	if err != nil {
		return 0, err
	}

	// commit changes since no message was sent
	ep.Chain.Coordinator.CommitBlock(ep.Chain)

	err = ep.Counterparty.UpdateClient()
	if err != nil {
		return 0, err
	}

	return sequence, nil
}

// RecvPacket receives a packet on the associated ep.
// The counterparty client is updated.
func (ep *Endpoint) RecvPacket(packet channeltypes.Packet) error {
	_, err := ep.RecvPacketWithResult(packet)
	if err != nil {
		return err
	}

	return nil
}

// RecvPacketWithResult receives a packet on the associated endpoint and the result
// of the transaction is returned. The counterparty client is updated.
func (ep *Endpoint) RecvPacketWithResult(packet channeltypes.Packet) (*abci.ExecTxResult, error) {
	// get proof of packet commitment on source
	packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	proof, proofHeight := ep.Counterparty.Chain.QueryProof(packetKey)

	recvMsg := channeltypes.NewMsgRecvPacket(packet, proof, proofHeight, ep.Chain.SenderAccount.GetAddress().String())

	// receive on counterparty and update source client
	res, err := ep.Chain.SendMsgs(recvMsg)
	if err != nil {
		return nil, err
	}

	if err := ep.Counterparty.UpdateClient(); err != nil {
		return nil, err
	}

	return res, nil
}

// WriteAcknowledgement writes an acknowledgement on the channel associated with the ep.
// The counterparty client is updated.
func (ep *Endpoint) WriteAcknowledgement(ack exported.Acknowledgement, packet exported.PacketI) error {
	// no need to send message, acting as a handler
	err := ep.Chain.App.GetIBCKeeper().ChannelKeeper.WriteAcknowledgement(ep.Chain.GetContext(), packet, ack)
	if err != nil {
		return err
	}

	// commit changes since no message was sent
	ep.Chain.Coordinator.CommitBlock(ep.Chain)

	return ep.Counterparty.UpdateClient()
}

// AcknowledgePacket sends a MsgAcknowledgement to the channel associated with the ep.
func (ep *Endpoint) AcknowledgePacket(packet channeltypes.Packet, ack []byte) error {
	// get proof of acknowledgement on counterparty
	packetKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	proof, proofHeight := ep.Counterparty.QueryProof(packetKey)

	ackMsg := channeltypes.NewMsgAcknowledgement(packet, ack, proof, proofHeight, ep.Chain.SenderAccount.GetAddress().String())

	return ep.Chain.sendMsgs(ackMsg)
}

// AcknowledgePacketWithResult sends a MsgAcknowledgement to the channel associated with the endpoint and returns the result.
func (ep *Endpoint) AcknowledgePacketWithResult(packet channeltypes.Packet, ack []byte) (*abci.ExecTxResult, error) {
	// get proof of acknowledgement on counterparty
	packetKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	proof, proofHeight := ep.Counterparty.QueryProof(packetKey)

	ackMsg := channeltypes.NewMsgAcknowledgement(packet, ack, proof, proofHeight, ep.Chain.SenderAccount.GetAddress().String())

	return ep.Chain.SendMsgs(ackMsg)
}

// TimeoutPacketWithResult sends a MsgTimeout to the channel associated with the ep.
func (ep *Endpoint) TimeoutPacketWithResult(packet channeltypes.Packet) (*abci.ExecTxResult, error) {
	// get proof for timeout based on channel order
	var packetKey []byte

	switch ep.ChannelConfig.Order {
	case channeltypes.ORDERED:
		packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
	case channeltypes.UNORDERED:
		packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	default:
		return nil, fmt.Errorf("unsupported order type %s", ep.ChannelConfig.Order)
	}

	counterparty := ep.Counterparty
	proof, proofHeight := counterparty.QueryProof(packetKey)
	nextSeqRecv, found := counterparty.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceRecv(counterparty.Chain.GetContext(), counterparty.ChannelConfig.PortID, counterparty.ChannelID)
	require.True(ep.Chain.TB, found)

	timeoutMsg := channeltypes.NewMsgTimeout(
		packet, nextSeqRecv,
		proof, proofHeight, ep.Chain.SenderAccount.GetAddress().String(),
	)

	return ep.Chain.SendMsgs(timeoutMsg)
}

// TimeoutPacket sends a MsgTimeout to the channel associated with the ep.
func (ep *Endpoint) TimeoutPacket(packet channeltypes.Packet) error {
	_, err := ep.TimeoutPacketWithResult(packet)
	return err
}

// TimeoutOnClose sends a MsgTimeoutOnClose to the channel associated with the ep.
func (ep *Endpoint) TimeoutOnClose(packet channeltypes.Packet) error {
	// get proof for timeout based on channel order
	var packetKey []byte

	switch ep.ChannelConfig.Order {
	case channeltypes.ORDERED:
		packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
	case channeltypes.UNORDERED:
		packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	default:
		return fmt.Errorf("unsupported order type %s", ep.ChannelConfig.Order)
	}

	proof, proofHeight := ep.Counterparty.QueryProof(packetKey)

	channelKey := host.ChannelKey(packet.GetDestPort(), packet.GetDestChannel())
	closedProof, _ := ep.Counterparty.QueryProof(channelKey)

	nextSeqRecv, found := ep.Counterparty.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceRecv(ep.Counterparty.Chain.GetContext(), ep.Counterparty.ChannelConfig.PortID, ep.Counterparty.ChannelID)
	require.True(ep.Chain.TB, found)

	timeoutOnCloseMsg := channeltypes.NewMsgTimeoutOnClose(
		packet, nextSeqRecv,
		proof, closedProof, proofHeight, ep.Chain.SenderAccount.GetAddress().String(),
	)

	return ep.Chain.sendMsgs(timeoutOnCloseMsg)
}

// Deprecated: usage of this function should be replaced by `UpdateChannel`
// SetChannelState sets a channel state
func (ep *Endpoint) SetChannelState(state channeltypes.State) error {
	channel := ep.GetChannel()

	channel.State = state
	ep.Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(ep.Chain.GetContext(), ep.ChannelConfig.PortID, ep.ChannelID, channel)

	ep.Chain.Coordinator.CommitBlock(ep.Chain)

	return ep.Counterparty.UpdateClient()
}

// UpdateChannel updates the channel associated with the given ep. It accepts a
// closure which takes a channel allowing the caller to modify its fields.
func (ep *Endpoint) UpdateChannel(updater func(channel *channeltypes.Channel)) {
	channel := ep.GetChannel()
	updater(&channel)
	ep.SetChannel(channel)

	ep.Chain.Coordinator.CommitBlock(ep.Chain)

	err := ep.Counterparty.UpdateClient()
	require.NoError(ep.Chain.TB, err)
}

// GetClientLatestHeight returns the latest height for the client state for this ep.
// The client state is expected to exist otherwise testing will fail.
func (ep *Endpoint) GetClientLatestHeight() exported.Height {
	return ep.Chain.GetClientLatestHeight(ep.ClientID)
}

// GetClientState retrieves the client state for this ep. The
// client state is expected to exist otherwise testing will fail.
func (ep *Endpoint) GetClientState() exported.ClientState {
	return ep.Chain.GetClientState(ep.ClientID)
}

// SetClientState sets the client state for this ep.
func (ep *Endpoint) SetClientState(clientState exported.ClientState) {
	ep.Chain.App.GetIBCKeeper().ClientKeeper.SetClientState(ep.Chain.GetContext(), ep.ClientID, clientState)
}

// GetConsensusState retrieves the Consensus State for this endpoint at the provided height.
// The consensus state is expected to exist otherwise testing will fail.
func (ep *Endpoint) GetConsensusState(height exported.Height) exported.ConsensusState {
	consensusState, found := ep.Chain.GetConsensusState(ep.ClientID, height)
	require.True(ep.Chain.TB, found)

	return consensusState
}

// SetConsensusState sets the consensus state for this ep.
func (ep *Endpoint) SetConsensusState(consensusState exported.ConsensusState, height exported.Height) {
	ep.Chain.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(ep.Chain.GetContext(), ep.ClientID, height, consensusState)
}

// GetConnection retrieves an IBC Connection for the ep. The
// connection is expected to exist otherwise testing will fail.
func (ep *Endpoint) GetConnection() connectiontypes.ConnectionEnd {
	connection, found := ep.Chain.App.GetIBCKeeper().ConnectionKeeper.GetConnection(ep.Chain.GetContext(), ep.ConnectionID)
	require.True(ep.Chain.TB, found)

	return connection
}

// SetConnection sets the connection for this ep.
func (ep *Endpoint) SetConnection(connection connectiontypes.ConnectionEnd) {
	ep.Chain.App.GetIBCKeeper().ConnectionKeeper.SetConnection(ep.Chain.GetContext(), ep.ConnectionID, connection)
}

// GetChannel retrieves an IBC Channel for the ep. The channel
// is expected to exist otherwise testing will fail.
func (ep *Endpoint) GetChannel() channeltypes.Channel {
	channel, found := ep.Chain.App.GetIBCKeeper().ChannelKeeper.GetChannel(ep.Chain.GetContext(), ep.ChannelConfig.PortID, ep.ChannelID)
	require.True(ep.Chain.TB, found)

	return channel
}

// SetChannel sets the channel for this ep.
func (ep *Endpoint) SetChannel(channel channeltypes.Channel) {
	ep.Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(ep.Chain.GetContext(), ep.ChannelConfig.PortID, ep.ChannelID, channel)
}

// QueryClientStateProof performs and abci query for a client stat associated
// with this endpoint and returns the ClientState along with the proof.
func (ep *Endpoint) QueryClientStateProof() (exported.ClientState, []byte) {
	// retrieve client state to provide proof for
	clientState := ep.GetClientState()

	clientKey := host.FullClientStateKey(ep.ClientID)
	clientProof, _ := ep.QueryProof(clientKey)

	return clientState, clientProof
}

// UpdateConnection updates the connection associated with the given ep. It accepts a
// closure which takes a connection allowing the caller to modify the connection fields.
func (ep *Endpoint) UpdateConnection(updater func(connection *connectiontypes.ConnectionEnd)) {
	connection := ep.GetConnection()
	updater(&connection)

	ep.SetConnection(connection)
}
