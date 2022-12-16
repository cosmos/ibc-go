package ibctesting

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// PathM represents a multihop channel path between two chains.
type PathM struct {
	EndpointA *EndpointM
	EndpointZ *EndpointM
}

// SetChannelOrdered sets the channel order for both endpoints to ORDERED. Default channel is Unordered.
func (path *PathM) SetChannelOrdered() {
	path.EndpointA.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointZ.ChannelConfig.Order = channeltypes.ORDERED
}

// LinkedPaths is a list of linked ibc paths, A -> B -> C -> ... -> Z, where {A,B,C,...,Z} are chains, and A/Z is the first/last chain endpoint.
type LinkedPaths []*Path

// CreateLinkedChains creates `num` chains and set up a Path between each pair of chains
// return the coordinator, the `num` chains, and `num-1` connected Paths
func CreateLinkedChains(
	t *suite.Suite,
	num int,
) (*Coordinator, LinkedPaths) {
	coord := NewCoordinator(t.T(), num)
	paths := make([]*Path, num-1)

	for i := 0; i < num-1; i++ {
		paths[i] = NewPath(coord.GetChain(GetChainID(i+1)), coord.GetChain(GetChainID(i+2)))
	}

	// create connections for each path
	for _, path := range paths {
		path := path
		t.Require().Equal(path.EndpointA.ConnectionID, "")
		t.Require().Equal(path.EndpointB.ConnectionID, "")
		coord.SetupConnections(path)
		t.Require().NotEqual(path.EndpointA.ConnectionID, "")
		t.Require().NotEqual(path.EndpointB.ConnectionID, "")
	}

	return coord, paths
}

// ToPathM converts a LinkedPaths to a PathM where the EndpointA has the same linking paths as the LinkedPaths and
// EndpointZ has the reverse linking paths.
func (paths LinkedPaths) ToPathM() *PathM {
	a, z := NewEndpointMFromLinkedPaths(paths)
	return &PathM{
		EndpointA: &a,
		EndpointZ: &z,
	}
}

// Last returns the last Path in LinkedPaths.
func (paths LinkedPaths) Last() *Path {
	return paths[len(paths)-1]
}

// First returns the first Path in LinkedPaths.
func (paths LinkedPaths) First() *Path {
	return paths[0]
}

// A returns the first chain in the paths, aka. the source chain.
func (paths LinkedPaths) A() *Endpoint {
	return paths.First().EndpointA
}

// Z returns the last chain in the paths, aka. the destination chain.
func (paths LinkedPaths) Z() *Endpoint {
	return paths.Last().EndpointB
}

// Reverse a list of paths from chain A to chain Z.
// Return a list of paths from chain Z to chain A, where the endpoints A/B are also swapped.
func (paths LinkedPaths) Reverse() LinkedPaths {
	var reversed LinkedPaths
	for i := range paths {
		orgPath := paths[len(paths)-1-i]
		path := Path{
			EndpointA: orgPath.EndpointB,
			EndpointB: orgPath.EndpointA,
		}
		reversed = append(reversed, &path)
	}
	return reversed
}

// UpdateClients iterates through each chain in the path and calls UpdateClient
func (paths LinkedPaths) UpdateClients() LinkedPaths {
	for _, path := range paths {
		if path.EndpointB.ClientID != "" {
			if err := path.EndpointB.UpdateClient(); err != nil {
				panic(err)
			}
		}
	}
	return paths
}

// GetConnectionHops returns connection IDs on {A, B,... Y}
func (paths LinkedPaths) GetConnectionHops() (connectionHops []string) {
	for _, path := range paths {
		connectionHops = append(connectionHops, path.EndpointA.ConnectionID)
	}
	return
}

// ChanOpenInit is a copy of endpoint.ChanOpenInit which allows specifiying connectionHops
func ChanOpenInit(paths LinkedPaths) {
	endpoint := paths.A()

	msg := channeltypes.NewMsgChannelOpenInit(
		endpoint.ChannelConfig.PortID,
		endpoint.ChannelConfig.Version, endpoint.ChannelConfig.Order, paths.GetConnectionHops(),
		endpoint.Counterparty.ChannelConfig.PortID,
		endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := endpoint.Chain.SendMsgs(msg)
	require.NoError(endpoint.Chain.T, err)

	endpoint.ChannelID, err = ParseChannelIDFromEvents(res.GetEvents())
	require.NoError(endpoint.Chain.T, err)

	// update version to selected app version
	// NOTE: this update must be performed after SendMsgs()
	endpoint.ChannelConfig.Version = endpoint.GetChannel().Version

	// update clients
	paths.UpdateClients()
}

// ChanOpenTry generates a multihop proof to call ChanOpenTry on chain Z.
// Confirms a channel open init on chain A.
func ChanOpenTry(paths LinkedPaths) {
	endpointA := paths.A()
	endpointZ := paths.Z()

	err := endpointZ.UpdateClient()
	require.NoError(endpointZ.Chain.T, err)

	req := &channeltypes.QueryChannelRequest{
		PortId:    endpointA.ChannelConfig.PortID,
		ChannelId: endpointA.ChannelID,
	}

	// receive the channel response and marshal to expected value bytes
	resp, err := endpointA.Chain.App.GetIBCKeeper().Channel(endpointA.Chain.GetContext(), req)
	require.NoError(endpointA.Chain.T, err)
	expectedVal, err := resp.Channel.Marshal()
	require.NoError(endpointA.Chain.T, err)

	channelKey := host.ChannelKey(endpointA.ChannelConfig.PortID, endpointA.ChannelID)
	proofs, err := GenerateMultiHopProof(paths, channelKey, expectedVal, false)
	require.NoError(endpointA.Chain.T, err)

	// verify call to ChanOpenTry completes successfully
	height := endpointZ.GetClientState().GetLatestHeight()
	proof, err := proofs.Marshal()
	require.NoError(endpointZ.Chain.T, err)

	msg := channeltypes.NewMsgChannelOpenTry(
		endpointZ.ChannelConfig.PortID,
		endpointZ.ChannelConfig.Version, endpointZ.ChannelConfig.Order, paths.Reverse().GetConnectionHops(),
		endpointA.ChannelConfig.PortID, endpointA.ChannelID,
		endpointA.ChannelConfig.Version,
		proof, height.(types.Height),
		endpointZ.Chain.SenderAccount.GetAddress().String(),
	)
	res, err := endpointZ.Chain.SendMsgs(msg)
	require.NoError(endpointZ.Chain.T, err)

	if endpointZ.ChannelID == "" {
		endpointZ.ChannelID, err = ParseChannelIDFromEvents(res.GetEvents())
		require.NoError(endpointZ.Chain.T, err)
	}

	// update version to selected app version
	// NOTE: this update must be performed after the endpoint channelID is set
	endpointZ.ChannelConfig.Version = endpointZ.GetChannel().Version

	// update clients
	paths.Reverse().UpdateClients()
}

// ChanOpenAck generates a multihop proof to call ChanOpenAck on chain A.
// Confirms a channel open Try on chain Z.
func ChanOpenAck(paths LinkedPaths) {
	endpointA := paths.A()
	endpointZ := paths.Z()

	err := endpointA.UpdateClient()
	require.NoError(endpointA.Chain.T, err)

	channelKey := host.ChannelKey(endpointZ.ChannelConfig.PortID, endpointZ.ChannelID)
	// query the channel
	req := &channeltypes.QueryChannelRequest{
		PortId:    endpointZ.ChannelConfig.PortID,
		ChannelId: endpointZ.ChannelID,
	}

	// receive the channel response and marshal to expected value bytes
	resp, err := endpointZ.Chain.App.GetIBCKeeper().Channel(endpointZ.Chain.GetContext(), req)
	require.NoError(endpointZ.Chain.T, err)
	expectedVal, err := resp.Channel.Marshal()
	require.NoError(endpointZ.Chain.T, err)

	// generate multihop proof given keypath and value
	proofs, err := GenerateMultiHopProof(paths.Reverse(), channelKey, expectedVal, false)
	require.NoError(endpointZ.Chain.T, err)
	// verify call to ChanOpenTry completes successfully
	height := endpointA.GetClientState().GetLatestHeight()
	proof, err := proofs.Marshal()
	require.NoError(endpointZ.Chain.T, err)

	msg := channeltypes.NewMsgChannelOpenAck(
		endpointA.ChannelConfig.PortID, endpointA.ChannelID,
		endpointZ.ChannelID, endpointZ.ChannelConfig.Version, // testing doesn't use flexible selection
		proof, height.(types.Height),
		endpointA.Chain.SenderAccount.GetAddress().String(),
	)

	err = endpointA.Chain.sendMsgs(msg)
	require.NoError(endpointA.Chain.T, err)

	endpointA.ChannelConfig.Version = endpointA.GetChannel().Version

	// update clients
	paths.UpdateClients()
}

// ChanOpenConfirm generates a multihop proof to call ChanOpenConfirm on chain Z.
// Confirms a channel open Ack on chain A.
func ChanOpenConfirm(paths LinkedPaths) {
	endpointA := paths.A()
	endpointZ := paths.Z()

	err := endpointZ.UpdateClient()
	require.NoError(endpointA.Chain.T, err)

	channelKey := host.ChannelKey(endpointA.ChannelConfig.PortID, FirstChannelID)
	// query the channel
	req := &channeltypes.QueryChannelRequest{
		PortId:    endpointA.ChannelConfig.PortID,
		ChannelId: endpointA.ChannelID,
	}

	// receive the channel response and marshal to expected value bytes
	resp, err := endpointA.Chain.App.GetIBCKeeper().Channel(endpointA.Chain.GetContext(), req)
	require.NoError(endpointA.Chain.T, err)
	expectedVal, err := resp.Channel.Marshal()
	require.NoError(endpointA.Chain.T, err)

	// generate multihop proof given keypath and value
	proofs, err := GenerateMultiHopProof(paths, channelKey, expectedVal, false)
	require.NoError(endpointA.Chain.T, err)
	// verify call to ChanOpenTry completes successfully
	height := endpointZ.GetClientState().GetLatestHeight()
	proof, err := proofs.Marshal()
	require.NoError(endpointA.Chain.T, err)

	msg := channeltypes.NewMsgChannelOpenConfirm(
		endpointZ.ChannelConfig.PortID, endpointZ.ChannelID,
		proof, height.(types.Height),
		endpointZ.Chain.SenderAccount.GetAddress().String(),
	)
	err = endpointZ.Chain.sendMsgs(msg)
	require.NoError(endpointA.Chain.T, err)

	// update clients
	paths.Reverse().UpdateClients()
}

// SetupChannel completes a multihop channel handshake
func SetupChannel(paths LinkedPaths) {
	ChanOpenInit(paths)
	ChanOpenTry(paths)
	ChanOpenAck(paths)
	ChanOpenConfirm(paths)
}

// ReceivePacket receives a packet on chain Z with multihop proof
func RecvPacket(paths LinkedPaths, packet channeltypes.Packet) (*sdk.Result, error) {

	endpointA := paths.A()
	endpointZ := paths.Z()

	// get proof of packet commitment from chainA
	packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	expectedVal := channeltypes.CommitPacket(endpointA.Chain.Codec, packet)

	// generate multihop proof given keypath and value
	proofs, err := GenerateMultiHopProof(paths, packetKey, expectedVal, false)
	require.NoError(endpointA.Chain.T, err)
	proofHeight := endpointZ.GetClientState().GetLatestHeight()
	proof, err := proofs.Marshal()
	require.NoError(endpointA.Chain.T, err)

	recvMsg := channeltypes.NewMsgRecvPacket(
		packet,
		proof,
		proofHeight.(types.Height),
		endpointZ.Chain.SenderAccount.GetAddress().String(),
	)

	// receive on counterparty and update source client
	res, err := endpointZ.Chain.SendMsgs(recvMsg)
	if err != nil {
		return nil, err
	}

	if err := endpointA.UpdateClient(); err != nil {
		return nil, err
	}

	return res, nil
}
