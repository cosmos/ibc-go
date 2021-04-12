package ibctesting

import (
	"fmt"

	//	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
	ibctmtypes "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
)

// Endpoint
type Endpoint struct {
	Chain        *TestChain
	Counterparty *Endpoint
	ClientID     string
	ConnectionID string
	ChannelID    string

	ClientType string
}

func NewEndpoint(chain *TestChain) *Endpoint {
	return &Endpoint{
		Chain:      chain,
		ClientType: exported.Tendermint,
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
		clientState, consensusState, endpoint.Chain.SenderAccount.GetAddress(),
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
		endpoint.Chain.SenderAccount.GetAddress(),
	)
	require.NoError(endpoint.Chain.t, err)

	return endpoint.Chain.sendMsgs(msg)

}

// ConnOpenInit will construct and execute a MsgConnectionOpenInit on the associated endpoint.
// TODO parse connection ID
func (endpoint *Endpoint) ConnOpenInit() error {
	msg := connectiontypes.NewMsgConnectionOpenInit(
		endpoint.ClientID,
		endpoint.Counterparty.ClientID,
		endpoint.Counterparty.Chain.GetPrefix(), DefaultOpenInitVersion, DefaultDelayPeriod,
		endpoint.Chain.SenderAccount.GetAddress(),
	)
	return endpoint.Chain.sendMsgs(msg)
}

// ConnOpenTry will construct and execute a MsgConnectionOpenTry on the associated endpoint.
// TODO handle client update
// TODO parse connection ID
func (endpoint *Endpoint) ConnOpenTry() error {
	counterpartyClient, proofClient := endpoint.Counterparty.Chain.QueryClientStateProof(endpoint.Counterparty.ClientID)

	connectionKey := host.ConnectionKey(endpoint.Counterparty.ConnectionID)
	proofInit, proofHeight := endpoint.Counterparty.Chain.QueryProof(connectionKey)

	proofConsensus, consensusHeight := endpoint.Counterparty.Chain.QueryConsensusStateProof(endpoint.Counterparty.ClientID)

	msg := connectiontypes.NewMsgConnectionOpenTry(
		"", endpoint.ClientID, // does not support handshake continuation
		endpoint.Counterparty.ConnectionID, endpoint.Counterparty.ClientID,
		counterpartyClient, endpoint.Counterparty.Chain.GetPrefix(), []*connectiontypes.Version{ConnectionVersion}, DefaultDelayPeriod,
		proofInit, proofClient, proofConsensus,
		proofHeight, consensusHeight,
		endpoint.Chain.SenderAccount.GetAddress(),
	)
	return endpoint.Chain.sendMsgs(msg)
}

// ConnOpenAck will construct and execute a MsgConnectionOpenAck on the associated endpoint.
// TODO handle client update
func (endpoint *Endpoint) ConnOpenAck() error {
	counterpartyClient, proofClient := endpoint.Counterparty.Chain.QueryClientStateProof(endpoint.Counterparty.ClientID)

	connectionKey := host.ConnectionKey(endpoint.Counterparty.ConnectionID)
	proofTry, proofHeight := endpoint.Counterparty.Chain.QueryProof(connectionKey)

	proofConsensus, consensusHeight := endpoint.Counterparty.Chain.QueryConsensusStateProof(endpoint.Counterparty.ClientID)

	msg := connectiontypes.NewMsgConnectionOpenAck(
		endpoint.ConnectionID, endpoint.Counterparty.ConnectionID, counterpartyClient, // testing doesn't use flexible selection
		proofTry, proofClient, proofConsensus,
		proofHeight, consensusHeight,
		ConnectionVersion,
		endpoint.Chain.SenderAccount.GetAddress(),
	)
	return endpoint.Chain.sendMsgs(msg)
}

// ConnOpenConfirm will construct and execute a MsgConnectionOpenConfirm on the associated endpoint.
func (endpoint *Endpoint) ConnOpenConfirm() error {
	connectionKey := host.ConnectionKey(endpoint.Counterparty.ConnectionID)
	proof, height := endpoint.Counterparty.Chain.QueryProof(connectionKey)

	msg := connectiontypes.NewMsgConnectionOpenConfirm(
		endpoint.ConnectionID,
		proof, height,
		endpoint.Chain.SenderAccount.GetAddress(),
	)
	return endpoint.Chain.sendMsgs(msg)
}
