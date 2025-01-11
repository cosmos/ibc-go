package host

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/host/keeper"
	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v9/modules/core/05-port/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	ibcexported "github.com/cosmos/ibc-go/v9/modules/core/exported"
)

var (
	_ porttypes.IBCModule             = (*IBCModule)(nil)
	_ porttypes.PacketDataUnmarshaler = (*IBCModule)(nil)
	_ porttypes.UpgradableModule      = (*IBCModule)(nil)
)

// IBCModule implements the ICS26 interface for interchain accounts host chains
type IBCModule struct {
	keeper keeper.Keeper
}

// NewIBCModule creates a new IBCModule given the associated keeper
func NewIBCModule(k keeper.Keeper) IBCModule {
	return IBCModule{
		keeper: k,
	}
}

// OnChanOpenInit implements the IBCModule interface
func (IBCModule) OnChanOpenInit(
	_ context.Context,
	_ channeltypes.Order,
	_ []string,
	_ string,
	_ string,
	_ channeltypes.Counterparty,
	_ string,
) (string, error) {
	return "", errorsmod.Wrap(icatypes.ErrInvalidChannelFlow, "channel handshake must be initiated by controller chain")
}

// OnChanOpenTry implements the IBCModule interface
func (im IBCModule) OnChanOpenTry(
	ctx context.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	if !im.keeper.GetParams(ctx).HostEnabled {
		return "", types.ErrHostSubModuleDisabled
	}

	return im.keeper.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, counterparty, counterpartyVersion)
}

// OnChanOpenAck implements the IBCModule interface
func (IBCModule) OnChanOpenAck(
	_ context.Context,
	_,
	_ string,
	_ string,
	_ string,
) error {
	return errorsmod.Wrap(icatypes.ErrInvalidChannelFlow, "channel handshake must be initiated by controller chain")
}

// OnChanOpenConfirm implements the IBCModule interface
func (im IBCModule) OnChanOpenConfirm(
	ctx context.Context,
	portID,
	channelID string,
) error {
	if !im.keeper.GetParams(ctx).HostEnabled {
		return types.ErrHostSubModuleDisabled
	}

	return im.keeper.OnChanOpenConfirm(ctx, portID, channelID)
}

// OnChanCloseInit implements the IBCModule interface
func (IBCModule) OnChanCloseInit(
	_ context.Context,
	_ string,
	_ string,
) error {
	// Disallow user-initiated channel closing for interchain account channels
	return errorsmod.Wrap(ibcerrors.ErrInvalidRequest, "user cannot close channel")
}

// OnChanCloseConfirm implements the IBCModule interface
func (im IBCModule) OnChanCloseConfirm(
	ctx context.Context,
	portID,
	channelID string,
) error {
	return im.keeper.OnChanCloseConfirm(ctx, portID, channelID)
}

// OnRecvPacket implements the IBCModule interface
func (im IBCModule) OnRecvPacket(
	ctx context.Context,
	_ string,
	packet channeltypes.Packet,
	_ sdk.AccAddress,
) ibcexported.Acknowledgement {
	if !im.keeper.GetParams(ctx).HostEnabled {
		im.keeper.Logger.Info("host submodule is disabled")
		if err := im.keeper.EmitHostDisabledEvent(ctx, packet); err != nil {
			return channeltypes.NewErrorAcknowledgement(err)
		}

		return channeltypes.NewErrorAcknowledgement(types.ErrHostSubModuleDisabled)
	}

	txResponse, err := im.keeper.OnRecvPacket(ctx, packet)
	ack := channeltypes.NewResultAcknowledgement(txResponse)
	if err != nil {
		ack = channeltypes.NewErrorAcknowledgement(err)
		im.keeper.Logger.Error(fmt.Sprintf("%s sequence %d", err.Error(), packet.Sequence))
	} else {
		im.keeper.Logger.Info("successfully handled packet", "sequence", packet.Sequence)
	}

	// Emit an event indicating a successful or failed acknowledgement.
	if err := im.keeper.EmitAcknowledgementEvent(ctx, packet, ack, err); err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	// NOTE: acknowledgement will be written synchronously during IBC handler execution.
	return ack
}

// OnAcknowledgementPacket implements the IBCModule interface
func (IBCModule) OnAcknowledgementPacket(
	_ context.Context,
	_ string,
	_ channeltypes.Packet,
	_ []byte,
	_ sdk.AccAddress,
) error {
	return errorsmod.Wrap(icatypes.ErrInvalidChannelFlow, "cannot receive acknowledgement on a host channel end, a host chain does not send a packet over the channel")
}

// OnTimeoutPacket implements the IBCModule interface
func (IBCModule) OnTimeoutPacket(
	_ context.Context,
	_ string,
	_ channeltypes.Packet,
	_ sdk.AccAddress,
) error {
	return errorsmod.Wrap(icatypes.ErrInvalidChannelFlow, "cannot cause a packet timeout on a host channel end, a host chain does not send a packet over the channel")
}

// OnChanUpgradeInit implements the IBCModule interface
func (IBCModule) OnChanUpgradeInit(_ context.Context, _, _ string, _ channeltypes.Order, _ []string, _ string) (string, error) {
	return "", errorsmod.Wrap(icatypes.ErrInvalidChannelFlow, "channel upgrade handshake must be initiated by controller chain")
}

// OnChanUpgradeTry implements the IBCModule interface
func (im IBCModule) OnChanUpgradeTry(ctx context.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, counterpartyVersion string) (string, error) {
	if !im.keeper.GetParams(ctx).HostEnabled {
		return "", types.ErrHostSubModuleDisabled
	}

	return im.keeper.OnChanUpgradeTry(ctx, portID, channelID, proposedOrder, proposedConnectionHops, counterpartyVersion)
}

// OnChanUpgradeAck implements the IBCModule interface
func (IBCModule) OnChanUpgradeAck(_ context.Context, _, _, _ string) error {
	return errorsmod.Wrap(icatypes.ErrInvalidChannelFlow, "channel upgrade handshake must be initiated by controller chain")
}

// OnChanUpgradeOpen implements the IBCModule interface
func (IBCModule) OnChanUpgradeOpen(_ context.Context, _, _ string, _ channeltypes.Order, _ []string, _ string) {
}

// UnmarshalPacketData attempts to unmarshal the provided packet data bytes
// into an InterchainAccountPacketData. This function implements the optional
// PacketDataUnmarshaler interface required for ADR 008 support.
func (im IBCModule) UnmarshalPacketData(ctx context.Context, portID string, channelID string, bz []byte) (interface{}, string, error) {
	var data icatypes.InterchainAccountPacketData
	err := data.UnmarshalJSON(bz)
	if err != nil {
		return nil, "", err
	}

	version, ok := im.keeper.GetAppVersion(ctx, portID, channelID)
	if !ok {
		return nil, "", errorsmod.Wrapf(ibcerrors.ErrNotFound, "app version not found for port %s and channel %s", portID, channelID)
	}

	return data, version, nil
}
