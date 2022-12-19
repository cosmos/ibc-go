package ibctesting

import (
	clienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v6/modules/core/24-host"
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

	_, proof := ep.Counterparty.QueryChannelProof()
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

	_, proof := ep.Counterparty.QueryChannelProof()
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

	_, proof := ep.Counterparty.QueryChannelProof()
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
// ie. self's client state is propogated from the counterparty chain following the multihop channel path.
func (ep *EndpointM) UpdateAllClients() error {
	for _, path := range ep.Counterparty.paths {
		err := path.EndpointA.UpdateClient()
		if err != nil {
			return err
		}
	}
	return nil
}

// GetConnectionHops returns the connection hops for the multihop channel.
func (ep *EndpointM) GetConnectionHops() []string {
	return ep.paths.GetConnectionHops()
}

// QueryChannelProof queries the channel proof on the endpoint chain.
func (ep *EndpointM) QueryChannelProof() (*channeltypes.Channel, []byte) {
	// request := &channeltypes.QueryChannelRequest{
	// 	PortId:    ep.ChannelConfig.PortID,
	// 	ChannelId: ep.ChannelID,
	// }
	// resp, err := ep.Chain.App.GetIBCKeeper().Channel(ep.Chain.GetContext(), request)
	// require.NoError(ep.Chain.T, err, "could not query channel from chain %s", ep.Chain.ChainID)

	// channel := resp.GetChannel()
	channel := ep.GetChannel()
	channelKey := host.ChannelKey(ep.ChannelConfig.PortID, ep.ChannelID)
	proof, err := GenerateMultiHopProof(
		ep.paths,
		channelKey,
		ep.Chain.Codec.MustMarshal(&channel),
	)
	require.NoError(
		ep.Chain.T,
		err,
		"could not generate proof for channel %s on chain %s",
		ep.ChannelID,
		ep.Chain.ChainID,
	)

	return &channel, ep.Chain.Codec.MustMarshal(proof)
}
