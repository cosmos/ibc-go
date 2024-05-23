package transfer

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	convertinternal "github.com/cosmos/ibc-go/v8/modules/apps/transfer/internal/convert"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
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
	chanCap *capabilitytypes.Capability,
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

	// Claim channel capability passed back by IBC module
	if err := im.keeper.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return "", err
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
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	if err := ValidateTransferChannelParams(ctx, im.keeper, order, portID, channelID); err != nil {
		return "", err
	}

	// OpenTry must claim the channelCapability that IBC passes into the callback
	if err := im.keeper.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return "", err
	}

	if !slices.Contains(types.SupportedVersions, counterpartyVersion) {
		return types.V2, nil
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

// unmarshalPacketDataBytesToICS20V2 attempts to unmarshal the provided packet data bytes into a FungibleTokenPacketDataV2.
// The version of ics20 should be provided and should be either ics20-1 or ics20-2.
func (IBCModule) unmarshalPacketDataBytesToICS20V2(bz []byte, ics20Version string) (types.FungibleTokenPacketDataV2, error) {
	switch ics20Version {
	case types.V1:
		var datav1 types.FungibleTokenPacketData
		if err := json.Unmarshal(bz, &datav1); err != nil {
			return types.FungibleTokenPacketDataV2{}, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "cannot unmarshal ICS20-V2 transfer packet data: %s", err.Error())
		}

		if err := datav1.ValidateBasic(); err != nil {
			return types.FungibleTokenPacketDataV2{}, err
		}

		return convertinternal.PacketDataV1ToV2(datav1), nil
	case types.V2:
		var datav2 types.FungibleTokenPacketDataV2
		if err := json.Unmarshal(bz, &datav2); err != nil {
			return types.FungibleTokenPacketDataV2{}, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "cannot unmarshal ICS20-V2 transfer packet data: %s", err.Error())
		}

		if err := datav2.ValidateBasic(); err != nil {
			return types.FungibleTokenPacketDataV2{}, err
		}

		return datav2, nil
	default:
		return types.FungibleTokenPacketDataV2{}, errorsmod.Wrap(types.ErrInvalidVersion, ics20Version)
	}
}

// OnRecvPacket implements the IBCModule interface. A successful acknowledgement
// is returned if the packet data is successfully decoded and the receive application
// logic returns without error.
func (im IBCModule) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	ack := channeltypes.NewResultAcknowledgement([]byte{byte(1)})

	ics20Version, found := im.keeper.GetICS4Wrapper().GetAppVersion(ctx, packet.DestinationPort, packet.DestinationChannel)
	if !found {
		return channeltypes.NewErrorAcknowledgement(errorsmod.Wrapf(ibcerrors.ErrNotFound, "app version not found for port %s and channel %s", packet.DestinationPort, packet.DestinationChannel))
	}

	data, ackErr := im.unmarshalPacketDataBytesToICS20V2(packet.GetData(), ics20Version)
	if ackErr != nil {
		ackErr = errorsmod.Wrapf(ibcerrors.ErrInvalidType, ackErr.Error())
		im.keeper.Logger(ctx).Error(fmt.Sprintf("%s sequence %d", ackErr.Error(), packet.Sequence))
		ack = channeltypes.NewErrorAcknowledgement(ackErr)
	}

	// only attempt the application logic if the packet data
	// was successfully decoded
	if ack.Success() {
		err := im.keeper.OnRecvPacket(ctx, packet, data)
		if err != nil {
			ack = channeltypes.NewErrorAcknowledgement(err)
			ackErr = err
			im.keeper.Logger(ctx).Error(fmt.Sprintf("%s sequence %d", ackErr.Error(), packet.Sequence))
		} else {
			im.keeper.Logger(ctx).Info("successfully handled ICS-20 packet", "sequence", packet.Sequence)
		}
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypePacket,
		sdk.NewAttribute(types.AttributeKeySender, data.Sender),
		sdk.NewAttribute(types.AttributeKeyReceiver, data.Receiver),
		sdk.NewAttribute(types.AttributeKeyTokens, types.Tokens(data.Tokens).String()),
		sdk.NewAttribute(types.AttributeKeyMemo, data.Memo),
	))

	eventAttributes := []sdk.Attribute{
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		sdk.NewAttribute(types.AttributeKeyAckSuccess, fmt.Sprintf("%t", ack.Success())),
	}
	if ackErr != nil {
		eventAttributes = append(eventAttributes, sdk.NewAttribute(types.AttributeKeyAckError, ackErr.Error()))
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		sdk.EventTypeMessage,
		eventAttributes...,
	))

	// NOTE: acknowledgement will be written synchronously during IBC handler execution.
	return ack
}

// OnAcknowledgementPacket implements the IBCModule interface
func (im IBCModule) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	var ack channeltypes.Acknowledgement
	if err := types.ModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrUnknownRequest, "cannot unmarshal ICS-20 transfer packet acknowledgement: %v", err)
	}

	ics20Version, found := im.keeper.GetICS4Wrapper().GetAppVersion(ctx, packet.SourcePort, packet.SourceChannel)
	if !found {
		return errorsmod.Wrapf(ibcerrors.ErrNotFound, "app version not found for port %s and channel %s", packet.SourcePort, packet.SourceChannel)
	}

	data, err := im.unmarshalPacketDataBytesToICS20V2(packet.GetData(), ics20Version)
	if err != nil {
		return err
	}

	if err := im.keeper.OnAcknowledgementPacket(ctx, packet, data, ack); err != nil {
		return err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypePacket,
			sdk.NewAttribute(sdk.AttributeKeySender, data.Sender),
			sdk.NewAttribute(types.AttributeKeyReceiver, data.Receiver),
			sdk.NewAttribute(types.AttributeKeyTokens, types.Tokens(data.Tokens).String()),
			sdk.NewAttribute(types.AttributeKeyMemo, data.Memo),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
			sdk.NewAttribute(types.AttributeKeyAck, ack.String()),
		),
	})

	switch resp := ack.Response.(type) {
	case *channeltypes.Acknowledgement_Result:
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypePacket,
				sdk.NewAttribute(types.AttributeKeyAckSuccess, string(resp.Result)),
			),
		)
	case *channeltypes.Acknowledgement_Error:
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypePacket,
				sdk.NewAttribute(types.AttributeKeyAckError, resp.Error),
			),
		)
	}

	return nil
}

// OnTimeoutPacket implements the IBCModule interface
func (im IBCModule) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	ics20Version, found := im.keeper.GetICS4Wrapper().GetAppVersion(ctx, packet.SourcePort, packet.SourceChannel)
	if !found {
		return errorsmod.Wrapf(ibcerrors.ErrNotFound, "app version not found for port %s and channel %s", packet.SourcePort, packet.SourceChannel)
	}

	data, err := im.unmarshalPacketDataBytesToICS20V2(packet.GetData(), ics20Version)
	if err != nil {
		return err
	}

	// refund tokens
	if err := im.keeper.OnTimeoutPacket(ctx, packet, data); err != nil {
		return err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeTimeout,
			sdk.NewAttribute(sdk.AttributeKeySender, data.Sender),
			sdk.NewAttribute(types.AttributeKeyReceiver, data.Receiver),
			sdk.NewAttribute(types.AttributeKeyTokens, types.Tokens(data.Tokens).String()),
			sdk.NewAttribute(types.AttributeKeyMemo, data.Memo),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
	})

	return nil
}

// OnChanUpgradeInit implements the IBCModule interface
func (im IBCModule) OnChanUpgradeInit(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, proposedVersion string) (string, error) {
	if err := ValidateTransferChannelParams(ctx, im.keeper, proposedOrder, portID, channelID); err != nil {
		return "", err
	}

	if !slices.Contains(types.SupportedVersions, proposedVersion) {
		return "", errorsmod.Wrapf(types.ErrInvalidVersion, "invalid counterparty version: expected one of %s, got %s", types.SupportedVersions, proposedVersion)
	}

	return proposedVersion, nil
}

// OnChanUpgradeTry implements the IBCModule interface
func (im IBCModule) OnChanUpgradeTry(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, counterpartyVersion string) (string, error) {
	if err := ValidateTransferChannelParams(ctx, im.keeper, proposedOrder, portID, channelID); err != nil {
		return "", err
	}

	if !slices.Contains(types.SupportedVersions, counterpartyVersion) {
		return types.V2, nil
	}

	return counterpartyVersion, nil
}

// OnChanUpgradeAck implements the IBCModule interface
func (IBCModule) OnChanUpgradeAck(ctx sdk.Context, portID, channelID, counterpartyVersion string) error {
	if !slices.Contains(types.SupportedVersions, counterpartyVersion) {
		return errorsmod.Wrapf(types.ErrInvalidVersion, "invalid counterparty version: expected one of %s, got %s", types.SupportedVersions, counterpartyVersion)
	}

	return nil
}

// OnChanUpgradeOpen implements the IBCModule interface
func (IBCModule) OnChanUpgradeOpen(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, proposedVersion string) {
}

// UnmarshalPacketData attempts to unmarshal the provided packet data bytes
// into a FungibleTokenPacketData. This function implements the optional
// PacketDataUnmarshaler interface required for ADR 008 support.
func (im IBCModule) UnmarshalPacketData(ctx sdk.Context, portID, channelID string, bz []byte) (interface{}, error) {
	ics20Version, found := im.keeper.GetICS4Wrapper().GetAppVersion(ctx, portID, channelID)
	if !found {
		return nil, errorsmod.Wrapf(ibcerrors.ErrNotFound, "app version not found for port %s and channel %s", portID, channelID)
	}

	ftpd, err := im.unmarshalPacketDataBytesToICS20V2(bz, ics20Version)
	if err != nil {
		return nil, err
	}

	return ftpd, nil
}
