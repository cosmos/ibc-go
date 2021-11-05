package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/v2/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v2/modules/core/04-channel/types"
)

// TrySendTx takes in a transaction from an authentication module and attempts to send the packet
// if the base application has the capability to send on the provided portID
func (k Keeper) TrySendTx(ctx sdk.Context, chanCap *capabilitytypes.Capability, portID string, icaPacketData types.InterchainAccountPacketData) (uint64, error) {
	// Check for the active channel
	activeChannelID, found := k.GetActiveChannelID(ctx, portID)
	if !found {
		return 0, sdkerrors.Wrapf(types.ErrActiveChannelNotFound, "failed to retrieve active channel for port %s", portID)
	}

	sourceChannelEnd, found := k.channelKeeper.GetChannel(ctx, portID, activeChannelID)
	if !found {
		return 0, sdkerrors.Wrap(channeltypes.ErrChannelNotFound, activeChannelID)
	}

	destinationPort := sourceChannelEnd.GetCounterparty().GetPortID()
	destinationChannel := sourceChannelEnd.GetCounterparty().GetChannelID()

	return k.createOutgoingPacket(ctx, portID, activeChannelID, destinationPort, destinationChannel, chanCap, icaPacketData)
}

func (k Keeper) createOutgoingPacket(
	ctx sdk.Context,
	sourcePort,
	sourceChannel,
	destinationPort,
	destinationChannel string,
	chanCap *capabilitytypes.Capability,
	icaPacketData types.InterchainAccountPacketData,
) (uint64, error) {
	if err := icaPacketData.ValidateBasic(); err != nil {
		return 0, sdkerrors.Wrap(err, "invalid interchain account packet data")
	}

	// get the next sequence
	sequence, found := k.channelKeeper.GetNextSequenceSend(ctx, sourcePort, sourceChannel)
	if !found {
		return 0, sdkerrors.Wrapf(channeltypes.ErrSequenceSendNotFound, "failed to retrieve next sequence send for channel %s on port %s", sourceChannel, sourcePort)
	}

	// timeoutTimestamp is set to be a max number here so that we never recieve a timeout
	// ics-27-1 uses ordered channels which can close upon recieving a timeout, which is an undesired effect
	const timeoutTimestamp = ^uint64(0) >> 1 // Shift the unsigned bit to satisfy hermes relayer timestamp conversion

	packet := channeltypes.NewPacket(
		icaPacketData.GetBytes(),
		sequence,
		sourcePort,
		sourceChannel,
		destinationPort,
		destinationChannel,
		clienttypes.ZeroHeight(),
		timeoutTimestamp,
	)

	if err := k.ics4Wrapper.SendPacket(ctx, chanCap, packet); err != nil {
		return 0, err
	}

	return packet.Sequence, nil
}

// AuthenticateTx ensures the provided msgs contain the correct interchain account signer address retrieved
// from state using the provided controller port identifier
func (k Keeper) AuthenticateTx(ctx sdk.Context, msgs []sdk.Msg, portID string) error {
	interchainAccountAddr, found := k.GetInterchainAccountAddress(ctx, portID)
	if !found {
		return sdkerrors.Wrapf(types.ErrInterchainAccountNotFound, "failed to retrieve interchain account on port %s", portID)
	}

	for _, msg := range msgs {
		for _, signer := range msg.GetSigners() {
			if interchainAccountAddr != signer.String() {
				return sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "unexpected signer address: expected %s, got %s", interchainAccountAddr, signer.String())
			}
		}
	}

	return nil
}

func (k Keeper) executeTx(ctx sdk.Context, sourcePort, destPort, destChannel string, msgs []sdk.Msg) error {
	if err := k.AuthenticateTx(ctx, msgs, sourcePort); err != nil {
		return err
	}

	for _, msg := range msgs {
		if err := msg.ValidateBasic(); err != nil {
			return err
		}
	}

	// CacheContext returns a new context with the multi-store branched into a cached storage object
	// writeCache is called only if all msgs succeed, performing state transitions atomically
	cacheCtx, writeCache := ctx.CacheContext()
	for _, msg := range msgs {
		if _, err := k.executeMsg(cacheCtx, msg); err != nil {
			return err
		}
	}

	writeCache()

	return nil
}

// It tries to get the handler from router. And, if router exites, it will perform message.
func (k Keeper) executeMsg(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
	handler := k.msgRouter.Handler(msg)
	if handler == nil {
		return nil, types.ErrInvalidRoute
	}

	return handler(ctx, msg)
}

func (k Keeper) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet) error {
	var data types.InterchainAccountPacketData

	if err := types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		// UnmarshalJSON errors are indeterminate and therefore are not wrapped and included in failed acks
		return sdkerrors.Wrapf(types.ErrUnknownDataType, "cannot unmarshal ICS-27 interchain account packet data")
	}

	switch data.Type {
	case types.EXECUTE_TX:
		msgs, err := types.DeserializeCosmosTx(k.cdc, data.Data)
		if err != nil {
			return err
		}

		if err = k.executeTx(ctx, packet.SourcePort, packet.DestinationPort, packet.DestinationChannel, msgs); err != nil {
			return err
		}

		return nil
	default:
		return types.ErrUnknownDataType
	}
}

// OnTimeoutPacket removes the active channel associated with the provided packet, the underlying channel end is closed
// due to the semantics of ORDERED channels
func (k Keeper) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet) error {
	k.DeleteActiveChannelID(ctx, packet.SourcePort)

	return nil
}
