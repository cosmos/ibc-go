package keeper

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// ChanOpenInit is called by a module to initiate a channel opening handshake with
// a module on another chain. The counterparty channel identifier is validated to be
// empty in msg validation.
func (k *Keeper) ChanOpenInit(
	ctx sdk.Context,
	order types.Order,
	connectionHops []string,
	portID string,
	counterparty types.Counterparty,
	version string,
) (string, error) {
	// connection hop length checked on msg.ValidateBasic()
	connectionEnd, found := k.connectionKeeper.GetConnection(ctx, connectionHops[0])
	if !found {
		return "", errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, connectionHops[0])
	}

	getVersions := connectionEnd.Versions
	if len(getVersions) != 1 {
		return "", errorsmod.Wrapf(
			connectiontypes.ErrInvalidVersion,
			"single version must be negotiated on connection before opening channel, got: %v",
			getVersions,
		)
	}

	if !connectiontypes.VerifySupportedFeature(getVersions[0], order.String()) {
		return "", errorsmod.Wrapf(
			connectiontypes.ErrInvalidVersion,
			"connection version %s does not support channel ordering: %s",
			getVersions[0], order.String(),
		)
	}

	if status := k.clientKeeper.GetClientStatus(ctx, connectionEnd.ClientId); status != exported.Active {
		return "", errorsmod.Wrapf(clienttypes.ErrClientNotActive, "client (%s) status is %s", connectionEnd.ClientId, status)
	}

	channelID := k.GenerateChannelIdentifier(ctx)

	return channelID, nil
}

// WriteOpenInitChannel writes a channel which has successfully passed the OpenInit handshake step.
// The channel is set in state and all the associated Send and Recv sequences are set to 1.
// An event is emitted for the handshake step.
func (k *Keeper) WriteOpenInitChannel(
	ctx sdk.Context,
	portID,
	channelID string,
	order types.Order,
	connectionHops []string,
	counterparty types.Counterparty,
	version string,
) {
	channel := types.NewChannel(types.INIT, order, counterparty, connectionHops, version)
	k.SetChannel(ctx, portID, channelID, channel)

	k.SetNextSequenceSend(ctx, portID, channelID, 1)
	k.SetNextSequenceRecv(ctx, portID, channelID, 1)
	k.SetNextSequenceAck(ctx, portID, channelID, 1)

	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", types.UNINITIALIZED, "new-state", types.INIT)

	defer telemetry.IncrCounter(1, "ibc", "channel", "open-init")

	emitChannelOpenInitEvent(ctx, portID, channelID, channel)
}

// ChanOpenTry is called by a module to accept the first step of a channel opening
// handshake initiated by a module on another chain.
func (k *Keeper) ChanOpenTry(
	ctx sdk.Context,
	order types.Order,
	connectionHops []string,
	portID string,
	counterparty types.Counterparty,
	counterpartyVersion string,
	initProof []byte,
	proofHeight exported.Height,
) (string, error) {
	// connection hops only supports a single connection
	if len(connectionHops) != 1 {
		return "", errorsmod.Wrapf(types.ErrTooManyConnectionHops, "expected 1, got %d", len(connectionHops))
	}

	// generate a new channel
	channelID := k.GenerateChannelIdentifier(ctx)

	connectionEnd, found := k.connectionKeeper.GetConnection(ctx, connectionHops[0])
	if !found {
		return "", errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, connectionHops[0])
	}

	if connectionEnd.State != connectiontypes.OPEN {
		return "", errorsmod.Wrapf(connectiontypes.ErrInvalidConnectionState, "connection state is not OPEN (got %s)", connectionEnd.State)
	}

	getVersions := connectionEnd.Versions
	if len(getVersions) != 1 {
		return "", errorsmod.Wrapf(
			connectiontypes.ErrInvalidVersion,
			"single version must be negotiated on connection before opening channel, got: %v",
			getVersions,
		)
	}

	if !connectiontypes.VerifySupportedFeature(getVersions[0], order.String()) {
		return "", errorsmod.Wrapf(
			connectiontypes.ErrInvalidVersion,
			"connection version %s does not support channel ordering: %s",
			getVersions[0], order.String(),
		)
	}

	counterpartyHops := []string{connectionEnd.Counterparty.ConnectionId}

	// expectedCounterpaty is the counterparty of the counterparty's channel end
	// (i.e self)
	expectedCounterparty := types.NewCounterparty(portID, "")
	expectedChannel := types.NewChannel(
		types.INIT, order, expectedCounterparty,
		counterpartyHops, counterpartyVersion,
	)

	if err := k.connectionKeeper.VerifyChannelState(
		ctx, connectionEnd, proofHeight, initProof,
		counterparty.PortId, counterparty.ChannelId, expectedChannel,
	); err != nil {
		return "", err
	}

	return channelID, nil
}

// WriteOpenTryChannel writes a channel which has successfully passed the OpenTry handshake step.
// The channel is set in state. If a previous channel state did not exist, all the Send and Recv
// sequences are set to 1. An event is emitted for the handshake step.
func (k *Keeper) WriteOpenTryChannel(
	ctx sdk.Context,
	portID,
	channelID string,
	order types.Order,
	connectionHops []string,
	counterparty types.Counterparty,
	version string,
) {
	k.SetNextSequenceSend(ctx, portID, channelID, 1)
	k.SetNextSequenceRecv(ctx, portID, channelID, 1)
	k.SetNextSequenceAck(ctx, portID, channelID, 1)

	channel := types.NewChannel(types.TRYOPEN, order, counterparty, connectionHops, version)

	k.SetChannel(ctx, portID, channelID, channel)

	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", types.UNINITIALIZED, "new-state", types.TRYOPEN)

	defer telemetry.IncrCounter(1, "ibc", "channel", "open-try")

	emitChannelOpenTryEvent(ctx, portID, channelID, channel)
}

// ChanOpenAck is called by the handshake-originating module to acknowledge the
// acceptance of the initial request by the counterparty module on the other chain.
func (k *Keeper) ChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyVersion,
	counterpartyChannelID string,
	tryProof []byte,
	proofHeight exported.Height,
) error {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if channel.State != types.INIT {
		return errorsmod.Wrapf(types.ErrInvalidChannelState, "channel state should be INIT (got %s)", channel.State)
	}

	connectionEnd, found := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, channel.ConnectionHops[0])
	}

	if connectionEnd.State != connectiontypes.OPEN {
		return errorsmod.Wrapf(connectiontypes.ErrInvalidConnectionState, "connection state is not OPEN (got %s)", connectionEnd.State)
	}

	counterpartyHops := []string{connectionEnd.Counterparty.ConnectionId}

	// counterparty of the counterparty channel end (i.e self)
	expectedCounterparty := types.NewCounterparty(portID, channelID)
	expectedChannel := types.NewChannel(
		types.TRYOPEN, channel.Ordering, expectedCounterparty,
		counterpartyHops, counterpartyVersion,
	)

	return k.connectionKeeper.VerifyChannelState(
		ctx, connectionEnd, proofHeight, tryProof,
		channel.Counterparty.PortId, counterpartyChannelID,
		expectedChannel)
}

// WriteOpenAckChannel writes an updated channel state for the successful OpenAck handshake step.
// An event is emitted for the handshake step.
func (k *Keeper) WriteOpenAckChannel(
	ctx sdk.Context,
	portID,
	channelID,
	counterpartyVersion,
	counterpartyChannelID string,
) {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		panic(fmt.Errorf("could not find existing channel when updating channel state in successful ChanOpenAck step, channelID: %s, portID: %s", channelID, portID))
	}

	channel.State = types.OPEN
	channel.Version = counterpartyVersion
	channel.Counterparty.ChannelId = counterpartyChannelID
	k.SetChannel(ctx, portID, channelID, channel)

	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", types.INIT, "new-state", types.OPEN)

	defer telemetry.IncrCounter(1, "ibc", "channel", "open-ack")

	emitChannelOpenAckEvent(ctx, portID, channelID, channel)
}

// ChanOpenConfirm is called by the handshake-accepting module to confirm the acknowledgement
// of the handshake-originating module on the other chain and finish the channel opening handshake.
func (k *Keeper) ChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
	ackProof []byte,
	proofHeight exported.Height,
) error {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if channel.State != types.TRYOPEN {
		return errorsmod.Wrapf(
			types.ErrInvalidChannelState,
			"channel state is not TRYOPEN (got %s)", channel.State,
		)
	}

	connectionEnd, found := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, channel.ConnectionHops[0])
	}

	if connectionEnd.State != connectiontypes.OPEN {
		return errorsmod.Wrapf(connectiontypes.ErrInvalidConnectionState, "connection state is not OPEN (got %s)", connectionEnd.State)
	}

	counterpartyHops := []string{connectionEnd.Counterparty.ConnectionId}

	counterparty := types.NewCounterparty(portID, channelID)
	expectedChannel := types.NewChannel(
		types.OPEN, channel.Ordering, counterparty,
		counterpartyHops, channel.Version,
	)

	// NOTE: If the counterparty has initialized an upgrade in the same block as performing the
	// ACK handshake step, this channel end will be incapable of opening.
	return k.connectionKeeper.VerifyChannelState(
		ctx, connectionEnd, proofHeight, ackProof,
		channel.Counterparty.PortId, channel.Counterparty.ChannelId,
		expectedChannel)
}

// WriteOpenConfirmChannel writes an updated channel state for the successful OpenConfirm handshake step.
// An event is emitted for the handshake step.
func (k *Keeper) WriteOpenConfirmChannel(
	ctx sdk.Context,
	portID,
	channelID string,
) {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		panic(fmt.Errorf("could not find existing channel when updating channel state in successful ChanOpenConfirm step, channelID: %s, portID: %s", channelID, portID))
	}

	channel.State = types.OPEN
	k.SetChannel(ctx, portID, channelID, channel)
	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", types.TRYOPEN, "new-state", types.OPEN)

	defer telemetry.IncrCounter(1, "ibc", "channel", "open-confirm")

	emitChannelOpenConfirmEvent(ctx, portID, channelID, channel)
}

// Closing Handshake
//
// This section defines the set of functions required to close a channel handshake
// as defined in https://github.com/cosmos/ibc/tree/master/spec/core/ics-004-channel-and-packet-semantics#closing-handshake
//
// ChanCloseInit is called by either module to close their end of the channel. Once
// closed, channels cannot be reopened.
func (k *Keeper) ChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if channel.State == types.CLOSED {
		return errorsmod.Wrap(types.ErrInvalidChannelState, "channel is already CLOSED")
	}

	connectionEnd, found := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, channel.ConnectionHops[0])
	}

	if status := k.clientKeeper.GetClientStatus(ctx, connectionEnd.ClientId); status != exported.Active {
		return errorsmod.Wrapf(clienttypes.ErrClientNotActive, "client (%s) status is %s", connectionEnd.ClientId, status)
	}

	if connectionEnd.State != connectiontypes.OPEN {
		return errorsmod.Wrapf(connectiontypes.ErrInvalidConnectionState, "connection state is not OPEN (got %s)", connectionEnd.State)
	}

	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", channel.State, "new-state", types.CLOSED)

	defer telemetry.IncrCounter(1, "ibc", "channel", "close-init")

	channel.State = types.CLOSED
	k.SetChannel(ctx, portID, channelID, channel)

	emitChannelCloseInitEvent(ctx, portID, channelID, channel)

	return nil
}

// ChanCloseConfirm is called by the counterparty module to close their end of the
// channel, since the other end has been closed.
func (k *Keeper) ChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
	initProof []byte,
	proofHeight exported.Height,
) error {
	channel, found := k.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(types.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if channel.State == types.CLOSED {
		return errorsmod.Wrap(types.ErrInvalidChannelState, "channel is already CLOSED")
	}

	connectionEnd, found := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, channel.ConnectionHops[0])
	}

	if connectionEnd.State != connectiontypes.OPEN {
		return errorsmod.Wrapf(connectiontypes.ErrInvalidConnectionState, "connection state is not OPEN (got %s)", connectionEnd.State)
	}

	counterpartyHops := []string{connectionEnd.Counterparty.ConnectionId}

	counterparty := types.NewCounterparty(portID, channelID)
	expectedChannel := types.Channel{
		State:          types.CLOSED,
		Ordering:       channel.Ordering,
		Counterparty:   counterparty,
		ConnectionHops: counterpartyHops,
		Version:        channel.Version,
	}

	if err := k.connectionKeeper.VerifyChannelState(
		ctx, connectionEnd, proofHeight, initProof,
		channel.Counterparty.PortId, channel.Counterparty.ChannelId,
		expectedChannel,
	); err != nil {
		return err
	}

	k.Logger(ctx).Info("channel state updated", "port-id", portID, "channel-id", channelID, "previous-state", channel.State, "new-state", types.CLOSED)

	defer telemetry.IncrCounter(1, "ibc", "channel", "close-confirm")

	channel.State = types.CLOSED
	k.SetChannel(ctx, portID, channelID, channel)

	emitChannelCloseConfirmEvent(ctx, portID, channelID, channel)

	return nil
}
