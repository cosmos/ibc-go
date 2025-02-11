package ibctesting

import (
	"errors"
	"fmt"
	"strings"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	abci "github.com/cometbft/cometbft/abci/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
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
func (endpoint *Endpoint) QueryProof(key []byte) ([]byte, clienttypes.Height) {
	// obtain the counterparty client height.
	latestCounterpartyHeight := endpoint.Counterparty.GetClientLatestHeight()
	// query proof on the counterparty using the latest height of the IBC client
	return endpoint.QueryProofAtHeight(key, latestCounterpartyHeight.GetRevisionHeight())
}

// QueryProofAtHeight queries proof associated with this endpoint using the proof height
// provided
func (endpoint *Endpoint) QueryProofAtHeight(key []byte, height uint64) ([]byte, clienttypes.Height) {
	// query proof on the counterparty using the latest height of the IBC client
	return endpoint.Chain.QueryProofAtHeight(key, int64(height))
}

// CreateClient creates an IBC client on the endpoint. It will update the
// clientID for the endpoint if the message is successfully executed.
// NOTE: a solo machine client will be created with an empty diversifier.
func (endpoint *Endpoint) CreateClient() (err error) {
	// ensure counterparty has committed state
	endpoint.Counterparty.Chain.NextBlock()

	var (
		clientState    exported.ClientState
		consensusState exported.ConsensusState
	)

	switch endpoint.ClientConfig.GetClientType() {
	case exported.Tendermint:
		tmConfig, ok := endpoint.ClientConfig.(*TendermintConfig)
		require.True(endpoint.Chain.TB, ok)

		height, ok := endpoint.Counterparty.Chain.LatestCommittedHeader.GetHeight().(clienttypes.Height)
		require.True(endpoint.Chain.TB, ok)
		clientState = ibctm.NewClientState(
			endpoint.Counterparty.Chain.ChainID, tmConfig.TrustLevel, tmConfig.TrustingPeriod, tmConfig.UnbondingPeriod, tmConfig.MaxClockDrift,
			height, commitmenttypes.GetSDKSpecs(), UpgradePath)
		consensusState = endpoint.Counterparty.Chain.LatestCommittedHeader.ConsensusState()
	case exported.Solomachine:
		// TODO
		//		solo := NewSolomachine(endpoint.Chain.TB, endpoint.Chain.Codec, clientID, "", 1)
		//		clientState = solo.ClientState()
		//		consensusState = solo.ConsensusState()
	default:
		err = fmt.Errorf("client type %s is not supported", endpoint.ClientConfig.GetClientType())
	}

	if err != nil {
		return err
	}

	msg, err := clienttypes.NewMsgCreateClient(
		clientState, consensusState, endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	require.NoError(endpoint.Chain.TB, err)

	res, err := endpoint.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	endpoint.ClientID, err = ParseClientIDFromEvents(res.Events)
	require.NoError(endpoint.Chain.TB, err)

	return nil
}

// UpdateClient updates the IBC client associated with the endpoint.
func (endpoint *Endpoint) UpdateClient() (err error) {
	// ensure counterparty has committed state
	endpoint.Chain.Coordinator.CommitBlock(endpoint.Counterparty.Chain)

	var header exported.ClientMessage

	switch endpoint.ClientConfig.GetClientType() {
	case exported.Tendermint:
		trustedHeight, ok := endpoint.GetClientLatestHeight().(clienttypes.Height)
		require.True(endpoint.Chain.TB, ok)
		header, err = endpoint.Counterparty.Chain.IBCClientHeader(endpoint.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
	default:
		err = fmt.Errorf("client type %s is not supported", endpoint.ClientConfig.GetClientType())
	}

	if err != nil {
		return err
	}

	msg, err := clienttypes.NewMsgUpdateClient(
		endpoint.ClientID, header,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	require.NoError(endpoint.Chain.TB, err)

	return endpoint.Chain.sendMsgs(msg)
}

// FreezeClient freezes the IBC client associated with the endpoint.
func (endpoint *Endpoint) FreezeClient() {
	clientState := endpoint.Chain.GetClientState(endpoint.ClientID)
	tmClientState, ok := clientState.(*ibctm.ClientState)
	require.True(endpoint.Chain.TB, ok)

	tmClientState.FrozenHeight = clienttypes.NewHeight(0, 1)
	endpoint.Chain.App.GetIBCKeeper().ClientKeeper.SetClientState(endpoint.Chain.GetContext(), endpoint.ClientID, tmClientState)
}

// UpgradeChain will upgrade a chain's chainID to the next revision number.
// It will also update the counterparty client.
// TODO: implement actual upgrade chain functionality via scheduling an upgrade
// and upgrading the client via MsgUpgradeClient
// see reference https://github.com/cosmos/ibc-go/pull/1169
func (endpoint *Endpoint) UpgradeChain() error {
	if strings.TrimSpace(endpoint.Counterparty.ClientID) == "" {
		return errors.New("cannot upgrade chain if there is no counterparty client")
	}

	clientState := endpoint.Counterparty.GetClientState()
	tmClientState, ok := clientState.(*ibctm.ClientState)
	require.True(endpoint.Chain.TB, ok)

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
	baseapp.SetChainID(newChainID)(endpoint.Chain.App.GetBaseApp())
	endpoint.Chain.ChainID = newChainID
	endpoint.Chain.ProposedHeader.ChainID = newChainID
	endpoint.Chain.NextBlock() // commit changes

	// update counterparty client manually
	tmClientState.ChainId = newChainID
	tmClientState.LatestHeight = clienttypes.NewHeight(revisionNumber+1, tmClientState.LatestHeight.GetRevisionHeight()+1)

	endpoint.Counterparty.SetClientState(clientState)

	tmConsensusState := &ibctm.ConsensusState{
		Timestamp:          endpoint.Chain.LatestCommittedHeader.GetTime(),
		Root:               commitmenttypes.NewMerkleRoot(endpoint.Chain.LatestCommittedHeader.Header.GetAppHash()),
		NextValidatorsHash: endpoint.Chain.LatestCommittedHeader.Header.NextValidatorsHash,
	}

	latestHeight := endpoint.Counterparty.GetClientLatestHeight()

	endpoint.Counterparty.SetConsensusState(tmConsensusState, latestHeight)

	// ensure the next update isn't identical to the one set in state
	endpoint.Chain.Coordinator.IncrementTime()
	endpoint.Chain.NextBlock()

	return endpoint.Counterparty.UpdateClient()
}

// ConnOpenInit will construct and execute a MsgConnectionOpenInit on the associated endpoint.
func (endpoint *Endpoint) ConnOpenInit() error {
	msg := connectiontypes.NewMsgConnectionOpenInit(
		endpoint.ClientID,
		endpoint.Counterparty.ClientID,
		endpoint.Counterparty.Chain.GetPrefix(), DefaultOpenInitVersion, endpoint.ConnectionConfig.DelayPeriod,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := endpoint.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	endpoint.ConnectionID, err = ParseConnectionIDFromEvents(res.Events)
	require.NoError(endpoint.Chain.TB, err)

	return nil
}

// ConnOpenTry will construct and execute a MsgConnectionOpenTry on the associated endpoint.
func (endpoint *Endpoint) ConnOpenTry() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.TB, err)

	initProof, proofHeight := endpoint.QueryConnectionHandshakeProof()

	msg := connectiontypes.NewMsgConnectionOpenTry(
		endpoint.ClientID, endpoint.Counterparty.ConnectionID, endpoint.Counterparty.ClientID,
		endpoint.Counterparty.Chain.GetPrefix(), []*connectiontypes.Version{ConnectionVersion},
		endpoint.ConnectionConfig.DelayPeriod, initProof, proofHeight,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := endpoint.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	if endpoint.ConnectionID == "" {
		endpoint.ConnectionID, err = ParseConnectionIDFromEvents(res.Events)
		require.NoError(endpoint.Chain.TB, err)
	}

	return nil
}

// ConnOpenAck will construct and execute a MsgConnectionOpenAck on the associated endpoint.
func (endpoint *Endpoint) ConnOpenAck() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.TB, err)

	tryProof, proofHeight := endpoint.QueryConnectionHandshakeProof()

	msg := connectiontypes.NewMsgConnectionOpenAck(
		endpoint.ConnectionID, endpoint.Counterparty.ConnectionID, // testing doesn't use flexible selection
		tryProof, proofHeight, ConnectionVersion,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	return endpoint.Chain.sendMsgs(msg)
}

// ConnOpenConfirm will construct and execute a MsgConnectionOpenConfirm on the associated endpoint.
func (endpoint *Endpoint) ConnOpenConfirm() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.TB, err)

	connectionKey := host.ConnectionKey(endpoint.Counterparty.ConnectionID)
	proof, height := endpoint.Counterparty.Chain.QueryProof(connectionKey)

	msg := connectiontypes.NewMsgConnectionOpenConfirm(
		endpoint.ConnectionID,
		proof, height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	return endpoint.Chain.sendMsgs(msg)
}

// QueryConnectionHandshakeProof returns all the proofs necessary to execute OpenTry or Open Ack of
// the connection handshakes. It returns the proof of the counterparty connection and the proof height.
func (endpoint *Endpoint) QueryConnectionHandshakeProof() (
	connectionProof []byte, proofHeight clienttypes.Height,
) {
	// query proof for the connection on the counterparty
	connectionKey := host.ConnectionKey(endpoint.Counterparty.ConnectionID)
	connectionProof, proofHeight = endpoint.Counterparty.QueryProof(connectionKey)

	return connectionProof, proofHeight
}

var sequenceNumber int

// IncrementNextChannelSequence incrementes the value "nextChannelSequence" in the store,
// which is used to determine the next channel ID.
// This guarantees that we'll have always different IDs while running tests.
func (endpoint *Endpoint) IncrementNextChannelSequence() {
	if endpoint.disableUniqueChannelIDs {
		return
	}
	sequenceNumber++
	endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.SetNextChannelSequence(endpoint.Chain.GetContext(), uint64(sequenceNumber))
}

// ChanOpenInit will construct and execute a MsgChannelOpenInit on the associated endpoint.
func (endpoint *Endpoint) ChanOpenInit() error {
	endpoint.IncrementNextChannelSequence()
	msg := channeltypes.NewMsgChannelOpenInit(
		endpoint.ChannelConfig.PortID,
		endpoint.ChannelConfig.Version, endpoint.ChannelConfig.Order, []string{endpoint.ConnectionID},
		endpoint.Counterparty.ChannelConfig.PortID,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := endpoint.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	endpoint.ChannelID, err = ParseChannelIDFromEvents(res.Events)
	require.NoError(endpoint.Chain.TB, err)

	// update version to selected app version
	// NOTE: this update must be performed after SendMsgs()
	endpoint.ChannelConfig.Version = endpoint.GetChannel().Version
	endpoint.Counterparty.ChannelConfig.Version = endpoint.GetChannel().Version

	return nil
}

// ChanOpenTry will construct and execute a MsgChannelOpenTry on the associated endpoint.
func (endpoint *Endpoint) ChanOpenTry() error {
	endpoint.IncrementNextChannelSequence()
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.TB, err)

	channelKey := host.ChannelKey(endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	proof, height := endpoint.Counterparty.Chain.QueryProof(channelKey)

	msg := channeltypes.NewMsgChannelOpenTry(
		endpoint.ChannelConfig.PortID,
		endpoint.ChannelConfig.Version, endpoint.ChannelConfig.Order, []string{endpoint.ConnectionID},
		endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID, endpoint.Counterparty.ChannelConfig.Version,
		proof, height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := endpoint.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	if endpoint.ChannelID == "" {
		endpoint.ChannelID, err = ParseChannelIDFromEvents(res.Events)
		require.NoError(endpoint.Chain.TB, err)
	}

	// update version to selected app version
	// NOTE: this update must be performed after the endpoint channelID is set
	endpoint.ChannelConfig.Version = endpoint.GetChannel().Version
	endpoint.Counterparty.ChannelConfig.Version = endpoint.GetChannel().Version

	return nil
}

// ChanOpenAck will construct and execute a MsgChannelOpenAck on the associated endpoint.
func (endpoint *Endpoint) ChanOpenAck() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.TB, err)

	channelKey := host.ChannelKey(endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	proof, height := endpoint.Counterparty.Chain.QueryProof(channelKey)

	msg := channeltypes.NewMsgChannelOpenAck(
		endpoint.ChannelConfig.PortID, endpoint.ChannelID,
		endpoint.Counterparty.ChannelID, endpoint.Counterparty.ChannelConfig.Version, // testing doesn't use flexible selection
		proof, height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)

	if err = endpoint.Chain.sendMsgs(msg); err != nil {
		return err
	}

	endpoint.ChannelConfig.Version = endpoint.GetChannel().Version

	return nil
}

// ChanOpenConfirm will construct and execute a MsgChannelOpenConfirm on the associated endpoint.
func (endpoint *Endpoint) ChanOpenConfirm() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.TB, err)

	channelKey := host.ChannelKey(endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	proof, height := endpoint.Counterparty.Chain.QueryProof(channelKey)

	msg := channeltypes.NewMsgChannelOpenConfirm(
		endpoint.ChannelConfig.PortID, endpoint.ChannelID,
		proof, height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	return endpoint.Chain.sendMsgs(msg)
}

// ChanCloseInit will construct and execute a MsgChannelCloseInit on the associated endpoint.
//
// NOTE: does not work with ibc-transfer module
func (endpoint *Endpoint) ChanCloseInit() error {
	msg := channeltypes.NewMsgChannelCloseInit(
		endpoint.ChannelConfig.PortID, endpoint.ChannelID,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	return endpoint.Chain.sendMsgs(msg)
}

// SendPacket sends a packet through the channel keeper using the associated endpoint
// The counterparty client is updated so proofs can be sent to the counterparty chain.
// The packet sequence generated for the packet to be sent is returned. An error
// is returned if one occurs.
func (endpoint *Endpoint) SendPacket(
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (uint64, error) {
	// no need to send message, acting as a module
	sequence, err := endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.SendPacket(endpoint.Chain.GetContext(), endpoint.ChannelConfig.PortID, endpoint.ChannelID, timeoutHeight, timeoutTimestamp, data)
	if err != nil {
		return 0, err
	}

	// commit changes since no message was sent
	endpoint.Chain.Coordinator.CommitBlock(endpoint.Chain)

	err = endpoint.Counterparty.UpdateClient()
	if err != nil {
		return 0, err
	}

	return sequence, nil
}

// RecvPacket receives a packet on the associated endpoint.
// The counterparty client is updated.
func (endpoint *Endpoint) RecvPacket(packet channeltypes.Packet) error {
	_, err := endpoint.RecvPacketWithResult(packet)
	if err != nil {
		return err
	}

	return nil
}

// RecvPacketWithResult receives a packet on the associated endpoint and the result
// of the transaction is returned. The counterparty client is updated.
func (endpoint *Endpoint) RecvPacketWithResult(packet channeltypes.Packet) (*abci.ExecTxResult, error) {
	// get proof of packet commitment on source
	packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	proof, proofHeight := endpoint.Counterparty.Chain.QueryProof(packetKey)

	recvMsg := channeltypes.NewMsgRecvPacket(packet, proof, proofHeight, endpoint.Chain.SenderAccount.GetAddress().String())

	// receive on counterparty and update source client
	res, err := endpoint.Chain.SendMsgs(recvMsg)
	if err != nil {
		return nil, err
	}

	if err := endpoint.Counterparty.UpdateClient(); err != nil {
		return nil, err
	}

	return res, nil
}

// WriteAcknowledgement writes an acknowledgement on the channel associated with the endpoint.
// The counterparty client is updated.
func (endpoint *Endpoint) WriteAcknowledgement(ack exported.Acknowledgement, packet exported.PacketI) error {
	// no need to send message, acting as a handler
	err := endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.WriteAcknowledgement(endpoint.Chain.GetContext(), packet, ack)
	if err != nil {
		return err
	}

	// commit changes since no message was sent
	endpoint.Chain.Coordinator.CommitBlock(endpoint.Chain)

	return endpoint.Counterparty.UpdateClient()
}

// AcknowledgePacket sends a MsgAcknowledgement to the channel associated with the endpoint.
func (endpoint *Endpoint) AcknowledgePacket(packet channeltypes.Packet, ack []byte) error {
	// get proof of acknowledgement on counterparty
	packetKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	proof, proofHeight := endpoint.Counterparty.QueryProof(packetKey)

	ackMsg := channeltypes.NewMsgAcknowledgement(packet, ack, proof, proofHeight, endpoint.Chain.SenderAccount.GetAddress().String())

	return endpoint.Chain.sendMsgs(ackMsg)
}

// AcknowledgePacketWithResult sends a MsgAcknowledgement to the channel associated with the endpoint and returns the result.
func (endpoint *Endpoint) AcknowledgePacketWithResult(packet channeltypes.Packet, ack []byte) (*abci.ExecTxResult, error) {
	// get proof of acknowledgement on counterparty
	packetKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	proof, proofHeight := endpoint.Counterparty.QueryProof(packetKey)

	ackMsg := channeltypes.NewMsgAcknowledgement(packet, ack, proof, proofHeight, endpoint.Chain.SenderAccount.GetAddress().String())

	return endpoint.Chain.SendMsgs(ackMsg)
}

// TimeoutPacketWithResult sends a MsgTimeout to the channel associated with the endpoint.
func (endpoint *Endpoint) TimeoutPacketWithResult(packet channeltypes.Packet) (*abci.ExecTxResult, error) {
	// get proof for timeout based on channel order
	var packetKey []byte

	switch endpoint.ChannelConfig.Order {
	case channeltypes.ORDERED:
		packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
	case channeltypes.UNORDERED:
		packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	default:
		return nil, fmt.Errorf("unsupported order type %s", endpoint.ChannelConfig.Order)
	}

	counterparty := endpoint.Counterparty
	proof, proofHeight := counterparty.QueryProof(packetKey)
	nextSeqRecv, found := counterparty.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceRecv(counterparty.Chain.GetContext(), counterparty.ChannelConfig.PortID, counterparty.ChannelID)
	require.True(endpoint.Chain.TB, found)

	timeoutMsg := channeltypes.NewMsgTimeout(
		packet, nextSeqRecv,
		proof, proofHeight, endpoint.Chain.SenderAccount.GetAddress().String(),
	)

	return endpoint.Chain.SendMsgs(timeoutMsg)
}

// TimeoutPacket sends a MsgTimeout to the channel associated with the endpoint.
func (endpoint *Endpoint) TimeoutPacket(packet channeltypes.Packet) error {
	_, err := endpoint.TimeoutPacketWithResult(packet)
	return err
}

// TimeoutOnClose sends a MsgTimeoutOnClose to the channel associated with the endpoint.
func (endpoint *Endpoint) TimeoutOnClose(packet channeltypes.Packet) error {
	// get proof for timeout based on channel order
	var packetKey []byte

	switch endpoint.ChannelConfig.Order {
	case channeltypes.ORDERED:
		packetKey = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
	case channeltypes.UNORDERED:
		packetKey = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	default:
		return fmt.Errorf("unsupported order type %s", endpoint.ChannelConfig.Order)
	}

	proof, proofHeight := endpoint.Counterparty.QueryProof(packetKey)

	channelKey := host.ChannelKey(packet.GetDestPort(), packet.GetDestChannel())
	closedProof, _ := endpoint.Counterparty.QueryProof(channelKey)

	nextSeqRecv, found := endpoint.Counterparty.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceRecv(endpoint.Counterparty.Chain.GetContext(), endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	require.True(endpoint.Chain.TB, found)

	timeoutOnCloseMsg := channeltypes.NewMsgTimeoutOnClose(
		packet, nextSeqRecv,
		proof, closedProof, proofHeight, endpoint.Chain.SenderAccount.GetAddress().String(),
		endpoint.Counterparty.GetChannel().UpgradeSequence,
	)

	return endpoint.Chain.sendMsgs(timeoutOnCloseMsg)
}

// QueryChannelUpgradeProof returns all the proofs necessary to execute UpgradeTry/UpgradeAck/UpgradeOpen.
// It returns the proof for the channel on the endpoint's chain, the proof for the upgrade attempt on the
// endpoint's chain, and the height at which the proof was queried.
func (endpoint *Endpoint) QueryChannelUpgradeProof() ([]byte, []byte, clienttypes.Height) {
	channelKey := host.ChannelKey(endpoint.ChannelConfig.PortID, endpoint.ChannelID)
	channelProof, height := endpoint.QueryProof(channelKey)

	upgradeKey := host.ChannelUpgradeKey(endpoint.ChannelConfig.PortID, endpoint.ChannelID)
	upgradeProof, _ := endpoint.QueryProof(upgradeKey)

	return channelProof, upgradeProof, height
}

// ChanUpgradeInit sends a MsgChannelUpgradeInit on the associated endpoint.
// A default upgrade proposal is used with overrides from the ProposedUpgrade
// in the channel config, and submitted via governance proposal
func (endpoint *Endpoint) ChanUpgradeInit() error {
	upgrade := endpoint.GetProposedUpgrade()

	// create upgrade init message via gov proposal and submit the proposal
	msg := channeltypes.NewMsgChannelUpgradeInit(
		endpoint.ChannelConfig.PortID,
		endpoint.ChannelID,
		upgrade.Fields,
		endpoint.Chain.GetSimApp().IBCKeeper.GetAuthority(),
	)

	proposal, err := govtypesv1.NewMsgSubmitProposal(
		[]sdk.Msg{msg},
		sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, govtypesv1.DefaultMinDepositTokens)),
		endpoint.Chain.SenderAccount.GetAddress().String(),
		endpoint.ChannelID,
		"upgrade-init",
		fmt.Sprintf("gov proposal for initialising channel upgrade: %s", endpoint.ChannelID),
		false,
	)
	require.NoError(endpoint.Chain.TB, err)

	var proposalID uint64
	res, err := endpoint.Chain.SendMsgs(proposal)
	if err != nil {
		return err
	}

	proposalID, err = ParseProposalIDFromEvents(res.Events)
	require.NoError(endpoint.Chain.TB, err)

	return VoteAndCheckProposalStatus(endpoint, proposalID)
}

// ChanUpgradeTry sends a MsgChannelUpgradeTry on the associated endpoint.
func (endpoint *Endpoint) ChanUpgradeTry() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.TB, err)

	upgrade := endpoint.GetProposedUpgrade()
	channelProof, upgradeProof, height := endpoint.Counterparty.QueryChannelUpgradeProof()

	counterpartyUpgrade, found := endpoint.Counterparty.Chain.App.GetIBCKeeper().ChannelKeeper.GetUpgrade(endpoint.Counterparty.Chain.GetContext(), endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	require.True(endpoint.Chain.TB, found)

	if !found {
		return fmt.Errorf("could not find upgrade for channel %s", endpoint.ChannelID)
	}

	msg := channeltypes.NewMsgChannelUpgradeTry(
		endpoint.ChannelConfig.PortID,
		endpoint.ChannelID,
		upgrade.Fields.ConnectionHops,
		counterpartyUpgrade.Fields,
		endpoint.Counterparty.GetChannel().UpgradeSequence,
		channelProof,
		upgradeProof,
		height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)

	return endpoint.Chain.sendMsgs(msg)
}

// ChanUpgradeAck sends a MsgChannelUpgradeAck to the associated endpoint.
func (endpoint *Endpoint) ChanUpgradeAck() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.TB, err)

	channelProof, upgradeProof, height := endpoint.Counterparty.QueryChannelUpgradeProof()

	counterpartyUpgrade, found := endpoint.Counterparty.Chain.App.GetIBCKeeper().ChannelKeeper.GetUpgrade(endpoint.Counterparty.Chain.GetContext(), endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	require.True(endpoint.Chain.TB, found)

	msg := channeltypes.NewMsgChannelUpgradeAck(
		endpoint.ChannelConfig.PortID,
		endpoint.ChannelID,
		counterpartyUpgrade,
		channelProof,
		upgradeProof,
		height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)

	return endpoint.Chain.sendMsgs(msg)
}

// ChanUpgradeConfirm sends a MsgChannelUpgradeConfirm to the associated endpoint.
func (endpoint *Endpoint) ChanUpgradeConfirm() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.TB, err)

	channelProof, upgradeProof, height := endpoint.Counterparty.QueryChannelUpgradeProof()

	counterpartyUpgrade, found := endpoint.Counterparty.Chain.App.GetIBCKeeper().ChannelKeeper.GetUpgrade(endpoint.Counterparty.Chain.GetContext(), endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	require.True(endpoint.Chain.TB, found)

	msg := channeltypes.NewMsgChannelUpgradeConfirm(
		endpoint.ChannelConfig.PortID,
		endpoint.ChannelID,
		endpoint.Counterparty.GetChannel().State,
		counterpartyUpgrade,
		channelProof,
		upgradeProof,
		height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)

	return endpoint.Chain.sendMsgs(msg)
}

// ChanUpgradeOpen sends a MsgChannelUpgradeOpen to the associated endpoint.
func (endpoint *Endpoint) ChanUpgradeOpen() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.TB, err)

	channelKey := host.ChannelKey(endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	channelProof, height := endpoint.Counterparty.QueryProof(channelKey)

	msg := channeltypes.NewMsgChannelUpgradeOpen(
		endpoint.ChannelConfig.PortID,
		endpoint.ChannelID,
		endpoint.Counterparty.GetChannel().State,
		endpoint.Counterparty.GetChannel().UpgradeSequence,
		channelProof,
		height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)

	return endpoint.Chain.sendMsgs(msg)
}

// ChanUpgradeTimeout sends a MsgChannelUpgradeTimeout to the associated endpoint.
func (endpoint *Endpoint) ChanUpgradeTimeout() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.TB, err)

	channelKey := host.ChannelKey(endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	channelProof, height := endpoint.Counterparty.Chain.QueryProof(channelKey)

	msg := channeltypes.NewMsgChannelUpgradeTimeout(
		endpoint.ChannelConfig.PortID,
		endpoint.ChannelID,
		endpoint.Counterparty.GetChannel(),
		channelProof,
		height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)

	return endpoint.Chain.sendMsgs(msg)
}

// ChanUpgradeCancel sends a MsgChannelUpgradeCancel to the associated endpoint.
func (endpoint *Endpoint) ChanUpgradeCancel() error {
	err := endpoint.UpdateClient()
	require.NoError(endpoint.Chain.TB, err)

	errorReceiptKey := host.ChannelUpgradeErrorKey(endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	proofErrorReceipt, height := endpoint.Counterparty.Chain.QueryProof(errorReceiptKey)

	errorReceipt, found := endpoint.Counterparty.Chain.App.GetIBCKeeper().ChannelKeeper.GetUpgradeErrorReceipt(endpoint.Counterparty.Chain.GetContext(), endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	require.True(endpoint.Chain.TB, found)

	msg := channeltypes.NewMsgChannelUpgradeCancel(
		endpoint.ChannelConfig.PortID,
		endpoint.ChannelID,
		errorReceipt,
		proofErrorReceipt,
		height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)

	return endpoint.Chain.sendMsgs(msg)
}

// Deprecated: usage of this function should be replaced by `UpdateChannel`
// SetChannelState sets a channel state
func (endpoint *Endpoint) SetChannelState(state channeltypes.State) error {
	channel := endpoint.GetChannel()

	channel.State = state
	endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(endpoint.Chain.GetContext(), endpoint.ChannelConfig.PortID, endpoint.ChannelID, channel)

	endpoint.Chain.Coordinator.CommitBlock(endpoint.Chain)

	return endpoint.Counterparty.UpdateClient()
}

// UpdateChannel updates the channel associated with the given endpoint. It accepts a
// closure which takes a channel allowing the caller to modify its fields.
func (endpoint *Endpoint) UpdateChannel(updater func(channel *channeltypes.Channel)) {
	channel := endpoint.GetChannel()
	updater(&channel)
	endpoint.SetChannel(channel)

	endpoint.Chain.Coordinator.CommitBlock(endpoint.Chain)

	err := endpoint.Counterparty.UpdateClient()
	require.NoError(endpoint.Chain.TB, err)
}

// GetClientLatestHeight returns the latest height for the client state for this endpoint.
// The client state is expected to exist otherwise testing will fail.
func (endpoint *Endpoint) GetClientLatestHeight() exported.Height {
	return endpoint.Chain.GetClientLatestHeight(endpoint.ClientID)
}

// GetClientState retrieves the client state for this endpoint. The
// client state is expected to exist otherwise testing will fail.
func (endpoint *Endpoint) GetClientState() exported.ClientState {
	return endpoint.Chain.GetClientState(endpoint.ClientID)
}

// SetClientState sets the client state for this endpoint.
func (endpoint *Endpoint) SetClientState(clientState exported.ClientState) {
	endpoint.Chain.App.GetIBCKeeper().ClientKeeper.SetClientState(endpoint.Chain.GetContext(), endpoint.ClientID, clientState)
}

// GetConsensusState retrieves the Consensus State for this endpoint at the provided height.
// The consensus state is expected to exist otherwise testing will fail.
func (endpoint *Endpoint) GetConsensusState(height exported.Height) exported.ConsensusState {
	consensusState, found := endpoint.Chain.GetConsensusState(endpoint.ClientID, height)
	require.True(endpoint.Chain.TB, found)

	return consensusState
}

// SetConsensusState sets the consensus state for this endpoint.
func (endpoint *Endpoint) SetConsensusState(consensusState exported.ConsensusState, height exported.Height) {
	endpoint.Chain.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(endpoint.Chain.GetContext(), endpoint.ClientID, height, consensusState)
}

// GetConnection retrieves an IBC Connection for the endpoint. The
// connection is expected to exist otherwise testing will fail.
func (endpoint *Endpoint) GetConnection() connectiontypes.ConnectionEnd {
	connection, found := endpoint.Chain.App.GetIBCKeeper().ConnectionKeeper.GetConnection(endpoint.Chain.GetContext(), endpoint.ConnectionID)
	require.True(endpoint.Chain.TB, found)

	return connection
}

// SetConnection sets the connection for this endpoint.
func (endpoint *Endpoint) SetConnection(connection connectiontypes.ConnectionEnd) {
	endpoint.Chain.App.GetIBCKeeper().ConnectionKeeper.SetConnection(endpoint.Chain.GetContext(), endpoint.ConnectionID, connection)
}

// GetChannel retrieves an IBC Channel for the endpoint. The channel
// is expected to exist otherwise testing will fail.
func (endpoint *Endpoint) GetChannel() channeltypes.Channel {
	channel, found := endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.GetChannel(endpoint.Chain.GetContext(), endpoint.ChannelConfig.PortID, endpoint.ChannelID)
	require.True(endpoint.Chain.TB, found)

	return channel
}

// SetChannel sets the channel for this endpoint.
func (endpoint *Endpoint) SetChannel(channel channeltypes.Channel) {
	endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(endpoint.Chain.GetContext(), endpoint.ChannelConfig.PortID, endpoint.ChannelID, channel)
}

// GetChannelUpgrade retrieves an IBC Channel Upgrade for the endpoint. The upgrade
// is expected to exist otherwise testing will fail.
func (endpoint *Endpoint) GetChannelUpgrade() channeltypes.Upgrade {
	upgrade, found := endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.GetUpgrade(endpoint.Chain.GetContext(), endpoint.ChannelConfig.PortID, endpoint.ChannelID)
	require.True(endpoint.Chain.TB, found)

	return upgrade
}

// SetChannelUpgrade sets the channel upgrade for this endpoint.
func (endpoint *Endpoint) SetChannelUpgrade(upgrade channeltypes.Upgrade) {
	endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.SetUpgrade(endpoint.Chain.GetContext(), endpoint.ChannelConfig.PortID, endpoint.ChannelID, upgrade)
}

// SetChannelCounterpartyUpgrade sets the channel counterparty upgrade for this endpoint.
func (endpoint *Endpoint) SetChannelCounterpartyUpgrade(upgrade channeltypes.Upgrade) {
	endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.SetCounterpartyUpgrade(endpoint.Chain.GetContext(), endpoint.ChannelConfig.PortID, endpoint.ChannelID, upgrade)
}

// QueryClientStateProof performs and abci query for a client stat associated
// with this endpoint and returns the ClientState along with the proof.
func (endpoint *Endpoint) QueryClientStateProof() (exported.ClientState, []byte) {
	// retrieve client state to provide proof for
	clientState := endpoint.GetClientState()

	clientKey := host.FullClientStateKey(endpoint.ClientID)
	clientProof, _ := endpoint.QueryProof(clientKey)

	return clientState, clientProof
}

// GetProposedUpgrade returns a valid upgrade which can be used for UpgradeInit and UpgradeTry.
// By default, the endpoint's existing channel fields will be used for the upgrade fields and
// a sane default timeout will be used by querying the counterparty's latest height.
// If any non-empty values are specified in the ChannelConfig's ProposedUpgrade,
// those values will be used in the returned upgrade.
func (endpoint *Endpoint) GetProposedUpgrade() channeltypes.Upgrade {
	// create a default upgrade
	upgrade := channeltypes.Upgrade{
		Fields: channeltypes.UpgradeFields{
			Ordering:       endpoint.ChannelConfig.Order,
			ConnectionHops: []string{endpoint.ConnectionID},
			Version:        endpoint.ChannelConfig.Version,
		},
		Timeout:          channeltypes.NewTimeout(endpoint.Counterparty.Chain.GetTimeoutHeight(), 0),
		NextSequenceSend: 0,
	}

	override := endpoint.ChannelConfig.ProposedUpgrade
	if override.Timeout.IsValid() {
		upgrade.Timeout = override.Timeout
	}

	if override.Fields.Ordering != channeltypes.NONE {
		upgrade.Fields.Ordering = override.Fields.Ordering
	}

	if override.Fields.Version != "" {
		upgrade.Fields.Version = override.Fields.Version
	}

	if len(override.Fields.ConnectionHops) != 0 {
		upgrade.Fields.ConnectionHops = override.Fields.ConnectionHops
	}

	return upgrade
}

// UpdateConnection updates the connection associated with the given endpoint. It accepts a
// closure which takes a connection allowing the caller to modify the connection fields.
func (endpoint *Endpoint) UpdateConnection(updater func(connection *connectiontypes.ConnectionEnd)) {
	connection := endpoint.GetConnection()
	updater(&connection)

	endpoint.SetConnection(connection)
}
