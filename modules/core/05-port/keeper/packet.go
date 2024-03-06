package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// IDEA: Issue separate capabilities to each module in the stack. When we retrieve the capability in SendPacket
// and WriteAcknowledgement, we know who it came from. This allows us to put the packet data and ack correctly into the map
func (k Keeper) SendPacket(ctx sdk.Context, sourcePort, sourceChannel string,
	timeoutHeight clienttypes.Height, timeoutTimestamp uint64, packetData porttypes.RoutedPacketData) error {

	// if !k.scopedKeeper.AuthenticateCapability(ctx, channelCap, host.ChannelCapabilityPath(sourcePort, sourceChannel)) {
	// 	return errorsmod.Wrapf(types.ErrChannelCapabilityNotFound, "caller does not own capability for channel, port ID (%s) channel ID (%s)", sourcePort, sourceChannel)
	// }

	channel, ok := k.channelKeeper.GetChannel(ctx, sourcePort, sourceChannel)
	if !ok {
		return errorsmod.Wrapf(channeltypes.ErrChannelNotFound, "channel %s not found", sourceChannel)
	}

	var routedVersion types.RoutedVersion
	routeErr := k.cdc.UnmarshalJSON([]byte(channel.Version), &routedVersion)
	if routeErr != nil {
		// TODO: Figure out backwards compatibility
		return nil
	}

	// // input packet data from base application into the routed packet data map
	// routedPacketData := types.RoutedPacketData{PacketData: make(map[string][]byte)}
	// routedPacketData.PacketData[routedVersion.Modules[len(routedVersion.Modules)-1]] = data

	// send packet data to each module in the route
	// since this is routing from the base application to core IBC
	// the routing must occur in reverse order
	for i := len(routedVersion.Modules) - 1; i >= 0; i-- {
		module := routedVersion.Modules[i]
		cbs, exists := k.Router.GetRoute(module)
		if !exists {
			return errorsmod.Wrapf(types.ErrInvalidRoute, "route '%s' does not exist", module)
		}
		if packetData.PacketData[module] != nil {
			var err error
			err = cbs.OnSendPacket(ctx, sourcePort, sourceChannel, packetData.PacketData[module])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (k Keeper) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) exported.Acknowledgement {
	application, _, err := k.LookupModuleByChannel(ctx, packet.GetDestPort(), packet.GetDestChannel())
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
	routedAck := types.RoutedPacketAcknowledgement{PacketAck: make(map[string][]byte)}
	for _, module := range modules {
		cbs, exists := k.Router.GetRoute(module)
		if !exists {
			return channeltypes.NewErrorAcknowledgement(types.ErrInvalidRoute)
		}

		ack = cbs.OnRecvPacket(ctx, packet, relayer)
		if ack != nil {
			routedAck.PacketAck[module] = ack.Acknowledgement()
		}
		// return first unsuccessful ack
		if !ack.Success() {
			return ack
		}
	}

	return routedAck
}

func (k Keeper) OnAcknowledgementPacket(ctx sdk.Context, packet channeltypes.Packet, ack exported.Acknowledgement, relayer sdk.AccAddress) error {
	application, _, err := k.LookupModuleByChannel(ctx, packet.GetDestPort(), packet.GetDestChannel())
	if err != nil {
		return err
	}

	channel, ok := k.channelKeeper.GetChannel(ctx, packet.GetDestPort(), packet.GetDestChannel())
	if !ok {
		panic("channel not found")
	}

	var routedVersion types.RoutedVersion
	routeErr := k.cdc.UnmarshalJSON([]byte(channel.Version), &routedVersion)
	var modules []string
	routedAck := types.RoutedPacketAcknowledgement{PacketAck: make(map[string][]byte)}
	if routeErr != nil {
		if routedVersion.Modules[len(routedVersion.Modules)-1] != application {
			return types.ErrInvalidRoute
		}
		modules = routedVersion.Modules
	} else {
		// Lookup modules by port capability
		modules = []string{application}
		routedAck.PacketAck[application] = ack.Acknowledgement()
	}

	for _, module := range modules {
		cbs, exists := k.Router.GetRoute(module)
		if !exists {
			return types.ErrInvalidRoute
		}

		err := cbs.OnAcknowledgementPacket(ctx, packet, routedAck.PacketAck[module], relayer)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	application, _, err := k.LookupModuleByChannel(ctx, packet.GetDestPort(), packet.GetDestChannel())
	if err != nil {
		return err
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
			return types.ErrInvalidRoute
		}
		modules = routedVersion.Modules
	} else {
		// Lookup modules by port capability
		modules = []string{application}
	}

	for _, module := range modules {
		cbs, exists := k.Router.GetRoute(module)
		if !exists {
			return types.ErrInvalidRoute
		}

		err := cbs.OnTimeoutPacket(ctx, packet, relayer)
		if err != nil {
			return err
		}
	}
	return nil
}
