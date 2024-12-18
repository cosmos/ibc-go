package transfer

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/keeper"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
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

// IBCModule implements the ICS26 interface for transfer given the transfer keeper.
type IBCModule struct {
	keeper keeper.Keeper
}

// NewIBCModule creates a new IBCModule given the keeper
func NewIBCModule(k keeper.Keeper) IBCModule {
	return IBCModule{
		keeper: k,
	}
}

// ValidateTransferChannelParams does validation of a newly created transfer channel. A transfer
// channel must be UNORDERED, use the correct port (by default 'transfer'), and use the current
// supported version. Only 2^32 channels are allowed to be created.
func ValidateTransferChannelParams(
	ctx context.Context,
	transferkeeper keeper.Keeper,
	order channeltypes.Order,
	portID string,
	channelID string,
) error {
	// NOTE: for escrow address security only 2^32 channels are allowed to be created
	// Issue: https://github.com/cosmos/cosmos-sdk/issues/7737
	channelSequence, err := channeltypes.ParseChannelSequence(channelID)
	if err != nil {
		return err
	}
	if channelSequence > uint64(math.MaxUint32) {
		return errorsmod.Wrapf(types.ErrMaxTransferChannels, "channel sequence %d is greater than max allowed transfer channels %d", channelSequence, uint64(math.MaxUint32))
	}
	if order != channeltypes.UNORDERED {
		return errorsmod.Wrapf(channeltypes.ErrInvalidChannelOrdering, "expected %s channel, got %s ", channeltypes.UNORDERED, order)
	}

	// Require portID is the portID transfer module is bound to
	boundPort := transferkeeper.GetPort(ctx)
	if boundPort != portID {
		return errorsmod.Wrapf(porttypes.ErrInvalidPort, "invalid port: %s, expected %s", portID, boundPort)
	}

	return nil
}

// OnChanOpenInit implements the IBCModule interface
func (im IBCModule) OnChanOpenInit(
	ctx context.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	if err := ValidateTransferChannelParams(ctx, im.keeper, order, portID, channelID); err != nil {
		return "", err
	}

	// default to latest supported version
	if strings.TrimSpace(version) == "" {
		version = types.V2
	}

	if !slices.Contains(types.SupportedVersions, version) {
		return "", errorsmod.Wrapf(types.ErrInvalidVersion, "expected one of %s, got %s", types.SupportedVersions, version)
	}

	return version, nil
}

// OnChanOpenTry implements the IBCModule interface.
func (im IBCModule) OnChanOpenTry(
	ctx context.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	if err := ValidateTransferChannelParams(ctx, im.keeper, order, portID, channelID); err != nil {
		return "", err
	}

	if !slices.Contains(types.SupportedVersions, counterpartyVersion) {
		im.keeper.Logger.Debug("invalid counterparty version, proposing latest app version", "counterpartyVersion", counterpartyVersion, "version", types.V2)
		return types.V2, nil
	}

	return counterpartyVersion, nil
}

// OnChanOpenAck implements the IBCModule interface
func (IBCModule) OnChanOpenAck(
	ctx context.Context,
	portID,
	channelID string,
	_ string,
	counterpartyVersion string,
) error {
	if !slices.Contains(types.SupportedVersions, counterpartyVersion) {
		return errorsmod.Wrapf(types.ErrInvalidVersion, "invalid counterparty version: expected one of %s, got %s", types.SupportedVersions, counterpartyVersion)
	}

	return nil
}

// OnChanOpenConfirm implements the IBCModule interface
func (IBCModule) OnChanOpenConfirm(
	ctx context.Context,
	portID,
	channelID string,
) error {
	return nil
}

// OnChanCloseInit implements the IBCModule interface
func (IBCModule) OnChanCloseInit(
	ctx context.Context,
	portID,
	channelID string,
) error {
	// Disallow user-initiated channel closing for transfer channels
	return errorsmod.Wrap(ibcerrors.ErrInvalidRequest, "user cannot close channel")
}

// OnChanCloseConfirm implements the IBCModule interface
func (IBCModule) OnChanCloseConfirm(
	ctx context.Context,
	portID,
	channelID string,
) error {
	return nil
}

// OnRecvPacket implements the IBCModule interface. A successful acknowledgement
// is returned if the packet data is successfully decoded and the receive application
// logic returns without error.
// A nil acknowledgement may be returned when using the packet forwarding feature. This signals to core IBC that the acknowledgement will be written asynchronously.
func (im IBCModule) OnRecvPacket(
	ctx context.Context,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	var (
		ackErr error
		data   types.FungibleTokenPacketDataV2
	)

	ack := channeltypes.NewResultAcknowledgement([]byte{byte(1)})

	// we are explicitly wrapping this emit event call in an anonymous function so that
	// the packet data is evaluated after it has been assigned a value.
	defer func() {
		if err := im.keeper.EmitOnRecvPacketEvent(ctx, data, ack, ackErr); err != nil {
			ack = channeltypes.NewErrorAcknowledgement(err)
		}
	}()

	data, ackErr = types.UnmarshalPacketData(packet.GetData(), channelVersion)
	if ackErr != nil {
		ack = channeltypes.NewErrorAcknowledgement(ackErr)
		im.keeper.Logger.Error(fmt.Sprintf("%s sequence %d", ackErr.Error(), packet.Sequence))
		return ack
	}

	if ackErr = im.keeper.OnRecvPacket(ctx, packet, data); ackErr != nil {
		ack = channeltypes.NewErrorAcknowledgement(ackErr)
		im.keeper.Logger.Error(fmt.Sprintf("%s sequence %d", ackErr.Error(), packet.Sequence))
		return ack
	}

	im.keeper.Logger.Info("successfully handled ICS-20 packet", "sequence", packet.Sequence)

	if data.HasForwarding() {
		// NOTE: acknowledgement will be written asynchronously
		return nil
	}

	// NOTE: acknowledgement will be written synchronously during IBC handler execution.
	return ack
}

// OnAcknowledgementPacket implements the IBCModule interface
func (im IBCModule) OnAcknowledgementPacket(
	ctx context.Context,
	channelVersion string,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	var ack channeltypes.Acknowledgement
	if err := types.ModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrUnknownRequest, "cannot unmarshal ICS-20 transfer packet acknowledgement: %v", err)
	}

	data, err := types.UnmarshalPacketData(packet.GetData(), channelVersion)
	if err != nil {
		return err
	}

	if err := im.keeper.OnAcknowledgementPacket(ctx, packet, data, ack); err != nil {
		return err
	}

	return im.keeper.EmitOnAcknowledgementPacketEvent(ctx, data, ack)
}

// OnTimeoutPacket implements the IBCModule interface
func (im IBCModule) OnTimeoutPacket(
	ctx context.Context,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	data, err := types.UnmarshalPacketData(packet.GetData(), channelVersion)
	if err != nil {
		return err
	}

	// refund tokens
	if err := im.keeper.OnTimeoutPacket(ctx, packet, data); err != nil {
		return err
	}

	return im.keeper.EmitOnTimeoutEvent(ctx, data)
}

// OnChanUpgradeInit implements the IBCModule interface
func (im IBCModule) OnChanUpgradeInit(ctx context.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, proposedVersion string) (string, error) {
	if err := ValidateTransferChannelParams(ctx, im.keeper, proposedOrder, portID, channelID); err != nil {
		return "", err
	}

	if !slices.Contains(types.SupportedVersions, proposedVersion) {
		return "", errorsmod.Wrapf(types.ErrInvalidVersion, "invalid counterparty version: expected one of %s, got %s", types.SupportedVersions, proposedVersion)
	}

	return proposedVersion, nil
}

// OnChanUpgradeTry implements the IBCModule interface
func (im IBCModule) OnChanUpgradeTry(ctx context.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, counterpartyVersion string) (string, error) {
	if err := ValidateTransferChannelParams(ctx, im.keeper, proposedOrder, portID, channelID); err != nil {
		return "", err
	}

	if !slices.Contains(types.SupportedVersions, counterpartyVersion) {
		im.keeper.Logger.Debug("invalid counterparty version, proposing latest app version", "counterpartyVersion", counterpartyVersion, "version", types.V2)
		return types.V2, nil
	}

	return counterpartyVersion, nil
}

// OnChanUpgradeAck implements the IBCModule interface
func (IBCModule) OnChanUpgradeAck(ctx context.Context, portID, channelID, counterpartyVersion string) error {
	if !slices.Contains(types.SupportedVersions, counterpartyVersion) {
		return errorsmod.Wrapf(types.ErrInvalidVersion, "invalid counterparty version: expected one of %s, got %s", types.SupportedVersions, counterpartyVersion)
	}

	return nil
}

// OnChanUpgradeOpen implements the IBCModule interface
func (IBCModule) OnChanUpgradeOpen(ctx context.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, proposedVersion string) {
}

// UnmarshalPacketData attempts to unmarshal the provided packet data bytes
// into a FungibleTokenPacketData. This function implements the optional
// PacketDataUnmarshaler interface required for ADR 008 support.
func (im IBCModule) UnmarshalPacketData(ctx context.Context, portID string, channelID string, bz []byte) (interface{}, string, error) {
	ics20Version, found := im.keeper.GetICS4Wrapper().GetAppVersion(ctx, portID, channelID)
	if !found {
		return types.FungibleTokenPacketDataV2{}, "", errorsmod.Wrapf(ibcerrors.ErrNotFound, "app version not found for port %s and channel %s", portID, channelID)
	}

	ftpd, err := types.UnmarshalPacketData(bz, ics20Version)
	return ftpd, ics20Version, err
}
