package transfer

import (
	"fmt"
	"math"
	"slices"
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	transferv2 "github.com/cosmos/ibc-go/v8/modules/apps/transfer/v2"
	channelkeeper "github.com/cosmos/ibc-go/v8/modules/core/04-channel/keeper"
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

// IBCModule implements the ICS26 interface for transfer given the transfer transferKeeper.
type IBCModule struct {
	transferKeeper keeper.Keeper
	channelKeeper  channelkeeper.Keeper
}

// NewIBCModule creates a new IBCModule given the transferKeeper
func NewIBCModule(transferKeeper keeper.Keeper, channelKeeper channelkeeper.Keeper) IBCModule {
	return IBCModule{
		transferKeeper: transferKeeper,
		channelKeeper:  channelKeeper,
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
	if err := ValidateTransferChannelParams(ctx, im.transferKeeper, order, portID, channelID); err != nil {
		return "", err
	}

	if strings.TrimSpace(version) == "" {
		version = types.CurrentVersion
	}

	if !slices.Contains([]string{types.CurrentVersion, types.Version1}, version) {
		return "", errorsmod.Wrapf(types.ErrInvalidVersion, "invalid version: expected %s or %s, got %s", types.Version1, types.CurrentVersion, version)
	}

	// Claim channel capability passed back by IBC module
	if err := im.transferKeeper.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return "", err
	}

	if version == types.Version1 {
		return types.Version1, nil
	}

	// default to latest supported version
	return types.CurrentVersion, nil
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
	if err := ValidateTransferChannelParams(ctx, im.transferKeeper, order, portID, channelID); err != nil {
		return "", err
	}

	if !slices.Contains([]string{types.CurrentVersion, types.Version1}, counterpartyVersion) {
		return "", errorsmod.Wrapf(types.ErrInvalidVersion, "invalid counterparty version: expected %s or %s, got %s", types.Version1, types.CurrentVersion, counterpartyVersion)
	}

	// OpenTry must claim the channelCapability that IBC passes into the callback
	if err := im.transferKeeper.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return "", err
	}

	// return counterparty version if we support it
	if counterpartyVersion == types.CurrentVersion {
		return counterpartyVersion, nil
	}

	return types.Version1, nil
}

// OnChanOpenAck implements the IBCModule interface
func (im IBCModule) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	_ string,
	counterpartyVersion string,
) error {

	// TODO: this does not work with middleware
	// ref: https://github.com/cosmos/ibc/pull/1020/files#r1469559632
	channel, found := im.channelKeeper.GetChannel(ctx, portID, channelID)
	if !found {
		return errorsmod.Wrapf(channeltypes.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}

	if counterpartyVersion != channel.Version {
		return errorsmod.Wrapf(types.ErrInvalidVersion, "invalid counterparty version: expected %s, got %s", channel.Version, counterpartyVersion)
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

func getFungibleTokenPacketDataV2(bz []byte) (types.FungibleTokenPacketDataV2, error) {
	var data types.FungibleTokenPacketDataV2
	if err := types.ModuleCdc.UnmarshalJSON(bz, &data); err == nil {
		return data, nil
	}
	var datav1 types.FungibleTokenPacketData
	if err := types.ModuleCdc.UnmarshalJSON(bz, &datav1); err == nil {
		return transferv2.ConvertPacketV1ToPacketV2(datav1), nil
	}
	return types.FungibleTokenPacketDataV2{}, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "cannot unmarshal ICS-20 transfer packet data")
}

// OnRecvPacket implements the IBCModule interface. A successful acknowledgement
// is returned if the packet data is successfully decoded and the receive application
// logic returns without error.
func (im IBCModule) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	logger := im.transferKeeper.Logger(ctx)
	ack := channeltypes.NewResultAcknowledgement([]byte{byte(1)})

	data, ackErr := getFungibleTokenPacketDataV2(packet.GetData())
	if ackErr != nil {
		logger.Error(fmt.Sprintf("%s sequence %d", ackErr.Error(), packet.Sequence))
		ack = channeltypes.NewErrorAcknowledgement(ackErr)
	}

	// only attempt the application logic if the packet data
	// was successfully decoded
	if ack.Success() {
		err := im.transferKeeper.OnRecvPacket(ctx, packet, data)
		if err != nil {
			ack = channeltypes.NewErrorAcknowledgement(err)
			ackErr = err
			logger.Error(fmt.Sprintf("%s sequence %d", ackErr.Error(), packet.Sequence))
		} else {
			logger.Info("successfully handled ICS-20 packet", "sequence", packet.Sequence)
		}
	}


	eventAttributes := []sdk.Attribute{
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		sdk.NewAttribute(sdk.AttributeKeySender, data.Sender),
		sdk.NewAttribute(types.AttributeKeyReceiver, data.Receiver),
		// TODO: emit these for each token in the packet
		sdk.NewAttribute(types.AttributeKeyDenom, data.Tokens[0].GetFullDenomPath()),
		sdk.NewAttribute(types.AttributeKeyAmount, fmt.Sprintf("%d", data.Tokens[0].Amount)),
		sdk.NewAttribute(types.AttributeKeyMemo, data.Memo),
		sdk.NewAttribute(types.AttributeKeyAckSuccess, fmt.Sprintf("%t", ack.Success())),
	}

	if ackErr != nil {
		eventAttributes = append(eventAttributes, sdk.NewAttribute(types.AttributeKeyAckError, ackErr.Error()))
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypePacket,
			eventAttributes...,
		),
	)

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
	var data types.FungibleTokenPacketData
	if err := types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrUnknownRequest, "cannot unmarshal ICS-20 transfer packet data: %s", err.Error())
	}

	if err := im.transferKeeper.OnAcknowledgementPacket(ctx, packet, data, ack); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypePacket,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
			sdk.NewAttribute(sdk.AttributeKeySender, data.Sender),
			sdk.NewAttribute(types.AttributeKeyReceiver, data.Receiver),
			sdk.NewAttribute(types.AttributeKeyDenom, data.Denom),
			sdk.NewAttribute(types.AttributeKeyAmount, data.Amount),
			sdk.NewAttribute(types.AttributeKeyMemo, data.Memo),
			sdk.NewAttribute(types.AttributeKeyAck, ack.String()),
		),
	)

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
	var data types.FungibleTokenPacketData
	if err := types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrUnknownRequest, "cannot unmarshal ICS-20 transfer packet data: %s", err.Error())
	}
	// refund tokens
	if err := im.transferKeeper.OnTimeoutPacket(ctx, packet, data); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeTimeout,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
			sdk.NewAttribute(types.AttributeKeyRefundReceiver, data.Sender),
			sdk.NewAttribute(types.AttributeKeyRefundDenom, data.Denom),
			sdk.NewAttribute(types.AttributeKeyRefundAmount, data.Amount),
			sdk.NewAttribute(types.AttributeKeyMemo, data.Memo),
		),
	)

	return nil
}

// OnChanUpgradeInit implements the IBCModule interface
func (im IBCModule) OnChanUpgradeInit(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, proposedVersion string) (string, error) {
	if err := ValidateTransferChannelParams(ctx, im.transferKeeper, proposedOrder, portID, channelID); err != nil {
		return "", err
	}

	if proposedVersion != types.CurrentVersion {
		return "", errorsmod.Wrapf(types.ErrInvalidVersion, "expected %s, got %s", types.CurrentVersion, proposedVersion)
	}

	return proposedVersion, nil
}

// OnChanUpgradeTry implements the IBCModule interface
func (im IBCModule) OnChanUpgradeTry(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, counterpartyVersion string) (string, error) {
	if err := ValidateTransferChannelParams(ctx, im.transferKeeper, proposedOrder, portID, channelID); err != nil {
		return "", err
	}

	if counterpartyVersion != types.CurrentVersion {
		return "", errorsmod.Wrapf(types.ErrInvalidVersion, "expected %s, got %s", types.CurrentVersion, counterpartyVersion)
	}

	return counterpartyVersion, nil
}

// OnChanUpgradeAck implements the IBCModule interface
func (IBCModule) OnChanUpgradeAck(ctx sdk.Context, portID, channelID, counterpartyVersion string) error {
	if counterpartyVersion != types.CurrentVersion {
		return errorsmod.Wrapf(types.ErrInvalidVersion, "expected %s, got %s", types.CurrentVersion, counterpartyVersion)
	}

	return nil
}

// OnChanUpgradeOpen implements the IBCModule interface
func (IBCModule) OnChanUpgradeOpen(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, proposedVersion string) {
}

// UnmarshalPacketData attempts to unmarshal the provided packet data bytes
// into a FungibleTokenPacketData. This function implements the optional
// PacketDataUnmarshaler interface required for ADR 008 support.
func (IBCModule) UnmarshalPacketData(bz []byte) (interface{}, error) {
	var packetData types.FungibleTokenPacketData
	if err := types.ModuleCdc.UnmarshalJSON(bz, &packetData); err != nil {
		return nil, err
	}

	return packetData, nil
}
