package keeper

import (
	"context"
	"errors"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/keeper"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v9/modules/core/05-port/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	"github.com/cosmos/ibc-go/v9/modules/core/internal/telemetry"
	coretypes "github.com/cosmos/ibc-go/v9/modules/core/types"
)

var (
	_ clienttypes.MsgServer     = (*Keeper)(nil)
	_ connectiontypes.MsgServer = (*Keeper)(nil)
	_ channeltypes.MsgServer    = (*Keeper)(nil)
)

// CreateClient defines a rpc handler method for MsgCreateClient.
// NOTE: The raw bytes of the concrete types encoded into protobuf.Any is passed to the client keeper.
// The 02-client handler will route to the appropriate light client module based on client type and it is the responsibility
// of the light client module to unmarshal and interpret the proto encoded bytes.
// Backwards compatibility with older versions of ibc-go is maintained through the light client module reconstructing and encoding
// the expected concrete type to the protobuf.Any for proof verification.
func (k *Keeper) CreateClient(goCtx context.Context, msg *clienttypes.MsgCreateClient) (*clienttypes.MsgCreateClientResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	clientState, err := clienttypes.UnpackClientState(msg.ClientState)
	if err != nil {
		return nil, err
	}

	clientID, err := k.ClientKeeper.CreateClient(ctx, clientState.ClientType(), msg.ClientState.Value, msg.ConsensusState.Value)
	if err != nil {
		return nil, err
	}

	return &clienttypes.MsgCreateClientResponse{ClientId: clientID}, nil
}

// UpdateClient defines a rpc handler method for MsgUpdateClient.
func (k *Keeper) UpdateClient(goCtx context.Context, msg *clienttypes.MsgUpdateClient) (*clienttypes.MsgUpdateClientResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	clientMsg, err := clienttypes.UnpackClientMessage(msg.ClientMessage)
	if err != nil {
		return nil, err
	}

	if err = k.ClientKeeper.UpdateClient(ctx, msg.ClientId, clientMsg); err != nil {
		return nil, err
	}

	return &clienttypes.MsgUpdateClientResponse{}, nil
}

// UpgradeClient defines a rpc handler method for MsgUpgradeClient.
// NOTE: The raw bytes of the concrete types encoded into protobuf.Any is passed to the client keeper.
// The 02-client handler will route to the appropriate light client module based on client identifier and it is the responsibility
// of the light client module to unmarshal and interpret the proto encoded bytes.
// Backwards compatibility with older versions of ibc-go is maintained through the light client module reconstructing and encoding
// the expected concrete type to the protobuf.Any for proof verification.
func (k *Keeper) UpgradeClient(goCtx context.Context, msg *clienttypes.MsgUpgradeClient) (*clienttypes.MsgUpgradeClientResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := k.ClientKeeper.UpgradeClient(
		ctx, msg.ClientId,
		msg.ClientState.Value,
		msg.ConsensusState.Value,
		msg.ProofUpgradeClient,
		msg.ProofUpgradeConsensusState,
	); err != nil {
		return nil, err
	}

	return &clienttypes.MsgUpgradeClientResponse{}, nil
}

// SubmitMisbehaviour defines a rpc handler method for MsgSubmitMisbehaviour.
// Warning: DEPRECATED
// This handler is redundant as `MsgUpdateClient` is now capable of handling both a Header and a Misbehaviour
func (k *Keeper) SubmitMisbehaviour(goCtx context.Context, msg *clienttypes.MsgSubmitMisbehaviour) (*clienttypes.MsgSubmitMisbehaviourResponse, error) { //nolint:staticcheck // for now, we're using msgsubmitmisbehaviour.
	ctx := sdk.UnwrapSDKContext(goCtx)

	misbehaviour, err := clienttypes.UnpackClientMessage(msg.Misbehaviour)
	if err != nil {
		return nil, err
	}

	if err = k.ClientKeeper.UpdateClient(ctx, msg.ClientId, misbehaviour); err != nil {
		return nil, err
	}

	return &clienttypes.MsgSubmitMisbehaviourResponse{}, nil
}

// RecoverClient defines a rpc handler method for MsgRecoverClient.
func (k *Keeper) RecoverClient(goCtx context.Context, msg *clienttypes.MsgRecoverClient) (*clienttypes.MsgRecoverClientResponse, error) {
	if k.GetAuthority() != msg.Signer {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "expected %s, got %s", k.GetAuthority(), msg.Signer)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.ClientKeeper.RecoverClient(ctx, msg.SubjectClientId, msg.SubstituteClientId); err != nil {
		return nil, errorsmod.Wrap(err, "client recovery failed")
	}

	return &clienttypes.MsgRecoverClientResponse{}, nil
}

// IBCSoftwareUpgrade defines a rpc handler method for MsgIBCSoftwareUpgrade.
func (k *Keeper) IBCSoftwareUpgrade(goCtx context.Context, msg *clienttypes.MsgIBCSoftwareUpgrade) (*clienttypes.MsgIBCSoftwareUpgradeResponse, error) {
	if k.GetAuthority() != msg.Signer {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "expected %s, got %s", k.GetAuthority(), msg.Signer)
	}

	upgradedClientState, err := clienttypes.UnpackClientState(msg.UpgradedClientState)
	if err != nil {
		return nil, errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "cannot unpack client state: %s", err)
	}

	if err := k.ClientKeeper.ScheduleIBCSoftwareUpgrade(goCtx, msg.Plan, upgradedClientState); err != nil {
		return nil, errorsmod.Wrap(err, "failed to schedule upgrade")
	}

	return &clienttypes.MsgIBCSoftwareUpgradeResponse{}, nil
}

// ConnectionOpenInit defines a rpc handler method for MsgConnectionOpenInit.
func (k *Keeper) ConnectionOpenInit(goCtx context.Context, msg *connectiontypes.MsgConnectionOpenInit) (*connectiontypes.MsgConnectionOpenInitResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if _, err := k.ConnectionKeeper.ConnOpenInit(ctx, msg.ClientId, msg.Counterparty, msg.Version, msg.DelayPeriod); err != nil {
		return nil, errorsmod.Wrap(err, "connection handshake open init failed")
	}

	return &connectiontypes.MsgConnectionOpenInitResponse{}, nil
}

// ConnectionOpenTry defines a rpc handler method for MsgConnectionOpenTry.
func (k *Keeper) ConnectionOpenTry(goCtx context.Context, msg *connectiontypes.MsgConnectionOpenTry) (*connectiontypes.MsgConnectionOpenTryResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if _, err := k.ConnectionKeeper.ConnOpenTry(
		ctx, msg.Counterparty, msg.DelayPeriod, msg.ClientId,
		msg.CounterpartyVersions, msg.ProofInit, msg.ProofHeight,
	); err != nil {
		return nil, errorsmod.Wrap(err, "connection handshake open try failed")
	}

	return &connectiontypes.MsgConnectionOpenTryResponse{}, nil
}

// ConnectionOpenAck defines a rpc handler method for MsgConnectionOpenAck.
func (k *Keeper) ConnectionOpenAck(goCtx context.Context, msg *connectiontypes.MsgConnectionOpenAck) (*connectiontypes.MsgConnectionOpenAckResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := k.ConnectionKeeper.ConnOpenAck(
		ctx, msg.ConnectionId, msg.Version, msg.CounterpartyConnectionId,
		msg.ProofTry, msg.ProofHeight,
	); err != nil {
		return nil, errorsmod.Wrap(err, "connection handshake open ack failed")
	}

	return &connectiontypes.MsgConnectionOpenAckResponse{}, nil
}

// ConnectionOpenConfirm defines a rpc handler method for MsgConnectionOpenConfirm.
func (k *Keeper) ConnectionOpenConfirm(goCtx context.Context, msg *connectiontypes.MsgConnectionOpenConfirm) (*connectiontypes.MsgConnectionOpenConfirmResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := k.ConnectionKeeper.ConnOpenConfirm(
		ctx, msg.ConnectionId, msg.ProofAck, msg.ProofHeight,
	); err != nil {
		return nil, errorsmod.Wrap(err, "connection handshake open confirm failed")
	}

	return &connectiontypes.MsgConnectionOpenConfirmResponse{}, nil
}

// ChannelOpenInit defines a rpc handler method for MsgChannelOpenInit.
// ChannelOpenInit will perform 04-channel checks, route to the application
// callback, and write an OpenInit channel into state upon successful execution.
func (k *Keeper) ChannelOpenInit(goCtx context.Context, msg *channeltypes.MsgChannelOpenInit) (*channeltypes.MsgChannelOpenInitResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Retrieve application callbacks from router
	cbs, ok := k.PortKeeper.Route(msg.PortId)
	if !ok {
		ctx.Logger().Error("channel open init failed", "port-id", msg.PortId, "error", errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.PortId))
		return nil, errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.PortId)
	}

	// Perform 04-channel verification
	channelID, err := k.ChannelKeeper.ChanOpenInit(
		ctx, msg.Channel.Ordering, msg.Channel.ConnectionHops, msg.PortId, msg.Channel.Counterparty, msg.Channel.Version,
	)
	if err != nil {
		ctx.Logger().Error("channel open init failed", "error", errorsmod.Wrap(err, "channel handshake open init failed"))
		return nil, errorsmod.Wrap(err, "channel handshake open init failed")
	}

	// Perform application logic callback
	version, err := cbs.OnChanOpenInit(ctx, msg.Channel.Ordering, msg.Channel.ConnectionHops, msg.PortId, channelID, msg.Channel.Counterparty, msg.Channel.Version)
	if err != nil {
		ctx.Logger().Error("channel open init failed", "port-id", msg.PortId, "channel-id", channelID, "error", errorsmod.Wrap(err, "channel open init callback failed"))
		return nil, errorsmod.Wrapf(err, "channel open init callback failed for port ID: %s, channel ID: %s", msg.PortId, channelID)
	}

	// Write channel into state
	k.ChannelKeeper.WriteOpenInitChannel(ctx, msg.PortId, channelID, msg.Channel.Ordering, msg.Channel.ConnectionHops, msg.Channel.Counterparty, version)

	ctx.Logger().Info("channel open init succeeded", "channel-id", channelID, "version", version)

	return &channeltypes.MsgChannelOpenInitResponse{
		ChannelId: channelID,
		Version:   version,
	}, nil
}

// ChannelOpenTry defines a rpc handler method for MsgChannelOpenTry.
// ChannelOpenTry will perform 04-channel checks, route to the application
// callback, and write an OpenTry channel into state upon successful execution.
func (k *Keeper) ChannelOpenTry(goCtx context.Context, msg *channeltypes.MsgChannelOpenTry) (*channeltypes.MsgChannelOpenTryResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Retrieve application callbacks from router
	cbs, ok := k.PortKeeper.Route(msg.PortId)
	if !ok {
		ctx.Logger().Error("channel open try failed", "port-id", msg.PortId, "error", errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.PortId))
		return nil, errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.PortId)
	}

	// Perform 04-channel verification
	channelID, err := k.ChannelKeeper.ChanOpenTry(ctx, msg.Channel.Ordering, msg.Channel.ConnectionHops, msg.PortId, msg.Channel.Counterparty, msg.CounterpartyVersion, msg.ProofInit, msg.ProofHeight)
	if err != nil {
		ctx.Logger().Error("channel open try failed", "error", errorsmod.Wrap(err, "channel handshake open try failed"))
		return nil, errorsmod.Wrap(err, "channel handshake open try failed")
	}

	// Perform application logic callback
	version, err := cbs.OnChanOpenTry(ctx, msg.Channel.Ordering, msg.Channel.ConnectionHops, msg.PortId, channelID, msg.Channel.Counterparty, msg.CounterpartyVersion)
	if err != nil {
		ctx.Logger().Error("channel open try failed", "port-id", msg.PortId, "channel-id", channelID, "error", errorsmod.Wrap(err, "channel open try callback failed"))
		return nil, errorsmod.Wrapf(err, "channel open try callback failed for port ID: %s, channel ID: %s", msg.PortId, channelID)
	}

	// Write channel into state
	k.ChannelKeeper.WriteOpenTryChannel(ctx, msg.PortId, channelID, msg.Channel.Ordering, msg.Channel.ConnectionHops, msg.Channel.Counterparty, version)

	ctx.Logger().Info("channel open try succeeded", "channel-id", channelID, "port-id", msg.PortId, "version", version)

	return &channeltypes.MsgChannelOpenTryResponse{
		ChannelId: channelID,
		Version:   version,
	}, nil
}

// ChannelOpenAck defines a rpc handler method for MsgChannelOpenAck.
// ChannelOpenAck will perform 04-channel checks, route to the application
// callback, and write an OpenAck channel into state upon successful execution.
func (k *Keeper) ChannelOpenAck(goCtx context.Context, msg *channeltypes.MsgChannelOpenAck) (*channeltypes.MsgChannelOpenAckResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Retrieve application callbacks from router
	cbs, ok := k.PortKeeper.Route(msg.PortId)
	if !ok {
		ctx.Logger().Error("channel open ack failed", "port-id", msg.PortId, "error", errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.PortId))
		return nil, errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.PortId)
	}

	// Perform 04-channel verification
	if err := k.ChannelKeeper.ChanOpenAck(
		ctx, msg.PortId, msg.ChannelId, msg.CounterpartyVersion, msg.CounterpartyChannelId, msg.ProofTry, msg.ProofHeight,
	); err != nil {
		ctx.Logger().Error("channel open ack failed", "error", err.Error())
		return nil, errorsmod.Wrap(err, "channel handshake open ack failed")
	}

	// Write channel into state
	k.ChannelKeeper.WriteOpenAckChannel(ctx, msg.PortId, msg.ChannelId, msg.CounterpartyVersion, msg.CounterpartyChannelId)

	// Perform application logic callback
	if err := cbs.OnChanOpenAck(ctx, msg.PortId, msg.ChannelId, msg.CounterpartyChannelId, msg.CounterpartyVersion); err != nil {
		ctx.Logger().Error("channel open ack failed", "port-id", msg.PortId, "channel-id", msg.ChannelId, "error", errorsmod.Wrap(err, "channel open ack callback failed"))
		return nil, errorsmod.Wrapf(err, "channel open ack callback failed for port ID: %s, channel ID: %s", msg.PortId, msg.ChannelId)
	}

	ctx.Logger().Info("channel open ack succeeded", "channel-id", msg.ChannelId, "port-id", msg.PortId)

	return &channeltypes.MsgChannelOpenAckResponse{}, nil
}

// ChannelOpenConfirm defines a rpc handler method for MsgChannelOpenConfirm.
// ChannelOpenConfirm will perform 04-channel checks, route to the application
// callback, and write an OpenConfirm channel into state upon successful execution.
func (k *Keeper) ChannelOpenConfirm(goCtx context.Context, msg *channeltypes.MsgChannelOpenConfirm) (*channeltypes.MsgChannelOpenConfirmResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Retrieve application callbacks from router
	cbs, ok := k.PortKeeper.Route(msg.PortId)
	if !ok {
		ctx.Logger().Error("channel open confirm failed", "port-id", msg.PortId, "error", errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.PortId))
		return nil, errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.PortId)
	}

	// Perform 04-channel verification
	if err := k.ChannelKeeper.ChanOpenConfirm(ctx, msg.PortId, msg.ChannelId, msg.ProofAck, msg.ProofHeight); err != nil {
		ctx.Logger().Error("channel open confirm failed", "error", errorsmod.Wrap(err, "channel handshake open confirm failed"))
		return nil, errorsmod.Wrap(err, "channel handshake open confirm failed")
	}

	// Write channel into state
	k.ChannelKeeper.WriteOpenConfirmChannel(ctx, msg.PortId, msg.ChannelId)

	// Perform application logic callback
	if err := cbs.OnChanOpenConfirm(ctx, msg.PortId, msg.ChannelId); err != nil {
		ctx.Logger().Error("channel open confirm failed", "port-id", msg.PortId, "channel-id", msg.ChannelId, "error", errorsmod.Wrap(err, "channel open confirm callback failed"))
		return nil, errorsmod.Wrapf(err, "channel open confirm callback failed for port ID: %s, channel ID: %s", msg.PortId, msg.ChannelId)
	}

	ctx.Logger().Info("channel open confirm succeeded", "channel-id", msg.ChannelId, "port-id", msg.PortId)

	return &channeltypes.MsgChannelOpenConfirmResponse{}, nil
}

// ChannelCloseInit defines a rpc handler method for MsgChannelCloseInit.
func (k *Keeper) ChannelCloseInit(goCtx context.Context, msg *channeltypes.MsgChannelCloseInit) (*channeltypes.MsgChannelCloseInitResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Retrieve callbacks from router
	cbs, ok := k.PortKeeper.Route(msg.PortId)
	if !ok {
		ctx.Logger().Error("channel close init failed", "port-id", msg.PortId, "error", errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.PortId))
		return nil, errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.PortId)
	}

	if err := cbs.OnChanCloseInit(ctx, msg.PortId, msg.ChannelId); err != nil {
		ctx.Logger().Error("channel close init failed", "port-id", msg.PortId, "channel-id", msg.ChannelId, "error", errorsmod.Wrap(err, "channel close init callback failed"))
		return nil, errorsmod.Wrapf(err, "channel close init callback failed for port ID: %s, channel ID: %s", msg.PortId, msg.ChannelId)
	}

	err := k.ChannelKeeper.ChanCloseInit(ctx, msg.PortId, msg.ChannelId)
	if err != nil {
		ctx.Logger().Error("channel close init failed", "port-id", msg.PortId, "channel-id", msg.ChannelId, "error", err.Error())
		return nil, errorsmod.Wrap(err, "channel handshake close init failed")
	}

	ctx.Logger().Info("channel close init succeeded", "channel-id", msg.ChannelId, "port-id", msg.PortId)

	return &channeltypes.MsgChannelCloseInitResponse{}, nil
}

// ChannelCloseConfirm defines a rpc handler method for MsgChannelCloseConfirm.
func (k *Keeper) ChannelCloseConfirm(goCtx context.Context, msg *channeltypes.MsgChannelCloseConfirm) (*channeltypes.MsgChannelCloseConfirmResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Retrieve callbacks from router
	cbs, ok := k.PortKeeper.Route(msg.PortId)
	if !ok {
		ctx.Logger().Error("channel close confirm failed", "port-id", msg.PortId, "error", errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.PortId))
		return nil, errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.PortId)
	}

	if err := cbs.OnChanCloseConfirm(ctx, msg.PortId, msg.ChannelId); err != nil {
		ctx.Logger().Error("channel close confirm failed", "port-id", msg.PortId, "channel-id", msg.ChannelId, "error", errorsmod.Wrap(err, "channel close confirm callback failed"))
		return nil, errorsmod.Wrapf(err, "channel close confirm callback failed for port ID: %s, channel ID: %s", msg.PortId, msg.ChannelId)
	}

	err := k.ChannelKeeper.ChanCloseConfirm(ctx, msg.PortId, msg.ChannelId, msg.ProofInit, msg.ProofHeight, msg.CounterpartyUpgradeSequence)
	if err != nil {
		ctx.Logger().Error("channel close confirm failed", "port-id", msg.PortId, "channel-id", msg.ChannelId, "error", err.Error())
		return nil, errorsmod.Wrap(err, "channel handshake close confirm failed")
	}

	ctx.Logger().Info("channel close confirm succeeded", "channel-id", msg.ChannelId, "port-id", msg.PortId)

	return &channeltypes.MsgChannelCloseConfirmResponse{}, nil
}

// RecvPacket defines a rpc handler method for MsgRecvPacket.
func (k *Keeper) RecvPacket(goCtx context.Context, msg *channeltypes.MsgRecvPacket) (*channeltypes.MsgRecvPacketResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	relayer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		ctx.Logger().Error("receive packet failed", "error", errorsmod.Wrap(err, "Invalid address for msg Signer"))
		return nil, errorsmod.Wrap(err, "Invalid address for msg Signer")
	}

	// Retrieve callbacks from router
	cbs, ok := k.PortKeeper.Route(msg.Packet.DestinationPort)
	if !ok {
		ctx.Logger().Error("receive packet failed", "port-id", msg.Packet.SourcePort, "error", errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.Packet.DestinationPort))
		return nil, errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.Packet.DestinationPort)
	}

	// Perform TAO verification
	//
	// If the packet was already received, perform a no-op
	// Use a cached context to prevent accidental state changes
	cacheCtx, writeFn := ctx.CacheContext()
	channelVersion, err := k.ChannelKeeper.RecvPacket(cacheCtx, msg.Packet, msg.ProofCommitment, msg.ProofHeight)

	switch err {
	case nil:
		writeFn()
	case channeltypes.ErrNoOpMsg:
		// no-ops do not need event emission as they will be ignored
		ctx.Logger().Debug("no-op on redundant relay", "port-id", msg.Packet.SourcePort, "channel-id", msg.Packet.SourceChannel)
		return &channeltypes.MsgRecvPacketResponse{Result: channeltypes.NOOP}, nil
	default:
		ctx.Logger().Error("receive packet failed", "port-id", msg.Packet.SourcePort, "channel-id", msg.Packet.SourceChannel, "error", errorsmod.Wrap(err, "receive packet verification failed"))
		return nil, errorsmod.Wrap(err, "receive packet verification failed")
	}

	// Perform application logic callback
	//
	// Cache context so that we may discard state changes from callback if the acknowledgement is unsuccessful.
	cacheCtx, writeFn = ctx.CacheContext()
	ack := cbs.OnRecvPacket(cacheCtx, channelVersion, msg.Packet, relayer)
	if ack == nil || ack.Success() {
		// write application state changes for asynchronous and successful acknowledgements
		writeFn()
	} else {
		// Modify events in cached context to reflect unsuccessful acknowledgement
		ctx.EventManager().EmitEvents(convertToErrorEvents(cacheCtx.EventManager().Events()))
	}

	// Set packet acknowledgement only if the acknowledgement is not nil.
	// NOTE: IBC applications modules may call the WriteAcknowledgement asynchronously if the
	// acknowledgement is nil.
	if ack != nil {
		if err := k.ChannelKeeper.WriteAcknowledgement(ctx, msg.Packet, ack); err != nil {
			return nil, err
		}
	}

	defer telemetry.ReportRecvPacket(msg.Packet)

	ctx.Logger().Info("receive packet callback succeeded", "port-id", msg.Packet.SourcePort, "channel-id", msg.Packet.SourceChannel, "result", channeltypes.SUCCESS.String())

	return &channeltypes.MsgRecvPacketResponse{Result: channeltypes.SUCCESS}, nil
}

// Timeout defines a rpc handler method for MsgTimeout.
func (k *Keeper) Timeout(goCtx context.Context, msg *channeltypes.MsgTimeout) (*channeltypes.MsgTimeoutResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	relayer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		ctx.Logger().Error("timeout failed", "error", errorsmod.Wrap(err, "Invalid address for msg Signer"))
		return nil, errorsmod.Wrap(err, "Invalid address for msg Signer")
	}

	// Retrieve callbacks from router
	cbs, ok := k.PortKeeper.Route(msg.Packet.SourcePort)
	if !ok {
		ctx.Logger().Error("timeout failed", "port-id", msg.Packet.SourcePort, "error", errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.Packet.SourcePort))
		return nil, errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.Packet.SourcePort)
	}

	// Perform TAO verification
	//
	// If the timeout was already received, perform a no-op
	// Use a cached context to prevent accidental state changes
	cacheCtx, writeFn := ctx.CacheContext()
	channelVersion, err := k.ChannelKeeper.TimeoutPacket(cacheCtx, msg.Packet, msg.ProofUnreceived, msg.ProofHeight, msg.NextSequenceRecv)

	switch err {
	case nil:
		writeFn()
	case channeltypes.ErrNoOpMsg:
		// no-ops do not need event emission as they will be ignored
		ctx.Logger().Debug("no-op on redundant relay", "port-id", msg.Packet.SourcePort, "channel-id", msg.Packet.SourceChannel)
		return &channeltypes.MsgTimeoutResponse{Result: channeltypes.NOOP}, nil
	default:
		ctx.Logger().Error("timeout failed", "port-id", msg.Packet.SourcePort, "channel-id", msg.Packet.SourceChannel, "error", errorsmod.Wrap(err, "timeout packet verification failed"))
		return nil, errorsmod.Wrap(err, "timeout packet verification failed")
	}

	// Delete packet commitment
	if err = k.ChannelKeeper.TimeoutExecuted(ctx, msg.Packet); err != nil {
		return nil, err
	}

	// Perform application logic callback
	err = cbs.OnTimeoutPacket(ctx, channelVersion, msg.Packet, relayer)
	if err != nil {
		ctx.Logger().Error("timeout failed", "port-id", msg.Packet.SourcePort, "channel-id", msg.Packet.SourceChannel, "error", errorsmod.Wrap(err, "timeout packet callback failed"))
		return nil, errorsmod.Wrap(err, "timeout packet callback failed")
	}

	defer telemetry.ReportTimeoutPacket(msg.Packet, "height")

	ctx.Logger().Info("timeout packet callback succeeded", "port-id", msg.Packet.SourcePort, "channel-id", msg.Packet.SourceChannel, "result", channeltypes.SUCCESS.String())

	return &channeltypes.MsgTimeoutResponse{Result: channeltypes.SUCCESS}, nil
}

// TimeoutOnClose defines a rpc handler method for MsgTimeoutOnClose.
func (k *Keeper) TimeoutOnClose(goCtx context.Context, msg *channeltypes.MsgTimeoutOnClose) (*channeltypes.MsgTimeoutOnCloseResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	relayer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		ctx.Logger().Error("timeout on close failed", "error", errorsmod.Wrap(err, "Invalid address for msg Signer"))
		return nil, errorsmod.Wrap(err, "Invalid address for msg Signer")
	}

	cbs, ok := k.PortKeeper.Route(msg.Packet.SourcePort)
	if !ok {
		ctx.Logger().Error("timeout on close failed", "port-id", msg.Packet.SourcePort, "error", errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.Packet.SourcePort))
		return nil, errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.Packet.SourcePort)
	}

	// Perform TAO verification
	//
	// If the timeout was already received, perform a no-op
	// Use a cached context to prevent accidental state changes
	cacheCtx, writeFn := ctx.CacheContext()
	channelVersion, err := k.ChannelKeeper.TimeoutOnClose(cacheCtx, msg.Packet, msg.ProofUnreceived, msg.ProofClose, msg.ProofHeight, msg.NextSequenceRecv, msg.CounterpartyUpgradeSequence)

	switch err {
	case nil:
		writeFn()
	case channeltypes.ErrNoOpMsg:
		// no-ops do not need event emission as they will be ignored
		ctx.Logger().Debug("no-op on redundant relay", "port-id", msg.Packet.SourcePort, "channel-id", msg.Packet.SourceChannel)
		return &channeltypes.MsgTimeoutOnCloseResponse{Result: channeltypes.NOOP}, nil
	default:
		ctx.Logger().Error("timeout on close failed", "port-id", msg.Packet.SourcePort, "channel-id", msg.Packet.SourceChannel, "error", errorsmod.Wrap(err, "timeout on close packet verification failed"))
		return nil, errorsmod.Wrap(err, "timeout on close packet verification failed")
	}

	// Delete packet commitment
	if err = k.ChannelKeeper.TimeoutExecuted(ctx, msg.Packet); err != nil {
		return nil, err
	}

	// Perform application logic callback
	//
	// NOTE: MsgTimeout and MsgTimeoutOnClose use the same "OnTimeoutPacket"
	// application logic callback.
	err = cbs.OnTimeoutPacket(ctx, channelVersion, msg.Packet, relayer)
	if err != nil {
		ctx.Logger().Error("timeout on close failed", "port-id", msg.Packet.SourcePort, "channel-id", msg.Packet.SourceChannel, "error", errorsmod.Wrap(err, "timeout on close callback failed"))
		return nil, errorsmod.Wrap(err, "timeout on close callback failed")
	}

	defer telemetry.ReportTimeoutPacket(msg.Packet, "channel-closed")

	ctx.Logger().Info("timeout on close callback succeeded", "port-id", msg.Packet.SourcePort, "channel-id", msg.Packet.SourceChannel, "result", channeltypes.SUCCESS.String())

	return &channeltypes.MsgTimeoutOnCloseResponse{Result: channeltypes.SUCCESS}, nil
}

// Acknowledgement defines a rpc handler method for MsgAcknowledgement.
func (k *Keeper) Acknowledgement(goCtx context.Context, msg *channeltypes.MsgAcknowledgement) (*channeltypes.MsgAcknowledgementResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	relayer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		ctx.Logger().Error("acknowledgement failed", "error", errorsmod.Wrap(err, "Invalid address for msg Signer"))
		return nil, errorsmod.Wrap(err, "Invalid address for msg Signer")
	}

	cbs, ok := k.PortKeeper.Route(msg.Packet.SourcePort)
	if !ok {
		ctx.Logger().Error("acknowledgement failed", "port-id", msg.Packet.SourcePort, "error", errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.Packet.SourcePort))
		return nil, errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.Packet.SourcePort)
	}

	// Perform TAO verification
	//
	// If the acknowledgement was already received, perform a no-op
	// Use a cached context to prevent accidental state changes
	cacheCtx, writeFn := ctx.CacheContext()
	channelVersion, err := k.ChannelKeeper.AcknowledgePacket(cacheCtx, msg.Packet, msg.Acknowledgement, msg.ProofAcked, msg.ProofHeight)

	switch err {
	case nil:
		writeFn()
	case channeltypes.ErrNoOpMsg:
		// no-ops do not need event emission as they will be ignored
		ctx.Logger().Debug("no-op on redundant relay", "port-id", msg.Packet.SourcePort, "channel-id", msg.Packet.SourceChannel)
		return &channeltypes.MsgAcknowledgementResponse{Result: channeltypes.NOOP}, nil
	default:
		ctx.Logger().Error("acknowledgement failed", "port-id", msg.Packet.SourcePort, "channel-id", msg.Packet.SourceChannel, "error", errorsmod.Wrap(err, "acknowledge packet verification failed"))
		return nil, errorsmod.Wrap(err, "acknowledge packet verification failed")
	}

	// Perform application logic callback
	err = cbs.OnAcknowledgementPacket(ctx, channelVersion, msg.Packet, msg.Acknowledgement, relayer)
	if err != nil {
		ctx.Logger().Error("acknowledgement failed", "port-id", msg.Packet.SourcePort, "channel-id", msg.Packet.SourceChannel, "error", errorsmod.Wrap(err, "acknowledge packet callback failed"))
		return nil, errorsmod.Wrap(err, "acknowledge packet callback failed")
	}

	defer telemetry.ReportAcknowledgePacket(msg.Packet)

	ctx.Logger().Info("acknowledgement succeeded", "port-id", msg.Packet.SourcePort, "channel-id", msg.Packet.SourceChannel, "result", channeltypes.SUCCESS.String())

	return &channeltypes.MsgAcknowledgementResponse{Result: channeltypes.SUCCESS}, nil
}

// ChannelUpgradeInit defines a rpc handler method for MsgChannelUpgradeInit.
func (k *Keeper) ChannelUpgradeInit(goCtx context.Context, msg *channeltypes.MsgChannelUpgradeInit) (*channeltypes.MsgChannelUpgradeInitResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if k.GetAuthority() != msg.Signer {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "expected %s, got %s", k.GetAuthority(), msg.Signer)
	}

	app, ok := k.PortKeeper.Route(msg.PortId)
	if !ok {
		ctx.Logger().Error("channel upgrade init failed", "port-id", msg.PortId, "error", errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.PortId))
		return nil, errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.PortId)
	}

	cbs, ok := app.(porttypes.UpgradableModule)
	if !ok {
		ctx.Logger().Error("channel upgrade init failed", "port-id", msg.PortId, "error", errorsmod.Wrapf(porttypes.ErrInvalidRoute, "upgrade route not found to portID: %s", msg.PortId))
		return nil, errorsmod.Wrapf(porttypes.ErrInvalidRoute, "upgrade route not found to portID: %s", msg.PortId)
	}

	upgrade, err := k.ChannelKeeper.ChanUpgradeInit(ctx, msg.PortId, msg.ChannelId, msg.Fields)
	if err != nil {
		ctx.Logger().Error("channel upgrade init failed", "error", errorsmod.Wrap(err, "channel upgrade init failed"))
		return nil, errorsmod.Wrap(err, "channel upgrade init failed")
	}

	// NOTE: a cached context is used to discard ibc application state changes and events.
	// IBC applications must flush in-flight packets using the pre-upgrade channel parameters.
	cacheCtx, _ := ctx.CacheContext()
	upgradeVersion, err := cbs.OnChanUpgradeInit(cacheCtx, msg.PortId, msg.ChannelId, upgrade.Fields.Ordering, upgrade.Fields.ConnectionHops, upgrade.Fields.Version)
	if err != nil {
		ctx.Logger().Error("channel upgrade init callback failed", "port-id", msg.PortId, "channel-id", msg.ChannelId, "error", err.Error())
		return nil, errorsmod.Wrapf(err, "channel upgrade init callback failed for port ID: %s, channel ID: %s", msg.PortId, msg.ChannelId)
	}

	channel, upgrade := k.ChannelKeeper.WriteUpgradeInitChannel(ctx, msg.PortId, msg.ChannelId, upgrade, upgradeVersion)

	ctx.Logger().Info("channel upgrade init succeeded", "channel-id", msg.ChannelId, "version", upgradeVersion)
	keeper.EmitChannelUpgradeInitEvent(ctx, msg.PortId, msg.ChannelId, channel, upgrade)

	return &channeltypes.MsgChannelUpgradeInitResponse{
		Upgrade:         upgrade,
		UpgradeSequence: channel.UpgradeSequence,
	}, nil
}

// ChannelUpgradeTry defines a rpc handler method for MsgChannelUpgradeTry.
func (k *Keeper) ChannelUpgradeTry(goCtx context.Context, msg *channeltypes.MsgChannelUpgradeTry) (*channeltypes.MsgChannelUpgradeTryResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	app, ok := k.PortKeeper.Route(msg.PortId)
	if !ok {
		ctx.Logger().Error("channel upgrade try failed", "port-id", msg.PortId, "error", errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.PortId))
		return nil, errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.PortId)
	}

	cbs, ok := app.(porttypes.UpgradableModule)
	if !ok {
		ctx.Logger().Error("channel upgrade try failed", "port-id", msg.PortId, "error", errorsmod.Wrapf(porttypes.ErrInvalidRoute, "upgrade route not found to portID: %s", msg.PortId))
		return nil, errorsmod.Wrapf(porttypes.ErrInvalidRoute, "upgrade route not found to portID: %s", msg.PortId)
	}

	channel, upgrade, err := k.ChannelKeeper.ChanUpgradeTry(ctx, msg.PortId, msg.ChannelId, msg.ProposedUpgradeConnectionHops, msg.CounterpartyUpgradeFields, msg.CounterpartyUpgradeSequence, msg.ProofChannel, msg.ProofUpgrade, msg.ProofHeight)
	if err != nil {
		ctx.Logger().Error("channel upgrade try failed", "error", errorsmod.Wrap(err, "channel upgrade try failed"))
		if channeltypes.IsUpgradeError(err) {
			// In case the error is a wrapped upgrade error, we need to extract the inner error else process as normal
			var upgradeErr *channeltypes.UpgradeError
			if errors.As(err, &upgradeErr) {
				k.ChannelKeeper.WriteErrorReceipt(ctx, msg.PortId, msg.ChannelId, upgradeErr)
				// NOTE: a FAILURE result is returned to the client and an error receipt is written to state.
				// This signals to the relayer to begin the cancel upgrade handshake subprotocol.
				return &channeltypes.MsgChannelUpgradeTryResponse{Result: channeltypes.FAILURE}, nil
			}
		}

		// NOTE: an error is returned to baseapp and transaction state is not committed.
		return nil, errorsmod.Wrap(err, "channel upgrade try failed")
	}

	// NOTE: a cached context is used to discard ibc application state changes and events.
	// IBC applications must flush in-flight packets using the pre-upgrade channel parameters.
	cacheCtx, _ := ctx.CacheContext()
	upgradeVersion, err := cbs.OnChanUpgradeTry(cacheCtx, msg.PortId, msg.ChannelId, upgrade.Fields.Ordering, upgrade.Fields.ConnectionHops, upgrade.Fields.Version)
	if err != nil {
		ctx.Logger().Error("channel upgrade try callback failed", "port-id", msg.PortId, "channel-id", msg.ChannelId, "error", err.Error())
		return nil, errorsmod.Wrapf(err, "channel upgrade try callback failed for port ID: %s, channel ID: %s", msg.PortId, msg.ChannelId)
	}

	channel, upgrade = k.ChannelKeeper.WriteUpgradeTryChannel(ctx, msg.PortId, msg.ChannelId, upgrade, upgradeVersion)

	ctx.Logger().Info("channel upgrade try succeeded", "port-id", msg.PortId, "channel-id", msg.ChannelId)
	keeper.EmitChannelUpgradeTryEvent(ctx, msg.PortId, msg.ChannelId, channel, upgrade)

	return &channeltypes.MsgChannelUpgradeTryResponse{
		Result:          channeltypes.SUCCESS,
		Upgrade:         upgrade,
		UpgradeSequence: channel.UpgradeSequence,
	}, nil
}

// ChannelUpgradeAck defines a rpc handler method for MsgChannelUpgradeAck.
func (k *Keeper) ChannelUpgradeAck(goCtx context.Context, msg *channeltypes.MsgChannelUpgradeAck) (*channeltypes.MsgChannelUpgradeAckResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	app, ok := k.PortKeeper.Route(msg.PortId)
	if !ok {
		err := errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.PortId)
		ctx.Logger().Error("channel upgrade ack failed", "port-id", msg.PortId, "error", err)
		return nil, err
	}

	cbs, ok := app.(porttypes.UpgradableModule)
	if !ok {
		err := errorsmod.Wrapf(porttypes.ErrInvalidRoute, "upgrade route not found to portID: %s", msg.PortId)
		ctx.Logger().Error("channel upgrade ack failed", "port-id", msg.PortId, "error", err)
		return nil, err
	}

	err := k.ChannelKeeper.ChanUpgradeAck(ctx, msg.PortId, msg.ChannelId, msg.CounterpartyUpgrade, msg.ProofChannel, msg.ProofUpgrade, msg.ProofHeight)
	if err != nil {
		ctx.Logger().Error("channel upgrade ack failed", "error", errorsmod.Wrap(err, "channel upgrade ack failed"))
		if channeltypes.IsUpgradeError(err) {
			k.ChannelKeeper.MustAbortUpgrade(ctx, msg.PortId, msg.ChannelId, err)

			// NOTE: a FAILURE result is returned to the client and an error receipt is written to state.
			// This signals to the relayer to begin the cancel upgrade handshake subprotocol.
			return &channeltypes.MsgChannelUpgradeAckResponse{Result: channeltypes.FAILURE}, nil
		}

		// NOTE: an error is returned to baseapp and transaction state is not committed.
		return nil, errorsmod.Wrap(err, "channel upgrade ack failed")
	}

	// NOTE: a cached context is used to discard ibc application state changes and events.
	// IBC applications must flush in-flight packets using the pre-upgrade channel parameters.
	cacheCtx, _ := ctx.CacheContext()
	err = cbs.OnChanUpgradeAck(cacheCtx, msg.PortId, msg.ChannelId, msg.CounterpartyUpgrade.Fields.Version)
	if err != nil {
		channel, found := k.ChannelKeeper.GetChannel(ctx, msg.PortId, msg.ChannelId)
		if !found {
			return nil, errorsmod.Wrapf(channeltypes.ErrChannelNotFound, "channel not found for port ID (%s) channel ID (%s)", msg.PortId, msg.ChannelId)
		}

		ctx.Logger().Error("channel upgrade ack callback failed", "port-id", msg.PortId, "channel-id", msg.ChannelId, "error", err.Error())

		// explicitly wrap the application callback in an upgrade error with the correct upgrade sequence.
		// this prevents any errors caused from the application returning an UpgradeError with an incorrect sequence.
		k.ChannelKeeper.MustAbortUpgrade(ctx, msg.PortId, msg.ChannelId, channeltypes.NewUpgradeError(channel.UpgradeSequence, err))

		return &channeltypes.MsgChannelUpgradeAckResponse{Result: channeltypes.FAILURE}, nil
	}

	channel, upgrade := k.ChannelKeeper.WriteUpgradeAckChannel(ctx, msg.PortId, msg.ChannelId, msg.CounterpartyUpgrade)

	ctx.Logger().Info("channel upgrade ack succeeded", "port-id", msg.PortId, "channel-id", msg.ChannelId)
	keeper.EmitChannelUpgradeAckEvent(ctx, msg.PortId, msg.ChannelId, channel, upgrade)

	return &channeltypes.MsgChannelUpgradeAckResponse{Result: channeltypes.SUCCESS}, nil
}

// ChannelUpgradeConfirm defines a rpc handler method for MsgChannelUpgradeConfirm.
func (k *Keeper) ChannelUpgradeConfirm(goCtx context.Context, msg *channeltypes.MsgChannelUpgradeConfirm) (*channeltypes.MsgChannelUpgradeConfirmResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	app, ok := k.PortKeeper.Route(msg.PortId)
	if !ok {
		err := errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.PortId)
		ctx.Logger().Error("channel upgrade confirm failed", "port-id", msg.PortId, "error", err)
		return nil, err
	}

	cbs, ok := app.(porttypes.UpgradableModule)
	if !ok {
		err := errorsmod.Wrapf(porttypes.ErrInvalidRoute, "upgrade route not found to portID: %s", msg.PortId)
		ctx.Logger().Error("channel upgrade confirm failed", "port-id", msg.PortId, "error", err)
		return nil, err
	}

	err := k.ChannelKeeper.ChanUpgradeConfirm(ctx, msg.PortId, msg.ChannelId, msg.CounterpartyChannelState, msg.CounterpartyUpgrade, msg.ProofChannel, msg.ProofUpgrade, msg.ProofHeight)
	if err != nil {
		ctx.Logger().Error("channel upgrade confirm failed", "error", errorsmod.Wrap(err, "channel upgrade confirm failed"))
		if channeltypes.IsUpgradeError(err) {
			k.ChannelKeeper.MustAbortUpgrade(ctx, msg.PortId, msg.ChannelId, err)

			// NOTE: a FAILURE result is returned to the client and an error receipt is written to state.
			// This signals to the relayer to begin the cancel upgrade handshake subprotocol.
			return &channeltypes.MsgChannelUpgradeConfirmResponse{Result: channeltypes.FAILURE}, nil
		}

		// NOTE: an error is returned to baseapp and transaction state is not committed.
		return nil, errorsmod.Wrap(err, "channel upgrade confirm failed")
	}

	channel := k.ChannelKeeper.WriteUpgradeConfirmChannel(ctx, msg.PortId, msg.ChannelId, msg.CounterpartyUpgrade)
	ctx.Logger().Info("channel upgrade confirm succeeded", "port-id", msg.PortId, "channel-id", msg.ChannelId)
	keeper.EmitChannelUpgradeConfirmEvent(ctx, msg.PortId, msg.ChannelId, channel)

	// Move channel to OPEN state if both chains have finished flushing in-flight packets.
	// Counterparty channel state has been verified in ChanUpgradeConfirm.
	if channel.State == channeltypes.FLUSHCOMPLETE && msg.CounterpartyChannelState == channeltypes.FLUSHCOMPLETE {
		upgrade, found := k.ChannelKeeper.GetUpgrade(ctx, msg.PortId, msg.ChannelId)
		if !found {
			return nil, errorsmod.Wrapf(channeltypes.ErrUpgradeNotFound, "failed to retrieve channel upgrade: port ID (%s) channel ID (%s)", msg.PortId, msg.ChannelId)
		}

		cbs.OnChanUpgradeOpen(ctx, msg.PortId, msg.ChannelId, upgrade.Fields.Ordering, upgrade.Fields.ConnectionHops, upgrade.Fields.Version)
		channel := k.ChannelKeeper.WriteUpgradeOpenChannel(ctx, msg.PortId, msg.ChannelId)

		ctx.Logger().Info("channel upgrade open succeeded", "port-id", msg.PortId, "channel-id", msg.ChannelId)
		keeper.EmitChannelUpgradeOpenEvent(ctx, msg.PortId, msg.ChannelId, channel)
	}

	return &channeltypes.MsgChannelUpgradeConfirmResponse{Result: channeltypes.SUCCESS}, nil
}

// ChannelUpgradeOpen defines a rpc handler method for MsgChannelUpgradeOpen.
func (k *Keeper) ChannelUpgradeOpen(goCtx context.Context, msg *channeltypes.MsgChannelUpgradeOpen) (*channeltypes.MsgChannelUpgradeOpenResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	app, ok := k.PortKeeper.Route(msg.PortId)
	if !ok {
		err := errorsmod.Wrapf(porttypes.ErrInvalidRoute, "route not found to portID: %s", msg.PortId)
		ctx.Logger().Error("channel upgrade open failed", "port-id", msg.PortId, "error", err)
		return nil, err
	}

	cbs, ok := app.(porttypes.UpgradableModule)
	if !ok {
		err := errorsmod.Wrapf(porttypes.ErrInvalidRoute, "upgrade route not found to portID: %s", msg.PortId)
		ctx.Logger().Error("channel upgrade open failed", "port-id", msg.PortId, "error", err)
		return nil, err
	}

	if err := k.ChannelKeeper.ChanUpgradeOpen(ctx, msg.PortId, msg.ChannelId, msg.CounterpartyChannelState, msg.CounterpartyUpgradeSequence, msg.ProofChannel, msg.ProofHeight); err != nil {
		ctx.Logger().Error("channel upgrade open failed", "error", errorsmod.Wrap(err, "channel upgrade open failed"))
		return nil, errorsmod.Wrap(err, "channel upgrade open failed")
	}

	upgrade, found := k.ChannelKeeper.GetUpgrade(ctx, msg.PortId, msg.ChannelId)
	if !found {
		return nil, errorsmod.Wrapf(channeltypes.ErrUpgradeNotFound, "failed to retrieve channel upgrade: port ID (%s) channel ID (%s)", msg.PortId, msg.ChannelId)
	}

	cbs.OnChanUpgradeOpen(ctx, msg.PortId, msg.ChannelId, upgrade.Fields.Ordering, upgrade.Fields.ConnectionHops, upgrade.Fields.Version)
	channel := k.ChannelKeeper.WriteUpgradeOpenChannel(ctx, msg.PortId, msg.ChannelId)

	ctx.Logger().Info("channel upgrade open succeeded", "port-id", msg.PortId, "channel-id", msg.ChannelId)
	keeper.EmitChannelUpgradeOpenEvent(ctx, msg.PortId, msg.ChannelId, channel)

	return &channeltypes.MsgChannelUpgradeOpenResponse{}, nil
}

// ChannelUpgradeTimeout defines a rpc handler method for MsgChannelUpgradeTimeout.
func (k *Keeper) ChannelUpgradeTimeout(goCtx context.Context, msg *channeltypes.MsgChannelUpgradeTimeout) (*channeltypes.MsgChannelUpgradeTimeoutResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := k.ChannelKeeper.ChanUpgradeTimeout(ctx, msg.PortId, msg.ChannelId, msg.CounterpartyChannel, msg.ProofChannel, msg.ProofHeight); err != nil {
		return nil, errorsmod.Wrapf(err, "could not timeout upgrade for channel: %s", msg.ChannelId)
	}

	channel, upgrade := k.ChannelKeeper.WriteUpgradeTimeoutChannel(ctx, msg.PortId, msg.ChannelId)

	ctx.Logger().Info("channel upgrade timeout callback succeeded: portID %s, channelID %s", msg.PortId, msg.ChannelId)
	keeper.EmitChannelUpgradeTimeoutEvent(ctx, msg.PortId, msg.ChannelId, channel, upgrade)

	return &channeltypes.MsgChannelUpgradeTimeoutResponse{}, nil
}

// ChannelUpgradeCancel defines a rpc handler method for MsgChannelUpgradeCancel.
func (k *Keeper) ChannelUpgradeCancel(goCtx context.Context, msg *channeltypes.MsgChannelUpgradeCancel) (*channeltypes.MsgChannelUpgradeCancelResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	channel, found := k.ChannelKeeper.GetChannel(ctx, msg.PortId, msg.ChannelId)
	if !found {
		return nil, errorsmod.Wrapf(channeltypes.ErrChannelNotFound, "port ID (%s) channel ID (%s)", msg.PortId, msg.ChannelId)
	}

	// if the msgSender is authorized to make and cancel upgrades AND the current channel has not already reached FLUSHCOMPLETE
	// then we can restore immediately without any additional checks
	isAuthority := k.GetAuthority() == msg.Signer
	if isAuthority && channel.State != channeltypes.FLUSHCOMPLETE {
		upgrade, found := k.ChannelKeeper.GetUpgrade(ctx, msg.PortId, msg.ChannelId)
		if !found {
			return nil, errorsmod.Wrapf(channeltypes.ErrUpgradeNotFound, "failed to retrieve channel upgrade: port ID (%s) channel ID (%s)", msg.PortId, msg.ChannelId)
		}

		k.ChannelKeeper.WriteUpgradeCancelChannel(ctx, msg.PortId, msg.ChannelId, channel.UpgradeSequence)

		ctx.Logger().Info("channel upgrade cancel succeeded", "port-id", msg.PortId, "channel-id", msg.ChannelId)

		keeper.EmitChannelUpgradeCancelEvent(ctx, msg.PortId, msg.ChannelId, channel, upgrade)

		return &channeltypes.MsgChannelUpgradeCancelResponse{}, nil
	}

	if err := k.ChannelKeeper.ChanUpgradeCancel(ctx, msg.PortId, msg.ChannelId, msg.ErrorReceipt, msg.ProofErrorReceipt, msg.ProofHeight); err != nil {
		ctx.Logger().Error("channel upgrade cancel failed", "port-id", msg.PortId, "error", err.Error())
		return nil, errorsmod.Wrap(err, "channel upgrade cancel failed")
	}

	// get upgrade here since it will be deleted in WriteUpgradeCancelChannel
	upgrade, found := k.ChannelKeeper.GetUpgrade(ctx, msg.PortId, msg.ChannelId)
	if !found {
		return nil, errorsmod.Wrapf(channeltypes.ErrUpgradeNotFound, "failed to retrieve channel upgrade: port ID (%s) channel ID (%s)", msg.PortId, msg.ChannelId)
	}

	k.ChannelKeeper.WriteUpgradeCancelChannel(ctx, msg.PortId, msg.ChannelId, msg.ErrorReceipt.Sequence)

	ctx.Logger().Info("channel upgrade cancel succeeded", "port-id", msg.PortId, "channel-id", msg.ChannelId)

	// get channel here again to get latest state after write
	channel, found = k.ChannelKeeper.GetChannel(ctx, msg.PortId, msg.ChannelId)
	if !found {
		return nil, errorsmod.Wrapf(channeltypes.ErrChannelNotFound, "port ID (%s) channel ID (%s)", msg.PortId, msg.ChannelId)
	}
	keeper.EmitChannelUpgradeCancelEvent(ctx, msg.PortId, msg.ChannelId, channel, upgrade)

	return &channeltypes.MsgChannelUpgradeCancelResponse{}, nil
}

// PruneAcknowledgements defines a rpc handler method for MsgPruneAcknowledgements.
func (k *Keeper) PruneAcknowledgements(goCtx context.Context, msg *channeltypes.MsgPruneAcknowledgements) (*channeltypes.MsgPruneAcknowledgementsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	pruned, remaining, err := k.ChannelKeeper.PruneAcknowledgements(ctx, msg.PortId, msg.ChannelId, msg.Limit)
	if err != nil {
		return nil, err
	}

	return &channeltypes.MsgPruneAcknowledgementsResponse{
		TotalPrunedSequences:    pruned,
		TotalRemainingSequences: remaining,
	}, nil
}

// UpdateClientParams defines a rpc handler method for MsgUpdateParams.
func (k *Keeper) UpdateClientParams(goCtx context.Context, msg *clienttypes.MsgUpdateParams) (*clienttypes.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Signer {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "expected %s, got %s", k.GetAuthority(), msg.Signer)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	k.ClientKeeper.SetParams(ctx, msg.Params)

	return &clienttypes.MsgUpdateParamsResponse{}, nil
}

// UpdateConnectionParams defines a rpc handler method for MsgUpdateParams for the 03-connection submodule.
func (k *Keeper) UpdateConnectionParams(goCtx context.Context, msg *connectiontypes.MsgUpdateParams) (*connectiontypes.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Signer {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "expected %s, got %s", k.GetAuthority(), msg.Signer)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	k.ConnectionKeeper.SetParams(ctx, msg.Params)

	return &connectiontypes.MsgUpdateParamsResponse{}, nil
}

// UpdateChannelParams defines a rpc handler method for MsgUpdateParams.
func (k *Keeper) UpdateChannelParams(goCtx context.Context, msg *channeltypes.MsgUpdateParams) (*channeltypes.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	k.ChannelKeeper.SetParams(ctx, msg.Params)

	return &channeltypes.MsgUpdateParamsResponse{}, nil
}

// convertToErrorEvents converts all events to error events by appending the
// error attribute prefix to each event's attribute key.
func convertToErrorEvents(events sdk.Events) sdk.Events {
	if events == nil {
		return nil
	}

	newEvents := make(sdk.Events, len(events))
	for i, event := range events {
		newAttributes := make([]sdk.Attribute, len(event.Attributes))
		for j, attribute := range event.Attributes {
			newAttributes[j] = sdk.NewAttribute(coretypes.ErrorAttributeKeyPrefix+attribute.Key, attribute.Value)
		}

		newEvents[i] = sdk.NewEvent(coretypes.ErrorAttributeKeyPrefix+event.Type, newAttributes...)
	}

	return newEvents
}
