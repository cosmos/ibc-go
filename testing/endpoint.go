package ibctesting

import (
	"fmt"

	//	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
	ibctmtypes "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
	"github.com/cosmos/ibc-go/testing/mock"
)

// Endpoint
type Endpoint struct {
	Chain        *TestChain
	Counterparty *Endpoint
	ClientID     string
	ConnectionID string
	ChannelID    string
	PortID       string

	ClientType        string
	ChannelOrder      channeltypes.Order
	ConnectionVersion *connectiontypes.Version
	ChannelVersion    string
}

func NewEndpoint(chain *TestChain) *Endpoint {
	return &Endpoint{
		Chain:             chain,
		PortID:            mock.ModuleName,
		ClientType:        exported.Tendermint,
		ChannelOrder:      channeltypes.UNORDERED,
		ConnectionVersion: ConnectionVersion,
		ChannelVersion:    DefaultChannelVersion,
	}
}

// CreateClient creates an IBC client on the endpoint. It will update the
// clientID for the endpoint if the message is successfully executed.
// NOTE: a solo machine client will be created with an empty diversifier.
func (endpoint *Endpoint) CreateClient() (err error) {
	// TODO: remove?
	endpoint.Chain.Coordinator.CommitBlock(endpoint.Chain, endpoint.Counterparty.Chain)

	var (
		clientState    exported.ClientState
		consensusState exported.ConsensusState
	)

	switch endpoint.ClientType {
	case exported.Tendermint:
		//err = endpoint.Chain.CreateTMClient(counterparty, clientID)
		height := endpoint.Counterparty.Chain.LastHeader.GetHeight().(clienttypes.Height)
		clientState = ibctmtypes.NewClientState(
			endpoint.Counterparty.Chain.ChainID, DefaultTrustLevel, TrustingPeriod, UnbondingPeriod, MaxClockDrift,
			height, commitmenttypes.GetSDKSpecs(), UpgradePath, false, false,
		)
		consensusState = endpoint.Counterparty.Chain.LastHeader.ConsensusState()
	case exported.Solomachine:
		// TODO
		//		solo := NewSolomachine(chain.t, endpoint.Chain.Codec, clientID, "", 1)
		//		clientState = solo.ClientState()
		//		consensusState = solo.ConsensusState()

	default:
		err = fmt.Errorf("client type %s is not supported", endpoint.ClientType)
	}

	if err != nil {
		return err
	}

	msg, err := clienttypes.NewMsgCreateClient(
		clientState, consensusState, endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	require.NoError(endpoint.Chain.t, err)

	res, err := endpoint.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	endpoint.ClientID, err = ParseClientIDFromEvents(res.GetEvents())
	require.NoError(endpoint.Chain.t, err)

	return nil
}

// UpdateClient updates the IBC client associated with the endpoint.
func (endpoint *Endpoint) UpdateClient() (err error) {
	// TODO: remove?
	endpoint.Chain.Coordinator.CommitBlock(endpoint.Chain, endpoint.Counterparty.Chain)
	var (
		header exported.Header
	)

	switch endpoint.ClientType {
	case exported.Tendermint:
		header, err = endpoint.Chain.ConstructUpdateTMClientHeader(endpoint.Counterparty.Chain, endpoint.ClientID)

	default:
		err = fmt.Errorf("client type %s is not supported", endpoint.ClientType)
	}

	if err != nil {
		return err
	}

	msg, err := clienttypes.NewMsgUpdateClient(
		endpoint.ClientID, header,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	require.NoError(endpoint.Chain.t, err)

	return endpoint.Chain.sendMsgs(msg)

}

// ConnOpenInit will construct and execute a MsgConnectionOpenInit on the associated endpoint.
func (endpoint *Endpoint) ConnOpenInit() error {
	msg := connectiontypes.NewMsgConnectionOpenInit(
		endpoint.ClientID,
		endpoint.Counterparty.ClientID,
		endpoint.Counterparty.Chain.GetPrefix(), DefaultOpenInitVersion, DefaultDelayPeriod,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := endpoint.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	endpoint.ConnectionID, err = ParseConnectionIDFromEvents(res.GetEvents())
	require.NoError(endpoint.Chain.t, err)

	return nil
}

// ConnOpenTry will construct and execute a MsgConnectionOpenTry on the associated endpoint.
func (endpoint *Endpoint) ConnOpenTry() error {
	endpoint.UpdateClient()

	counterpartyClient, proofClient, proofConsensus, consensusHeight, proofInit, proofHeight := endpoint.QueryConnectionHandshakeProof()

	msg := connectiontypes.NewMsgConnectionOpenTry(
		"", endpoint.ClientID, // does not support handshake continuation
		endpoint.Counterparty.ConnectionID, endpoint.Counterparty.ClientID,
		counterpartyClient, endpoint.Counterparty.Chain.GetPrefix(), []*connectiontypes.Version{ConnectionVersion}, DefaultDelayPeriod,
		proofInit, proofClient, proofConsensus,
		proofHeight, consensusHeight,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := endpoint.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	if endpoint.ConnectionID == "" {
		endpoint.ConnectionID, err = ParseConnectionIDFromEvents(res.GetEvents())
		require.NoError(endpoint.Chain.t, err)
	}

	return nil
}

// ConnOpenAck will construct and execute a MsgConnectionOpenAck on the associated endpoint.
func (endpoint *Endpoint) ConnOpenAck() error {
	endpoint.UpdateClient()

	counterpartyClient, proofClient, proofConsensus, consensusHeight, proofTry, proofHeight := endpoint.QueryConnectionHandshakeProof()

	msg := connectiontypes.NewMsgConnectionOpenAck(
		endpoint.ConnectionID, endpoint.Counterparty.ConnectionID, counterpartyClient, // testing doesn't use flexible selection
		proofTry, proofClient, proofConsensus,
		proofHeight, consensusHeight,
		ConnectionVersion,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	return endpoint.Chain.sendMsgs(msg)
}

// ConnOpenConfirm will construct and execute a MsgConnectionOpenConfirm on the associated endpoint.
func (endpoint *Endpoint) ConnOpenConfirm() error {
	endpoint.UpdateClient()

	connectionKey := host.ConnectionKey(endpoint.Counterparty.ConnectionID)
	proof, height := endpoint.Counterparty.Chain.QueryProof(connectionKey)

	msg := connectiontypes.NewMsgConnectionOpenConfirm(
		endpoint.ConnectionID,
		proof, height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	return endpoint.Chain.sendMsgs(msg)
}

// ChanOpenInit will construct and execute a MsgChannelOpenInit on the associated endpoint.
func (endpoint *Endpoint) ChanOpenInit() error {
	msg := channeltypes.NewMsgChannelOpenInit(
		endpoint.PortID,
		endpoint.ChannelVersion, endpoint.ChannelOrder, []string{endpoint.ConnectionID},
		endpoint.Counterparty.PortID,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := endpoint.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	endpoint.ChannelID, err = ParseChannelIDFromEvents(res.GetEvents())
	require.NoError(endpoint.Chain.t, err)

	return nil
}

// ChanOpenTry will construct and execute a MsgChannelOpenTry on the associated endpoint.
func (endpoint *Endpoint) ChanOpenTry() error {
	endpoint.UpdateClient()

	channelKey := host.ChannelKey(endpoint.Counterparty.PortID, endpoint.Counterparty.ChannelID)
	proof, height := endpoint.Counterparty.Chain.QueryProof(channelKey)

	msg := channeltypes.NewMsgChannelOpenTry(
		endpoint.PortID, "", // does not support handshake continuation
		endpoint.ChannelVersion, endpoint.ChannelOrder, []string{endpoint.ConnectionID},
		endpoint.Counterparty.PortID, endpoint.Counterparty.ChannelID, endpoint.Counterparty.ChannelVersion,
		proof, height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := endpoint.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	if endpoint.ChannelID == "" {
		endpoint.ChannelID, err = ParseChannelIDFromEvents(res.GetEvents())
		require.NoError(endpoint.Chain.t, err)
	}

	return nil
}

// ChanOpenAck will construct and execute a MsgChannelOpenAck on the associated endpoint.
func (endpoint *Endpoint) ChanOpenAck() error {
	endpoint.UpdateClient()

	channelKey := host.ChannelKey(endpoint.Counterparty.PortID, endpoint.Counterparty.ChannelID)
	proof, height := endpoint.Counterparty.Chain.QueryProof(channelKey)

	msg := channeltypes.NewMsgChannelOpenAck(
		endpoint.PortID, endpoint.ChannelID,
		endpoint.Counterparty.ChannelID, endpoint.Counterparty.ChannelVersion, // testing doesn't use flexible selection
		proof, height,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	return endpoint.Chain.sendMsgs(msg)
}

// ChanOpenConfirm will construct and execute a MsgChannelOpenConfirm on the associated endpoint.
func (endpoint *Endpoint) ChanOpenConfirm() error {
	endpoint.UpdateClient()

	channelKey := host.ChannelKey(endpoint.Counterparty.PortID, endpoint.Counterparty.ChannelID)
	proof, height := endpoint.Counterparty.Chain.QueryProof(channelKey)

	msg := channeltypes.NewMsgChannelOpenConfirm(
		endpoint.PortID, endpoint.ChannelID,
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
		endpoint.PortID, endpoint.ChannelID,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	return endpoint.Chain.sendMsgs(msg)
}

// QueryProof queries proof associated with this endpoint using the lastest client state
// height on the counterparty chain.
func (endpoint *Endpoint) QueryProof(key []byte) ([]byte, clienttypes.Height) {
	// obtain the counterparty client representing the chain associated with the endpoint
	clientState := endpoint.Counterparty.Chain.GetClientState(endpoint.ClientID)

	// query proof on the counterparty using the latest height of the IBC client
	return endpoint.QueryProofAtHeight(key, clientState.GetLatestHeight().GetRevisionHeight())
}

// QueryProofAtHeight queries proof associated with this endpoint using the proof height
// providied
func (endpoint *Endpoint) QueryProofAtHeight(key []byte, height uint64) ([]byte, clienttypes.Height) {
	// query proof on the counterparty using the latest height of the IBC client
	return endpoint.Chain.QueryProofAtHeight(key, int64(height))
}

// QueryConnectionHandshakeProof returns all the proofs necessary to execute OpenTry or Open Ack of
// the connection handshakes. It returns the counterparty client state, proof of the counterparty
// client state, proof of the counterparty consensus state, the consensus state height, proof of
// the counterparty connection, and the proof height for all the proofs returned.
func (endpoint *Endpoint) QueryConnectionHandshakeProof() (
	clientState exported.ClientState, proofClient,
	proofConsensus []byte, consensusHeight clienttypes.Height,
	proofConnection []byte, proofHeight clienttypes.Height,
) {
	// obtain the client state on the counterparty chain
	clientState = endpoint.Counterparty.Chain.GetClientState(endpoint.Counterparty.ClientID)

	// query proof for the client state on the counterparty
	clientKey := host.FullClientStateKey(endpoint.Counterparty.ClientID)
	proofClient, proofHeight = endpoint.Counterparty.QueryProof(clientKey)

	consensusHeight = clientState.GetLatestHeight().(clienttypes.Height)

	// query proof for the consensus state on the counterparty
	consensusKey := host.FullConsensusStateKey(endpoint.Counterparty.ClientID, consensusHeight)
	proofConsensus, _ = endpoint.Counterparty.QueryProofAtHeight(consensusKey, proofHeight.GetRevisionHeight())

	// query proof for the connection on the counterparty
	connectionKey := host.ConnectionKey(endpoint.Counterparty.ConnectionID)
	proofConnection, _ = endpoint.Counterparty.QueryProofAtHeight(connectionKey, proofHeight.GetRevisionHeight())

	return
}
