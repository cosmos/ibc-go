package ibctesting

import (
	"bytes"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

// Path contains two endpoints representing two chains connected over IBC
type Path struct {
	EndpointA *Endpoint
	EndpointB *Endpoint
}

// NewPath constructs an endpoint for each chain using the default values
// for the endpoints. Each endpoint is updated to have a pointer to the
// counterparty endpoint.
func NewPath(chainA, chainB *TestChain) *Path {
	endpointA := NewDefaultEndpoint(chainA)
	endpointB := NewDefaultEndpoint(chainB)

	endpointA.Counterparty = endpointB
	endpointB.Counterparty = endpointA

	return &Path{
		EndpointA: endpointA,
		EndpointB: endpointB,
	}
}

// NewTransferPath constructs a new path between each chain suitable for use with
// the transfer module.
func NewTransferPath(chainA, chainB *TestChain) *Path {
	path := NewPath(chainA, chainB)
	path.EndpointA.ChannelConfig.PortID = TransferPort
	path.EndpointB.ChannelConfig.PortID = TransferPort
	path.EndpointA.ChannelConfig.Version = transfertypes.Version
	path.EndpointB.ChannelConfig.Version = transfertypes.Version

	return path
}

// SetChannelOrdered sets the channel order for both endpoints to ORDERED.
func (path *Path) SetChannelOrdered() {
	path.EndpointA.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointB.ChannelConfig.Order = channeltypes.ORDERED
}

// RelayPacket attempts to relay the packet first on EndpointA and then on EndpointB
// if EndpointA does not contain a packet commitment for that packet. An error is returned
// if a relay step fails or the packet commitment does not exist on either endpoint.
func (path *Path) RelayPacket(packet channeltypes.Packet) error {
	_, _, err := path.RelayPacketWithResults(packet)
	return err
}

// RelayPacketWithResults attempts to relay the packet first on EndpointA and then on EndpointB
// if EndpointA does not contain a packet commitment for that packet. The function returns:
// - The result of the packet receive transaction.
// - The acknowledgement written on the receiving chain.
// - An error if a relay step fails or the packet commitment does not exist on either endpoint.
func (path *Path) RelayPacketWithResults(packet channeltypes.Packet) (*abci.ExecTxResult, []byte, error) {
	pc := path.EndpointA.Chain.App.GetIBCKeeper().ChannelKeeper.GetPacketCommitment(path.EndpointA.Chain.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	if bytes.Equal(pc, channeltypes.CommitPacket(path.EndpointA.Chain.App.AppCodec(), packet)) {
		// packet found, relay from A to B
		if err := path.EndpointB.UpdateClient(); err != nil {
			return nil, nil, err
		}

		res, err := path.EndpointB.RecvPacketWithResult(packet)
		if err != nil {
			return nil, nil, err
		}

		ack, err := ParseAckFromEvents(res.Events)
		if err != nil {
			return nil, nil, err
		}

		if err := path.EndpointA.AcknowledgePacket(packet, ack); err != nil {
			return nil, nil, err
		}

		return res, ack, nil
	}

	pc = path.EndpointB.Chain.App.GetIBCKeeper().ChannelKeeper.GetPacketCommitment(path.EndpointB.Chain.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	if bytes.Equal(pc, channeltypes.CommitPacket(path.EndpointB.Chain.App.AppCodec(), packet)) {

		// packet found, relay B to A
		if err := path.EndpointA.UpdateClient(); err != nil {
			return nil, nil, err
		}

		res, err := path.EndpointA.RecvPacketWithResult(packet)
		if err != nil {
			return nil, nil, err
		}

		ack, err := ParseAckFromEvents(res.Events)
		if err != nil {
			return nil, nil, err
		}

		if err := path.EndpointB.AcknowledgePacket(packet, ack); err != nil {
			return nil, nil, err
		}

		return res, ack, nil
	}

	return nil, nil, fmt.Errorf("packet commitment does not exist on either endpoint for provided packet")
}

// Setup constructs a TM client, connection, and channel on both chains provided. It will
// fail if any error occurs. The clientID's, TestConnections, and TestChannels are returned
// for both chains. The channels created are connected to the ibc-transfer application.
func (path *Path) Setup() {
	path.SetupConnections()

	// channels can also be referenced through the returned connections
	path.CreateChannels()
}

// SetupClients is a helper function to create clients on both chains. It assumes the
// caller does not anticipate any errors.
func (path *Path) SetupClients() {
	err := path.EndpointA.CreateClient()
	if err != nil {
		panic(err)
	}

	err = path.EndpointB.CreateClient()
	if err != nil {
		panic(err)
	}
}

// SetupClientConnections is a helper function to create clients and the appropriate
// connections on both the source and counterparty chain. It assumes the caller does not
// anticipate any errors.
func (path *Path) SetupConnections() {
	path.SetupClients()

	path.CreateConnections()
}

// CreateConnection constructs and executes connection handshake messages in order to create
// OPEN channels on chainA and chainB. The connection information of for chainA and chainB
// are returned within a TestConnection struct. The function expects the connections to be
// successfully opened otherwise testing will fail.
func (path *Path) CreateConnections() {
	err := path.EndpointA.ConnOpenInit()
	if err != nil {
		panic(err)
	}

	err = path.EndpointB.ConnOpenTry()
	if err != nil {
		panic(err)
	}
	err = path.EndpointA.ConnOpenAck()
	if err != nil {
		panic(err)
	}

	err = path.EndpointB.ConnOpenConfirm()
	if err != nil {
		panic(err)
	}

	// ensure counterparty is up to date
	err = path.EndpointA.UpdateClient()
	if err != nil {
		panic(err)
	}
}

// CreateChannel constructs and executes channel handshake messages in order to create
// OPEN channels on chainA and chainB. The function expects the channels to be successfully
// opened otherwise testing will fail.
func (path *Path) CreateChannels() {
	err := path.EndpointA.ChanOpenInit()
	if err != nil {
		panic(err)
	}

	err = path.EndpointB.ChanOpenTry()
	if err != nil {
		panic(err)
	}

	err = path.EndpointA.ChanOpenAck()
	if err != nil {
		panic(err)
	}

	err = path.EndpointB.ChanOpenConfirm()
	if err != nil {
		panic(err)
	}
	// ensure counterparty is up to date
	err = path.EndpointA.UpdateClient()
	if err != nil {
		panic(err)
	}
}
