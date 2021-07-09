package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/ibc-go/modules/apps/ccv/parent/types"
	ccv "github.com/cosmos/ibc-go/modules/apps/ccv/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	abci "github.com/tendermint/tendermint/abci/types"
)

func (k Keeper) SendPacket(ctx sdk.Context, chainID string, valUpdates []abci.ValidatorUpdate) error {
	packetData := ccv.NewValidatorSetChangePacketData(valUpdates)
	packetDataBytes := packetData.GetBytes()

	channelID, ok := k.GetChainToChannel(ctx, chainID)
	if !ok {
		return sdkerrors.Wrapf(channeltypes.ErrChannelNotFound, "channel not found for channel ID: %s", channelID)
	}
	channel, ok := k.channelKeeper.GetChannel(ctx, types.PortID, channelID)
	if !ok {
		return sdkerrors.Wrapf(channeltypes.ErrChannelNotFound, "channel not found for channel ID: %s", channelID)
	}
	channelCap, ok := k.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(types.PortID, channelID))
	if !ok {
		return sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}

	// get the next sequence
	sequence, found := k.channelKeeper.GetNextSequenceSend(ctx, types.PortID, channelID)
	if !found {
		return sdkerrors.Wrapf(
			channeltypes.ErrSequenceSendNotFound,
			"source port: %s, source channel: %s", types.PortID, channelID,
		)
	}

	// Send ValidatorSet changes in IBC packet
	packet := channeltypes.NewPacket(
		packetDataBytes, sequence,
		types.PortID, channelID,
		channel.Counterparty.PortId, channel.Counterparty.ChannelId,
		clienttypes.Height{}, uint64(types.GetTimeoutTimestamp(ctx.BlockTime()).UnixNano()),
	)
	if err := k.channelKeeper.SendPacket(ctx, channelCap, packet); err != nil {
		return err
	}

	k.SetUnbondingChanges(ctx, chainID, sequence, valUpdates)
	return nil
}

func (k Keeper) OnAcknowledgementPacket(ctx sdk.Context, packet channeltypes.Packet, data ccv.ValidatorSetChangePacketData, ack channeltypes.Acknowledgement) error {
	chainID, ok := k.GetChannelToChain(ctx, packet.DestinationChannel)
	if !ok {
		return sdkerrors.Wrapf(ccv.ErrInvalidChildChain, "chain ID doesn't exist for channel ID: %s", packet.DestinationChannel)
	}
	if err := data.Unmarshal(packet.GetData()); err != nil {
		return err
	}
	k.registryKeeper.UnbondValidators(ctx, chainID, data.ValidatorUpdates)
	k.DeleteUnbondingChanges(ctx, chainID, packet.Sequence)
	return nil
}

func (k Keeper) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, data ccv.ValidatorSetChangePacketData) error {
	k.SetChannelStatus(ctx, packet.DestinationChannel, ccv.Invalid)
	// TODO: Unbonding everything?
	return nil
}

// EndBlockCallback is called for each baby chain in Endblock. It sends latest validator updates to each baby chain
// in a packet over the CCV channel.
func (k Keeper) EndBlockCallback(ctx sdk.Context, chainID string) bool {
	// SKIP THIS UNTIL registryKeeper is implemented
	if k.registryKeeper == nil {
		return false
	}
	valUpdates := k.registryKeeper.GetValidatorSetChanges(chainID)
	if len(valUpdates) != 0 {
		k.SendPacket(ctx, chainID, valUpdates)
	}
	return false
}
