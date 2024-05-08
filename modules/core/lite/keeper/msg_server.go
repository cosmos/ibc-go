package keeper

import (
	"context"

	"cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/cosmos/ibc-go/v8/modules/core/lite/types"
)

var _ channeltypes.PacketMsgServer = (*Keeper)(nil)

type Keeper struct {
	cdc                 codec.BinaryCodec
	channelStoreService store.KVStoreService
	clientStoreService  store.KVStoreService
	channelKeeper types.ChannelKeeper
	clientRouter types.ClientRouter
	appRouter porttypes.Router
}

func NewKeeper(cdc codec.BinaryCodec) *Keeper {
	return &Keeper{
		cdc: cdc,
	}
}

// SendPacket implements the MsgServer interface. It creates a new packet
// with the given source port and source channel and sends it over the channel
// end with the given destination port and channel identifiers.
func (k Keeper) SendPacket(context.Context, *channeltypes.MsgSendPacket) (*channeltypes.MsgSendPacketResponse, error) {

	return nil, nil
}

// ReceivePacket implements the MsgServer interface. It receives an incoming
// packet, which was sent over a channel end with the given port and channel
// identifiers, performs all necessary application logic, and then
// acknowledges the packet.
func (k Keeper) RecvPacket(goCtx context.Context, msg *channeltypes.MsgRecvPacket) (*channeltypes.MsgReceivePacketResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Lookup counterparty associated with our channel and ensure that it was packet was indeed
	// sent by our counterparty.
	// Note: This can be implemented by the current channelKeeper
	_, counterpartyChannelId, found := k.channelKeeper.GetCounterparty(goCtx, "", msg.Packet.DestinationChannel)
	if !found {
		return nil, channeltypes.ErrChannelNotFound
	}

	if counterpartyChannelId != msg.Packet.SourceChannel {
		return nil, channeltypes.ErrInvalidChannelIdentifier
	}

	// create key/value pair for proof verification
	key := host.PacketCommitmentKey(msg.Packet.SourcePort, msg.Packet.SourceChannel, msg.Packet.Sequence)
	commitment := types.CommitPacket(k.cdc, packet)

	// Get LightClientModule associated with the destination channel
	// Note: This can be implemented by the current clientRouter
	lightClientModule := k.clientRouter.Route(msg.Packet.DestinationChannel)

	// TODO: Use context instead of sdk.Context eventually
	if err := lightClientModule.VerifyMembership(
		ctx,
		msg.Packet.DestinationChannel,
		msg.ProofHeight,
		0, 0,
		msg.ProofCommitment,
		key,
		commitment,
	) {
		return nil, err
	}

	// Port should directly correspond to the application module route
	// No need for capabilities and mapping from portID to ModuleName
	appModule = k.appRouter.Route(msg.Packet.DestinationPort)

	// TODO: Figure out how to do caching generically without using SDK
	// Perform application logic callback
	//
	// Cache context so that we may discard state changes from callback if the acknowledgement is unsuccessful.
	cacheCtx, writeFn := ctx.CacheContext()
	ack := appModule.OnRecvPacket(cacheCtx, msg.Packet, relayer)
	if ack == nil || ack.Success() {
		// write application state changes for asynchronous and successful acknowledgements
		writeFn()
	} else {
		// Modify events in cached context to reflect unsuccessful acknowledgement
		// TODO: How do we create interface for this that isn't too SDK specific?
		ctx.EventManager().EmitEvents(convertToErrorEvents(cacheCtx.EventManager().Events()))
	}

	// Write acknowledgement to store
	if ack != nil {
		// Can be implemented by current channelKeeper with change in sdk.Context to context.Context
		k.channelKeeper.WriteAcknowledgement(goCtx, msg.Packet.DestinationPort, msg.Packet.DestinationChannel, ack.Acknowledgement)
	}

	return &channeltypes.MsgRecvPacketResponse{Result: channeltypes.SUCCESS}, nil
}


