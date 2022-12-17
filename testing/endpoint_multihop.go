package ibctesting

import (
	clienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
)

// EndpointM represents a multihop channel endpoint.
// It includes all intermediate endpoints in the linked paths.
// Invariants:
//   - paths[0].A == this.Endpoint
//   - paths[len(paths)-1].B == this.Counterparty
//
// None of the fields should be changed after creation.
type EndpointM struct {
	*Endpoint
	Counterparty *EndpointM

	// a list of single-hop Paths that are linked together,
	// eg. for chains {A,B,C,D} the linked paths would be Link{AB, BC, CD}
	paths LinkedPaths
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
	return A, Z
}

// ChanOpenInit will construct and execute a MsgChannelOpenInit on the associated EndpointM.
func (ep *EndpointM) ChanOpenInit() error {

	return nil
}

// ChanOpenTry will construct and execute a MsgChannelOpenTry on the associated EndpointM.
func (ep *EndpointM) ChanOpenTry() error {

	return nil
}

// ChanOpenAck will construct and execute a MsgChannelOpenAck on the associated EndpointM.
func (ep *EndpointM) ChanOpenAck() error {

	return nil
}

// ChanOpenConfirm will construct and execute a MsgChannelOpenConfirm on the associated EndpointM.
func (ep *EndpointM) ChanOpenConfirm() error {
	return nil
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
func (ep *EndpointM) SendPacket(
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (uint64, error) {

	return 0, nil
}

// RecvPacket receives a packet on the associated EndpointM.
// The counterparty client is updated.
func (ep *EndpointM) RecvPacket(packet channeltypes.Packet) error {

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
// ie. self's client state is propogated to the counterparty chain following the multihop channel path.
func (ep *EndpointM) UpdateAllClients() error {
	for _, path := range ep.paths {
		err := path.EndpointA.UpdateClient()
		if err != nil {
			return err
		}
	}
	return nil
}
