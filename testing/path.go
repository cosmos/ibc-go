package ibctesting

import (
	"bytes"
	"fmt"

	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
)

// Path
type Path struct {
	EndpointA *Endpoint
	EndpointB *Endpoint
}

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

// SetChannelOrdered sets the channel order for both endpoints to ORDERED.
func (path *Path) SetChannelOrdered() {
	path.EndpointA.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointB.ChannelConfig.Order = channeltypes.ORDERED
}

// RelayPacket
func (path *Path) RelayPacket(packet channeltypes.Packet, ack []byte) error {
	pc := path.EndpointA.Chain.App.IBCKeeper.ChannelKeeper.GetPacketCommitment(path.EndpointA.Chain.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	if bytes.Equal(pc, channeltypes.CommitPacket(path.EndpointA.Chain.App.AppCodec(), packet)) {

		// packet found, relay from A to B
		path.EndpointB.UpdateClient()

		if err := path.EndpointB.RecvPacket(packet); err != nil {
			return err
		}

		if err := path.EndpointA.AcknowledgePacket(packet, ack); err != nil {
			return err
		}
		return nil

	}

	pc = path.EndpointB.Chain.App.IBCKeeper.ChannelKeeper.GetPacketCommitment(path.EndpointB.Chain.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	if bytes.Equal(pc, channeltypes.CommitPacket(path.EndpointB.Chain.App.AppCodec(), packet)) {

		// packet found, relay B to A
		path.EndpointA.UpdateClient()

		if err := path.EndpointA.RecvPacket(packet); err != nil {
			return err
		}
		if err := path.EndpointB.AcknowledgePacket(packet, ack); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("packet commitment does not exist on either endpoint for provided packet")
}
