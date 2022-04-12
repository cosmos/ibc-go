package fee

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/keeper"
	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v3/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
	ibcmock "github.com/cosmos/ibc-go/v3/testing/mock"
)

var _ porttypes.Middleware = &IBCMiddleware{}

// IBCMiddlware implements the ICS26 callbacks for the fee middleware given the
// fee keeper and the underlying application.
type IBCMiddleware struct {
	*ibcmock.Module
	keeper keeper.Keeper
}

// NewIBCMiddlware creates a new IBCMiddlware given the keeper and underlying application
func NewIBCMiddleware(app porttypes.IBCModule, k keeper.Keeper) IBCMiddleware {
	return IBCMiddleware{
		Module: ibcmock.NewModule(app),
		keeper: k,
	}
}

// OnChanOpenInit implements the IBCModule interface
func (im IBCMiddleware) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) error {
	var versionMetadata types.Metadata
	if err := types.ModuleCdc.UnmarshalJSON([]byte(version), &versionMetadata); err != nil {
		// Since it is valid for fee version to not be specified, the above middleware version may be for a middleware
		// lower down in the stack. Thus, if it is not a fee version we pass the entire version string onto the underlying
		// application.
		return im.Module.OnChanOpenInit(ctx, order, connectionHops, portID, channelID,
			chanCap, counterparty, version)
	}

	if versionMetadata.FeeVersion != types.Version {
		return sdkerrors.Wrapf(types.ErrInvalidVersion, "expected %s, got %s", types.Version, versionMetadata.FeeVersion)
	}

	im.keeper.SetFeeEnabled(ctx, portID, channelID)

	// call underlying app's OnChanOpenInit callback with the appVersion
	return im.Module.OnChanOpenInit(ctx, order, connectionHops, portID, channelID,
		chanCap, counterparty, versionMetadata.AppVersion)
}

// OnChanOpenTry implements the IBCModule interface
// If the channel is not fee enabled the underlying application version will be returned
// If the channel is fee enabled we merge the underlying application version with the ics29 version
func (im IBCMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	var versionMetadata types.Metadata
	if err := types.ModuleCdc.UnmarshalJSON([]byte(counterpartyVersion), &versionMetadata); err != nil {
		// Since it is valid for fee version to not be specified, the above middleware version may be for a middleware
		// lower down in the stack. Thus, if it is not a fee version we pass the entire version string onto the underlying
		// application.
		return im.Module.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, counterpartyVersion)
	}

	if versionMetadata.FeeVersion != types.Version {
		return "", sdkerrors.Wrapf(types.ErrInvalidVersion, "expected %s, got %s", types.Version, versionMetadata.FeeVersion)
	}

	im.keeper.SetFeeEnabled(ctx, portID, channelID)

	// call underlying app's OnChanOpenTry callback with the app versions
	appVersion, err := im.Module.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, versionMetadata.AppVersion)
	if err != nil {
		return "", err
	}

	versionMetadata.AppVersion = appVersion

	versionBytes, err := types.ModuleCdc.MarshalJSON(&versionMetadata)
	if err != nil {
		return "", err
	}

	return string(versionBytes), nil
}

// OnChanOpenAck implements the IBCModule interface
func (im IBCMiddleware) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	// If handshake was initialized with fee enabled it must complete with fee enabled.
	// If handshake was initialized with fee disabled it must complete with fee disabled.
	if im.keeper.IsFeeEnabled(ctx, portID, channelID) {
		var versionMetadata types.Metadata
		if err := types.ModuleCdc.UnmarshalJSON([]byte(counterpartyVersion), &versionMetadata); err != nil {
			return sdkerrors.Wrap(types.ErrInvalidVersion, "failed to unmarshal ICS29 counterparty version metadata")
		}

		if versionMetadata.FeeVersion != types.Version {
			return sdkerrors.Wrapf(types.ErrInvalidVersion, "expected counterparty fee version: %s, got: %s", types.Version, versionMetadata.FeeVersion)
		}

		// call underlying app's OnChanOpenAck callback with the counterparty app version.
		return im.Module.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, versionMetadata.AppVersion)
	}

	// call underlying app's OnChanOpenAck callback with the counterparty app version.
	return im.Module.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

// OnChanOpenConfirm implements the IBCModule interface
func (im IBCMiddleware) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// call underlying app's OnChanOpenConfirm callback.
	return im.Module.OnChanOpenConfirm(ctx, portID, channelID)
}

// OnChanCloseInit implements the IBCModule interface
func (im IBCMiddleware) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	if err := im.Module.OnChanCloseInit(ctx, portID, channelID); err != nil {
		return err
	}

	// delete fee enabled on channel
	// and refund any remaining fees escrowed on channel
	im.keeper.DeleteFeeEnabled(ctx, portID, channelID)
	err := im.keeper.RefundFeesOnChannel(ctx, portID, channelID)
	// error should only be non-nil if there is a bug in the code
	// that causes module account to have insufficient funds to refund
	// all escrowed fees on the channel.
	// Disable all channels to allow for coordinated fix to the issue
	// and mitigate/reverse damage.
	// NOTE: Underlying application's packets will still go through, but
	// fee module will be disabled for all channels
	if err != nil {
		im.keeper.DisableAllChannels(ctx)
	}

	return nil
}

// OnChanCloseConfirm implements the IBCModule interface
func (im IBCMiddleware) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// delete fee enabled on channel
	// and refund any remaining fees escrowed on channel
	im.keeper.DeleteFeeEnabled(ctx, portID, channelID)
	err := im.keeper.RefundFeesOnChannel(ctx, portID, channelID)
	// error should only be non-nil if there is a bug in the code
	// that causes module account to have insufficient funds to refund
	// all escrowed fees on the channel.
	// Disable all channels to allow for coordinated fix to the issue
	// and mitigate/reverse damage.
	// NOTE: Underlying application's packets will still go through, but
	// fee module will be disabled for all channels
	if err != nil {
		im.keeper.DisableAllChannels(ctx)
	}
	return im.Module.OnChanCloseConfirm(ctx, portID, channelID)
}

// OnRecvPacket implements the IBCModule interface.
// If fees are not enabled, this callback will default to the ibc-core packet callback
func (im IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) exported.Acknowledgement {
	if !im.keeper.IsFeeEnabled(ctx, packet.DestinationPort, packet.DestinationChannel) {
		return im.Module.OnRecvPacket(ctx, packet, relayer)
	}

	ack := im.Module.OnRecvPacket(ctx, packet, relayer)

	// incase of async aknowledgement (ack == nil) store the relayer address for use later during async WriteAcknowledgement
	if ack == nil {
		im.keeper.SetRelayerAddressForAsyncAck(ctx, channeltypes.NewPacketId(packet.GetDestChannel(), packet.GetDestPort(), packet.GetSequence()), relayer.String())
		return nil
	}

	// if forwardRelayer is not found we refund recv_fee
	forwardRelayer, _ := im.keeper.GetCounterpartyAddress(ctx, relayer.String(), packet.GetDestChannel())

	return types.NewIncentivizedAcknowledgement(forwardRelayer, ack.Acknowledgement(), ack.Success())
}

// OnAcknowledgementPacket implements the IBCModule interface
// If fees are not enabled, this callback will default to the ibc-core packet callback
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	if !im.keeper.IsFeeEnabled(ctx, packet.SourcePort, packet.SourceChannel) {
		return im.Module.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
	}

	ack := new(types.IncentivizedAcknowledgement)
	if err := types.ModuleCdc.UnmarshalJSON(acknowledgement, ack); err != nil {
		return sdkerrors.Wrapf(err, "cannot unmarshal ICS-29 incentivized packet acknowledgement: %v", ack)
	}

	packetID := channeltypes.NewPacketId(packet.SourceChannel, packet.SourcePort, packet.Sequence)
	feesInEscrow, found := im.keeper.GetFeesInEscrow(ctx, packetID)
	if found {
		im.keeper.DistributePacketFees(ctx, ack.ForwardRelayerAddress, relayer, feesInEscrow.PacketFees)

		// removes the fees from the store as fees are now paid
		im.keeper.DeleteFeesInEscrow(ctx, packetID)
	}

	// call underlying callback
	return im.Module.OnAcknowledgementPacket(ctx, packet, ack.Result, relayer)
}

// OnTimeoutPacket implements the IBCModule interface
// If fees are not enabled, this callback will default to the ibc-core packet callback
func (im IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	if !im.keeper.IsFeeEnabled(ctx, packet.SourcePort, packet.SourceChannel) {
		return im.Module.OnTimeoutPacket(ctx, packet, relayer)
	}

	packetID := channeltypes.NewPacketId(packet.SourceChannel, packet.SourcePort, packet.Sequence)
	feesInEscrow, found := im.keeper.GetFeesInEscrow(ctx, packetID)
	if found {
		im.keeper.DistributePacketFeesOnTimeout(ctx, relayer, feesInEscrow.PacketFees)

		// removes the fee from the store as fee is now paid
		im.keeper.DeleteFeesInEscrow(ctx, packetID)
	}

	// call underlying callback
	return im.Module.OnTimeoutPacket(ctx, packet, relayer)
}

// SendPacket implements the ICS4 Wrapper interface
func (im IBCMiddleware) SendPacket(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet exported.PacketI,
) error {
	return im.keeper.SendPacket(ctx, chanCap, packet)
}

// WriteAcknowledgement implements the ICS4 Wrapper interface
func (im IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet exported.PacketI,
	ack exported.Acknowledgement,
) error {
	return im.keeper.WriteAcknowledgement(ctx, chanCap, packet, ack)
}
