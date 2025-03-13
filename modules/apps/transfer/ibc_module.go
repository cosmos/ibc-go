package transfer

import (
	"bytes"
	"fmt"
	"math"
	"slices"
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/internal/events"
	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/internal/telemetry"
	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

var (
	_ porttypes.IBCModule             = (*IBCModule)(nil)
	_ porttypes.PacketDataUnmarshaler = (*IBCModule)(nil)
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
	ctx sdk.Context,
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
	ctx sdk.Context,
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
		version = types.V1
	}

	if !slices.Contains(types.SupportedVersions, version) {
		return "", errorsmod.Wrapf(types.ErrInvalidVersion, "expected one of %s, got %s", types.SupportedVersions, version)
	}

	return version, nil
}

// OnChanOpenTry implements the IBCModule interface.
func (im IBCModule) OnChanOpenTry(
	ctx sdk.Context,
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
		im.keeper.Logger(ctx).Debug("invalid counterparty version, proposing latest app version", "counterpartyVersion", counterpartyVersion, "version", types.V1)
		return types.V1, nil
	}

	return counterpartyVersion, nil
}

// OnChanOpenAck implements the IBCModule interface
func (IBCModule) OnChanOpenAck(
	ctx sdk.Context,
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
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return nil
}

// OnChanCloseInit implements the IBCModule interface
func (IBCModule) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Disallow user-initiated channel closing for transfer channels
	return errorsmod.Wrap(ibcerrors.ErrInvalidRequest, "user cannot close channel")
}

// OnChanCloseConfirm implements the IBCModule interface
func (IBCModule) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return nil
}

// OnRecvPacket implements the IBCModule interface. A successful acknowledgement
// is returned if the packet data is successfully decoded and the receive application
// logic returns without error.
func (im IBCModule) OnRecvPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	var (
		ack    ibcexported.Acknowledgement
		ackErr error
		data   types.InternalTransferRepresentation
	)

	// we are explicitly wrapping this emit event call in an anonymous function so that
	// the packet data is evaluated after it has been assigned a value.
	defer func() {
		events.EmitOnRecvPacketEvent(ctx, data, ack, ackErr)
	}()

	data, ackErr = types.UnmarshalPacketData(packet.GetData(), channelVersion, "")
	if ackErr != nil {
		ack = channeltypes.NewErrorAcknowledgement(ackErr)
		im.keeper.Logger(ctx).Error(fmt.Sprintf("%s sequence %d", ackErr.Error(), packet.Sequence))
		return ack
	}

	// NOTE: this needs to set the ackErr variable and not do if ackErr := ... because the ackErr variable is used in the defer function
	ackErr = im.keeper.OnRecvPacket(
		ctx,
		data,
		packet.SourcePort,
		packet.SourceChannel,
		packet.DestinationPort,
		packet.DestinationChannel,
	)
	if ackErr != nil {
		ack = channeltypes.NewErrorAcknowledgement(ackErr)
		im.keeper.Logger(ctx).Error(fmt.Sprintf("%s sequence %d", ackErr.Error(), packet.Sequence))
		return ack
	}

	ack = channeltypes.NewResultAcknowledgement([]byte{byte(1)})

	telemetry.ReportOnRecvPacket(packet.SourcePort, packet.SourceChannel, packet.DestinationPort, packet.DestinationChannel, data.Token)

	im.keeper.Logger(ctx).Info("successfully handled ICS-20 packet", "sequence", packet.Sequence)

	// NOTE: acknowledgement will be written synchronously during IBC handler execution.
	return ack
}

// OnAcknowledgementPacket implements the IBCModule interface
func (im IBCModule) OnAcknowledgementPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	var ack channeltypes.Acknowledgement
	if err := types.ModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrUnknownRequest, "cannot unmarshal ICS-20 transfer packet acknowledgement: %v", err)
	}

	data, err := types.UnmarshalPacketData(packet.GetData(), channelVersion, "")
	if err != nil {
		return err
	}

	bz := types.ModuleCdc.MustMarshalJSON(&ack)
	if !bytes.Equal(bz, acknowledgement) {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "acknowledgement did not marshal to expected bytes: %X ≠ %X", bz, acknowledgement)
	}

	if err := im.keeper.OnAcknowledgementPacket(ctx, packet.SourcePort, packet.SourceChannel, data, ack); err != nil {
		return err
	}

	events.EmitOnAcknowledgementPacketEvent(ctx, data, ack)

	return nil
}

// OnTimeoutPacket implements the IBCModule interface
func (im IBCModule) OnTimeoutPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	data, err := types.UnmarshalPacketData(packet.GetData(), channelVersion, "")
	if err != nil {
		return err
	}

	// refund tokens
	if err := im.keeper.OnTimeoutPacket(ctx, packet.SourcePort, packet.SourceChannel, data); err != nil {
		return err
	}

	events.EmitOnTimeoutEvent(ctx, data)

	return nil
}

// UnmarshalPacketData attempts to unmarshal the provided packet data bytes
// into a FungibleTokenPacketData. This function implements the optional
// PacketDataUnmarshaler interface required for ADR 008 support.
func (im IBCModule) UnmarshalPacketData(ctx sdk.Context, portID string, channelID string, bz []byte) (interface{}, string, error) {
	ics20Version, found := im.keeper.GetICS4Wrapper().GetAppVersion(ctx, portID, channelID)
	if !found {
		return types.InternalTransferRepresentation{}, "", errorsmod.Wrapf(ibcerrors.ErrNotFound, "app version not found for port %s and channel %s", portID, channelID)
	}

	ftpd, err := types.UnmarshalPacketData(bz, ics20Version, "")
	return ftpd, ics20Version, err
}
