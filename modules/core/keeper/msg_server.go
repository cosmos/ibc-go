package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	clientv2types "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	internalerrors "github.com/cosmos/ibc-go/v10/modules/core/internal/errors"
	"github.com/cosmos/ibc-go/v10/modules/core/internal/telemetry"
)

var (
	_ clienttypes.MsgServer     = (*Keeper)(nil)
	_ clientv2types.MsgServer   = (*Keeper)(nil)
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

	// set the client creator so that IBC v2 counterparty can be set by same relayer
	k.ClientKeeper.SetClientCreator(ctx, clientID, sdk.MustAccAddressFromBech32(msg.Signer))

	return &clienttypes.MsgCreateClientResponse{ClientId: clientID}, nil
}

// RegisterCounterparty will register the IBC v2 counterparty info for the given client id
// it must be called by the same relayer that called CreateClient
func (k *Keeper) RegisterCounterparty(goCtx context.Context, msg *clientv2types.MsgRegisterCounterparty) (*clientv2types.MsgRegisterCounterpartyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	creator := k.ClientKeeper.GetClientCreator(ctx, msg.ClientId)
	if !creator.Equals(sdk.MustAccAddressFromBech32(msg.Signer)) {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "expected same signer as createClient submittor %s, got %s", creator, msg.Signer)
	}
	if _, ok := k.ClientV2Keeper.GetClientCounterparty(ctx, msg.ClientId); ok {
		return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "cannot register counterparty once it is already set")
	}

	counterpartyInfo := clientv2types.CounterpartyInfo{
		MerklePrefix: msg.CounterpartyMerklePrefix,
		ClientId:     msg.CounterpartyClientId,
	}
	k.ClientV2Keeper.SetClientCounterparty(ctx, msg.ClientId, counterpartyInfo)

	// initialize next sequence send to enable packet flow
	k.ChannelKeeperV2.SetNextSequenceSend(ctx, msg.ClientId, 1)

	return &clientv2types.MsgRegisterCounterpartyResponse{}, nil
}

// UpdateClient defines a rpc handler method for MsgUpdateClient.
func (k *Keeper) UpdateClient(goCtx context.Context, msg *clienttypes.MsgUpdateClient) (*clienttypes.MsgUpdateClientResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	clientMsg, err := clienttypes.UnpackClientMessage(msg.ClientMessage)
	if err != nil {
		return nil, err
	}

	// only check v2 params if this chain is setup with v2 clientKeepr
	if k.ClientV2Keeper != nil {
		// check if this relayer is allowed to update if v2 configuration are set
		config := k.ClientV2Keeper.GetConfig(ctx, msg.ClientId)
		if !config.IsAllowedRelayer(sdk.MustAccAddressFromBech32(msg.Signer)) {
			return nil, errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "relayer %s is not authorized to update client %s", msg.Signer, msg.ClientId)
		}
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

	ctx := sdk.UnwrapSDKContext(goCtx)
	upgradedClientState, err := clienttypes.UnpackClientState(msg.UpgradedClientState)
	if err != nil {
		return nil, errorsmod.Wrapf(clienttypes.ErrInvalidClientType, "cannot unpack client state: %s", err)
	}

	if err = k.ClientKeeper.ScheduleIBCSoftwareUpgrade(ctx, msg.Plan, upgradedClientState); err != nil {
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

	err := k.ChannelKeeper.ChanCloseConfirm(ctx, msg.PortId, msg.ChannelId, msg.ProofInit, msg.ProofHeight)
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
		ctx.EventManager().EmitEvents(internalerrors.ConvertToErrorEvents(cacheCtx.EventManager().Events()))
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
	channelVersion, err := k.ChannelKeeper.TimeoutOnClose(cacheCtx, msg.Packet, msg.ProofUnreceived, msg.ProofClose, msg.ProofHeight, msg.NextSequenceRecv)

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

	// Retrieve callbacks from router
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

// UpdateClientConfig defines an rpc handler method for MsgUpdateClientConfig for the 02-client v2 submodule.
func (k *Keeper) UpdateClientConfig(goCtx context.Context, msg *clientv2types.MsgUpdateClientConfig) (*clientv2types.MsgUpdateClientConfigResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	creator := k.ClientKeeper.GetClientCreator(ctx, msg.ClientId)
	if k.GetAuthority() != msg.Signer && !creator.Equals(sdk.MustAccAddressFromBech32(msg.Signer)) {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "authority %s or client creator %s is authorized to update params for %s, got %s",
			k.GetAuthority(), creator, msg.ClientId, msg.Signer,
		)
	}

	k.ClientV2Keeper.SetConfig(ctx, msg.ClientId, msg.Config)
	return &clientv2types.MsgUpdateClientConfigResponse{}, nil
}

// DeleteClientCreator defines an rpc handler method for MsgDeleteClientCreator for the 02-client v1 submodule.
func (k *Keeper) DeleteClientCreator(goCtx context.Context, msg *clienttypes.MsgDeleteClientCreator) (*clienttypes.MsgDeleteClientCreatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	creator := k.ClientKeeper.GetClientCreator(ctx, msg.ClientId)
	if creator == nil {
		return nil, errorsmod.Wrapf(ibcerrors.ErrNotFound, "creator for client %s not found", msg.ClientId)
	}

	// Check authorization
	if k.GetAuthority() != msg.Signer && !creator.Equals(sdk.MustAccAddressFromBech32(msg.Signer)) {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "authority %s or client creator %s is authorized to delete creator for %s, got %s",
			k.GetAuthority(), creator, msg.ClientId, msg.Signer,
		)
	}

	k.ClientKeeper.DeleteClientCreator(ctx, msg.ClientId)
	return &clienttypes.MsgDeleteClientCreatorResponse{}, nil
}
