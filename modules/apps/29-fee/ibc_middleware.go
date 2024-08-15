package fee

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/29-fee/keeper"
	"github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v9/modules/core/05-port/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

var (
	_ porttypes.Middleware             = (*IBCMiddleware)(nil)
	_ porttypes.PacketDataUnmarshaler  = (*IBCMiddleware)(nil)
	_ porttypes.UpgradableModule       = (*IBCMiddleware)(nil)
	_ porttypes.VersionWrapper         = (*IBCMiddleware)(nil)
	_ porttypes.AcknowledgementWrapper = (*IBCMiddleware)(nil)
)

// IBCMiddleware implements the ICS26 callbacks for the fee middleware given the
// fee keeper and the underlying application.
type IBCMiddleware struct {
	app    porttypes.ClassicIBCModule
	keeper keeper.Keeper
}

// NewIBCMiddleware creates a new IBCMiddleware given the keeper and underlying application
func NewIBCMiddleware(app porttypes.ClassicIBCModule, k keeper.Keeper) IBCMiddleware {
	return IBCMiddleware{
		app:    app,
		keeper: k,
	}
}

// OnChanOpenInit implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	if strings.TrimSpace(version) != "" && version != types.Version {
		return "", errorsmod.Wrapf(types.ErrInvalidVersion, "expected %s, got %s", types.Version, version)
	}

	im.keeper.SetFeeEnabled(ctx, portID, channelID)
	return types.Version, nil
}

// OnChanOpenTry implements the IBCMiddleware interface
// If the channel is not fee enabled the underlying application version will be returned
// If the channel is fee enabled we merge the underlying application version with the ics29 version
func (im IBCMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	if counterpartyVersion != types.Version {
		return "", errorsmod.Wrapf(types.ErrInvalidVersion, "expected %s, got %s", types.Version, counterpartyVersion)
	}
	im.keeper.SetFeeEnabled(ctx, portID, channelID)
	return counterpartyVersion, nil
}

// OnChanOpenAck implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	if strings.TrimSpace(counterpartyVersion) == "" {
		// disable fees for this channel
		im.keeper.DeleteFeeEnabled(ctx, portID, channelID)
		return nil
	}

	if counterpartyVersion != types.Version {
		return errorsmod.Wrapf(types.ErrInvalidVersion, "expected counterparty fee version: %s, got: %s", types.Version, counterpartyVersion)
	}

	im.keeper.SetFeeEnabled(ctx, portID, channelID) // TODO: can we delete this line
	return nil
}

// OnChanOpenConfirm implements the IBCMiddleware interface
func (IBCMiddleware) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return nil
}

// OnChanCloseInit implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	if !im.keeper.IsFeeEnabled(ctx, portID, channelID) {
		return nil
	}

	if im.keeper.IsLocked(ctx) {
		return types.ErrFeeModuleLocked
	}

	return im.keeper.RefundFeesOnChannelClosure(ctx, portID, channelID)
}

// OnChanCloseConfirm implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	if !im.keeper.IsFeeEnabled(ctx, portID, channelID) {
		return nil
	}

	if im.keeper.IsLocked(ctx) {
		return types.ErrFeeModuleLocked
	}

	return im.keeper.RefundFeesOnChannelClosure(ctx, portID, channelID)
}

// OnSendPacket implements the IBCModule interface.
func (IBCMiddleware) OnSendPacket(
	ctx sdk.Context,
	portID string,
	channelID string,
	sequence uint64,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
	signer sdk.AccAddress,
) error {
	return nil
}

// OnRecvPacket implements the IBCMiddleware interface.
// If fees are not enabled, this callback will default to the ibc-core packet callback
func (im IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) exported.RecvPacketResult {
	if !im.keeper.IsFeeEnabled(ctx, packet.DestinationPort, packet.DestinationChannel) {
		return exported.RecvPacketResult{Status: exported.Success}
	}

	forwardRelayer, _ := im.keeper.GetCounterpartyPayeeAddress(ctx, relayer.String(), packet.GetDestChannel())

	feeAcknowledgement := types.NewFeeAcknowledgement(forwardRelayer)

	ack, err := json.Marshal(feeAcknowledgement)
	if err != nil {
		panic(errors.New("cannot marshal acknowledgement into json"))
	}

	return exported.RecvPacketResult{Status: exported.Success, Acknowledgement: ack}
}

// OnAcknowledgementPacket implements the IBCMiddleware interface
// If fees are not enabled, this callback will default to the ibc-core packet callback
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	if !im.keeper.IsFeeEnabled(ctx, packet.SourcePort, packet.SourceChannel) {
		return nil
	}

	var ack types.FeeAcknowledgement
	if err := json.Unmarshal(acknowledgement, &ack); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "cannot unmarshal ICS-29 incentivized packet acknowledgement %v: %s", ack, err)
	}

	if im.keeper.IsLocked(ctx) {
		// if the fee keeper is locked then fee logic should be skipped
		// this may occur in the presence of a severe bug which leads to invalid state
		// the fee keeper will be unlocked after manual intervention
		// the acknowledgement has been unmarshalled into an ics29 acknowledgement
		// since the counterparty is still sending incentivized acknowledgements
		// for fee enabled channels
		//
		// Please see ADR 004 for more information.
		return nil
	}

	packetID := channeltypes.NewPacketID(packet.SourcePort, packet.SourceChannel, packet.Sequence)
	feesInEscrow, found := im.keeper.GetFeesInEscrow(ctx, packetID)
	if !found {
		return nil
	}

	payee, found := im.keeper.GetPayeeAddress(ctx, relayer.String(), packet.SourceChannel)
	if !found {
		payee = relayer.String()
	}

	payeeAddr, err := sdk.AccAddressFromBech32(payee)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to create sdk.Address from payee: %s", payee)
	}

	im.keeper.DistributePacketFeesOnAcknowledgement(ctx, ack.ForwardRelayerAddress, payeeAddr, feesInEscrow.PacketFees, packetID)
	return nil
}

// OnTimeoutPacket implements the IBCMiddleware interface
// If fees are not enabled, this callback will default to the ibc-core packet callback
func (im IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	if !im.keeper.IsFeeEnabled(ctx, packet.SourcePort, packet.SourceChannel) {
		return nil
	}

	// if the fee keeper is locked then fee logic should be skipped
	// this may occur in the presence of a severe bug which leads to invalid state
	// the fee keeper will be unlocked after manual intervention
	//
	// Please see ADR 004 for more information.
	if im.keeper.IsLocked(ctx) {
		return nil
	}

	packetID := channeltypes.NewPacketID(packet.SourcePort, packet.SourceChannel, packet.Sequence)
	feesInEscrow, found := im.keeper.GetFeesInEscrow(ctx, packetID)
	if !found {
		return nil
	}

	payee, found := im.keeper.GetPayeeAddress(ctx, relayer.String(), packet.SourceChannel)
	if !found {
		payee = relayer.String()
	}

	payeeAddr, err := sdk.AccAddressFromBech32(payee)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to create sdk.Address from payee: %s", payee)
	}

	im.keeper.DistributePacketFeesOnTimeout(ctx, payeeAddr, feesInEscrow.PacketFees, packetID)

	// call underlying callback
	return nil
}

// OnChanUpgradeInit implements the IBCModule interface
func (IBCMiddleware) OnChanUpgradeInit(
	ctx sdk.Context,
	portID string,
	channelID string,
	proposedOrder channeltypes.Order,
	proposedConnectionHops []string,
	proposedVersion string,
) (string, error) {
	if proposedVersion != types.Version {
		return "", errorsmod.Wrapf(types.ErrInvalidVersion, "expected %s, got %s", types.Version, proposedVersion)
	}
	return types.Version, nil
}

// OnChanUpgradeTry implements the IBCModule interface
func (IBCMiddleware) OnChanUpgradeTry(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, counterpartyVersion string) (string, error) {
	if counterpartyVersion != types.Version {
		return "", errorsmod.Wrapf(types.ErrInvalidVersion, "expected %s, got %s", types.Version, counterpartyVersion)
	}
	return counterpartyVersion, nil
}

// OnChanUpgradeAck implements the IBCModule interface
func (IBCMiddleware) OnChanUpgradeAck(ctx sdk.Context, portID, channelID, counterpartyVersion string) error {
	if counterpartyVersion != types.Version {
		return errorsmod.Wrapf(types.ErrInvalidVersion, "expected counterparty fee version: %s, got: %s", types.Version, counterpartyVersion)
	}
	return nil
}

// OnChanUpgradeOpen implements the IBCModule interface
func (im IBCMiddleware) OnChanUpgradeOpen(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, proposedVersion string) {
	if strings.TrimSpace(proposedVersion) == "" {
		// set fee disabled and pass through to the next middleware or application in callstack.
		im.keeper.DeleteFeeEnabled(ctx, portID, channelID)
		return
	}

	// set fee enabled and pass through to the next middleware of application in callstack.
	im.keeper.SetFeeEnabled(ctx, portID, channelID)
}

// WriteAcknowledgement implements the ICS4 Wrapper interface
func (im IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	packet exported.PacketI,
	ack []byte,
) error {
	return im.keeper.WriteAcknowledgement(ctx, packet, ack)
}

// GetAppVersion returns the application version of the underlying application
func (im IBCMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return im.keeper.GetAppVersion(ctx, portID, channelID)
}

// UnmarshalPacketData attempts to use the underlying app to unmarshal the packet data.
// If the underlying app does not support the PacketDataUnmarshaler interface, an error is returned.
// This function implements the optional PacketDataUnmarshaler interface required for ADR 008 support.
func (im IBCMiddleware) UnmarshalPacketData(ctx sdk.Context, portID string, channelID string, bz []byte) (interface{}, string, error) {
	unmarshaler, ok := im.app.(porttypes.PacketDataUnmarshaler)
	if !ok {
		return nil, "", errorsmod.Wrapf(types.ErrUnsupportedAction, "underlying app does not implement %T", (*porttypes.PacketDataUnmarshaler)(nil))
	}

	return unmarshaler.UnmarshalPacketData(ctx, portID, channelID, bz)
}

// WrapVersion returns the wrapped ics29 version based on the provided ics29 version and the underlying application version.
func (IBCMiddleware) WrapVersion(cbVersion, underlyingAppVersion string) string {
	if cbVersion != types.Version {
		panic(fmt.Errorf("invalid ics29 version provided. expected: %s, got: %s", types.Version, cbVersion))
	}

	metadata := types.Metadata{
		FeeVersion: cbVersion,
		AppVersion: underlyingAppVersion,
	}

	versionBytes := types.ModuleCdc.MustMarshalJSON(&metadata)

	return string(versionBytes)
}

// UnwrapVersionUnsafe attempts to unmarshal the version string into a ics29 version. An error is returned if unsuccessful.
func (IBCMiddleware) UnwrapVersionUnsafe(version string) (string, string, error) {
	metadata, err := types.MetadataFromVersion(version)
	if err != nil {
		// not an ics29 version
		return "", version, err
	}

	return metadata.FeeVersion, metadata.AppVersion, nil
}

// UnwrapVersionSafe unwraps a version contextually by relying on storage and the given portID and channelID.
func (im IBCMiddleware) UnwrapVersionSafe(ctx sdk.Context, portID, channelID, version string) (string, string) {
	if !im.keeper.IsFeeEnabled(ctx, portID, channelID) {
		return "", version
	}
	metadata, err := types.MetadataFromVersion(version)
	if err != nil {
		// This should not happen, as it would mean that the channel is broken. Only a severe bug would cause this.
		panic(errorsmod.Wrap(err, "failed to unwrap app version from channel version"))
	}
	return metadata.FeeVersion, metadata.AppVersion
}

func (im IBCMiddleware) WrapAcknowledgement(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress, prevResult, result exported.RecvPacketResult) exported.RecvPacketResult {
	if !im.keeper.IsFeeEnabled(ctx, packet.GetDestPort(), packet.GetDestChannel()) {
		return prevResult
	}

	var feeAck types.FeeAcknowledgement
	if err := json.Unmarshal(result.Acknowledgement, &feeAck); err != nil {
		panic(errorsmod.Wrap(err, "failed to wrap acknowledgement"))
	}

	return exported.RecvPacketResult{
		Status:          prevResult.Status,
		Acknowledgement: types.NewIncentivizedAcknowledgement(feeAck.ForwardRelayerAddress, prevResult.Acknowledgement, prevResult.Status == exported.Success).Acknowledgement(),
	}
}

// UnwrapAcknowledgement unwraps an acnkowledgement contextually by relying on storage and the given portID and channelID.
func (im IBCMiddleware) UnwrapAcknowledgement(ctx sdk.Context, portID, channelID string, ack []byte) ([]byte, []byte) {
	if !im.keeper.IsFeeEnabled(ctx, portID, channelID) {
		return nil, ack
	}

	var incentivizedAck types.IncentivizedAcknowledgement
	if err := json.Unmarshal(ack, &incentivizedAck); err != nil {
		panic(errorsmod.Wrap(err, "failed to unwrap acknowledgement"))
	}

	return types.NewFeeAcknowledgement(incentivizedAck.ForwardRelayerAddress).Acknowledgement(), incentivizedAck.AppAcknowledgement
}
