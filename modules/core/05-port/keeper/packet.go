package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/v8/modules/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

func (k Keeper) SendPacket(ctx sdk.Context, channelCap *capabilitytypes.Capability, sourcePort, sourceChannel string,
	timeoutHeight clienttypes.Height, timeoutTimestamp uint64, data []byte) (uint64, error) {

	if !k.scopedKeeper.AuthenticateCapability(ctx, channelCap, host.ChannelCapabilityPath(sourcePort, sourceChannel)) {
		return 0, errorsmod.Wrapf(types.ErrChannelCapabilityNotFound, "caller does not own capability for channel, port ID (%s) channel ID (%s)", sourcePort, sourceChannel)
	}

	channel, ok := k.channelKeeper.GetChannel(ctx, sourcePort, sourceChannel)
	if !ok {
		return 0, errorsmod.Wrapf(channeltypes.ErrChannelNotFound, "channel %s not found", sourceChannel)
	}

	var routedVersion types.RoutedVersion
	routeErr := k.cdc.UnmarshalJSON([]byte(channel.Version), &routedVersion)
	if routeErr != nil {
		// send directly to channel keeper for backwards compatibility
		return k.channelKeeper.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
	}

	// input packet data from base application into the routed packet data map
	routedPacketData := types.RoutedPacketData{PacketData: make(map[string][]byte)}
	routedPacketData.PacketData[routedVersion.Modules[len(routedVersion.Modules)-1]] = data

	// send packet data to each module in the route
	// since this is routing from the base application to core IBC
	// the routing must occur in reverse order
	// base app is skipped since it was the module that sent the original packet data
	for i := len(routedVersion.Modules) - 2; i >= 0; i-- {
		module := routedVersion.Modules[i]
		cbs, exists := k.Router.GetRoute(module)
		if !exists {
			return 0, errorsmod.Wrapf(types.ErrInvalidRoute, "route '%s' does not exist", module)
		}
		mw, ok := cbs.(types.Middleware)
		if ok {
			var err error
			routedPacketData, err = mw.ProcessPacket(ctx, sourcePort, sourceChannel, routedPacketData)
			if err != nil {
				return 0, err
			}
		}
	}

	packetData := k.cdc.MustMarshalJSON(&routedPacketData)
	return k.channelKeeper.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, packetData)
}

func (k Keeper) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) exported.Acknowledgement {
	application, chanCap, err := k.LookupModuleByChannel(ctx, packet.GetDestPort(), packet.GetDestChannel())
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	channel, ok := k.channelKeeper.GetChannel(ctx, packet.GetDestPort(), packet.GetDestChannel())
	if !ok {
		panic("channel not found")
	}

	var routedVersion types.RoutedVersion
	routeErr := k.cdc.UnmarshalJSON([]byte(channel.Version), &routedVersion)
	var modules []string
	if routeErr != nil {
		if routedVersion.Modules[len(routedVersion.Modules)-1] != application {
			return channeltypes.NewErrorAcknowledgement(types.ErrInvalidRoute)
		}
		modules = routedVersion.Modules
	} else {
		// Lookup modules by port capability
		modules = []string{application}
	}

	var ack exported.Acknowledgement
	for i, module := range modules {
		cbs, exists := k.Router.GetRoute(module)
		if !exists {
			return channeltypes.NewErrorAcknowledgement(types.ErrInvalidRoute)
		}

		if i == len(modules)-1 {
			ack = cbs.OnRecvPacket(ctx, packet, relayer)
		} else {
			mw, ok := cbs.(types.Middleware)
			if !ok {
				return channeltypes.NewErrorAcknowledgement(types.ErrInvalidRoute)
			}

			var routedPacketData types.RoutedPacketData
			if err := k.cdc.UnmarshalJSON(packet.GetData(), &routedPacketData); err != nil {
				return channeltypes.NewErrorAcknowledgement(types.ErrInvalidPacketData)
			}

			err := mw.ProcessRecvPacket(ctx, packet, relayer)
			if err != nil {
				return channeltypes.NewErrorAcknowledgement(err)
			}
		}
	}
	if ack != nil {
		var err error
		err = k.WriteAcknowledgement(ctx, chanCap, packet, ack)
		if err != nil {
			return channeltypes.NewErrorAcknowledgement(err)
		}
		// ack already written by above call
		return nil
	}
	return nil
}

func (k Keeper) WriteAcknowledgement(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet channeltypes.Packet, ack exported.Acknowledgement) error {
	if !k.scopedKeeper.AuthenticateCapability(ctx, chanCap, host.ChannelCapabilityPath(packet.GetDestPort(), packet.GetDestChannel())) {
		return errorsmod.Wrapf(types.ErrChannelCapabilityNotFound, "caller does not own capability for channel, port ID (%s) channel ID (%s)", packet.GetDestPort(), packet.GetDestChannel())
	}

	channel, ok := k.channelKeeper.GetChannel(ctx, packet.GetDestPort(), packet.GetDestChannel())
	if !ok {
		return errorsmod.Wrapf(channeltypes.ErrChannelNotFound, "channel %s not found", packet.GetDestChannel())
	}

	var routedVersion types.RoutedVersion
	routeErr := k.cdc.UnmarshalJSON([]byte(channel.Version), &routedVersion)
	if routeErr != nil {
		// send directly to channel keeper for backwards compatibility
		return k.channelKeeper.WriteAcknowledgement(ctx, packet, ack)
	}

	routedAck := types.RoutedPacketAcknowledgement{PacketAck: make(map[string][]byte)}
	routedAck.PacketAck[routedVersion.Modules[len(routedVersion.Modules)-1]] = ack.Acknowledgement()

	// send packet data to each module in the route
	// since this is routing from the base application to core IBC
	// the routing must occur in reverse order
	// base app is skipped since it was the module that sent the original packet data
	for i := len(routedVersion.Modules) - 2; i >= 0; i-- {
		module := routedVersion.Modules[i]
		cbs, exists := k.Router.GetRoute(module)
		if !exists {
			return errorsmod.Wrapf(types.ErrInvalidRoute, "route '%s' does not exist", module)
		}
		mw, ok := cbs.(types.Middleware)
		if ok {
			var err error
			routedAck, err = mw.ProcessWriteAck(ctx, packet, routedAck)
			if err != nil {
				return err
			}
		}
	}

	return k.channelKeeper.WriteAcknowledgement(ctx, packet, routedAck)
}
