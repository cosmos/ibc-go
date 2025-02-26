package ibctesting

import (
	"bytes"
	"errors"

	abci "github.com/cometbft/cometbft/abci/types"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
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
	path.EndpointA.ChannelConfig.Version = transfertypes.V1
	path.EndpointB.ChannelConfig.Version = transfertypes.V1

	return path
}

// SetChannelOrdered sets the channel order for both endpoints to ORDERED.
func (path *Path) SetChannelOrdered() {
	path.EndpointA.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointB.ChannelConfig.Order = channeltypes.ORDERED
}

// DisableUniqueChannelIDs provides an opt-out way to not have all channel IDs be different
// while testing.
func (path *Path) DisableUniqueChannelIDs() *Path {
	path.EndpointA.disableUniqueChannelIDs = true
	path.EndpointB.disableUniqueChannelIDs = true
	return path
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
	if bytes.Equal(pc, channeltypes.CommitPacket(packet)) {
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
	if bytes.Equal(pc, channeltypes.CommitPacket(packet)) {
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

	return nil, nil, errors.New("packet commitment does not exist on either endpoint for provided packet")
}

// Reversed returns a new path with endpoints reversed.
func (path *Path) Reversed() *Path {
	reversedPath := *path
	reversedPath.EndpointA, reversedPath.EndpointB = path.EndpointB, path.EndpointA
	return &reversedPath
}

// Setup constructs a TM client, connection, and channel on both chains provided. It will
// fail if any error occurs.
func (path *Path) Setup() {
	path.SetupConnections()

	// channels can also be referenced through the returned connections
	path.CreateChannels()
}

// SetupV2 constructs clients on both sides and then provides the counterparties for both sides
// This is all that is necessary for path setup with the IBC v2 protocol
func (path *Path) SetupV2() {
	path.SetupClients()

	path.SetupCounterparties()
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

// SetupCounterparties is a helper function to set the counterparties supporting IBC v2 on both
// chains. It assumes the caller does not anticipate any errors.
func (path *Path) SetupCounterparties() {
	if err := path.EndpointB.RegisterCounterparty(); err != nil {
		panic(err)
	}

	if err := path.EndpointA.RegisterCounterparty(); err != nil {
		panic(err)
	}
}

// SetupConnections is a helper function to create clients and the appropriate
// connections on both the source and counterparty chain. It assumes the caller does not
// anticipate any errors.
func (path *Path) SetupConnections() {
	path.SetupClients()

	path.CreateConnections()
}

// CreateConnections constructs and executes connection handshake messages in order to create
// OPEN connections on chainA and chainB. The function expects the connections to be
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

// CreateChannels constructs and executes channel handshake messages in order to create
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
