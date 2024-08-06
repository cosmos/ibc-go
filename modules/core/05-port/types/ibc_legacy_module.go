package types

import (
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	ibcexported "github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// LegacyIBCModule implements the ICS26 interface for transfer given the transfer keeper.
type LegacyIBCModule struct {
	cbs []ClassicIBCModule
}

// TODO: added this for testing purposes, we can remove later if tests are refactored.
func (im *LegacyIBCModule) GetCallbacks() []ClassicIBCModule {
	return im.cbs
}

// NewLegacyIBCModule creates a new IBCModule given the keeper
func NewLegacyIBCModule(cbs ...ClassicIBCModule) ClassicIBCModule {
	return &LegacyIBCModule{
		cbs: cbs,
	}
}

// OnChanOpenInit implements the IBCModule interface
func (im *LegacyIBCModule) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	var (
		negotiatedVersions = make([]string, len(im.cbs))
	)

	for i := len(im.cbs) - 1; i >= 0; i-- {
		cbVersion := version

		// TODO: document behaviour
		// handles default empty string + not using middleware when desired
		if wrapper, ok := im.cbs[i].(VersionWrapper); ok {
			appVersion, underlyingAppVersion, err := wrapper.UnwrapVersionUnsafe(version)
			if err != nil {
				// middleware disabled
				negotiatedVersions[i] = ""
				continue
			}
			cbVersion, version = appVersion, underlyingAppVersion
		}

		negotiatedVersion, err := im.cbs[i].OnChanOpenInit(ctx, order, connectionHops, portID, channelID, counterparty, cbVersion)
		if err != nil {
			return "", errorsmod.Wrapf(err, "channel open init callback failed for port ID: %s, channel ID: %s", portID, channelID)
		}
		negotiatedVersions[i] = negotiatedVersion
	}

	return reconstructVersion(im.cbs, negotiatedVersions)
}

// glue back proposed version for channel storage
func reconstructVersion(cbs []ClassicIBCModule, negotiatedVersions []string) (string, error) {
	version := negotiatedVersions[0]
	for i := 1; i < len(cbs); i++ {
		if strings.TrimSpace(negotiatedVersions[i]) != "" {
			wrapper, ok := cbs[i].(VersionWrapper)
			if !ok {
				return "", ibcerrors.ErrInvalidVersion
			}
			version = wrapper.WrapVersion(negotiatedVersions[i], version)
		}
	}
	return version, nil
}

// OnChanOpenTry implements the IBCModule interface.
func (LegacyIBCModule) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	return "", nil
}

// OnChanOpenAck implements the IBCModule interface
func (LegacyIBCModule) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	return nil
}

// OnChanOpenConfirm implements the IBCModule interface
func (LegacyIBCModule) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return nil
}

// OnChanCloseInit implements the IBCModule interface
func (LegacyIBCModule) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return nil
}

// OnChanCloseConfirm implements the IBCModule interface
func (LegacyIBCModule) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return nil
}

// OnSendPacket implements the IBCModule interface.
func (im LegacyIBCModule) OnSendPacket(
	ctx sdk.Context,
	portID string,
	channelID string,
	sequence uint64,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	dataBz []byte,
	signer sdk.AccAddress,
) error {
	// to maintain backwards compatibility, OnSendPacket iterates over the callbacks in order, as they are wired from bottom to top of the stack.
	for _, cb := range im.cbs {
		if err := cb.OnSendPacket(ctx, portID, channelID, sequence, timeoutHeight, timeoutTimestamp, dataBz, signer); err != nil {
			return errorsmod.Wrapf(err, "send packet callback failed for portID %s channelID %s", portID, channelID)
		}
	}
	return nil
}

// OnRecvPacket implements the IBCModule interface. A successful acknowledgement
// is returned if the packet data is successfully decoded and the receive application
// logic returns without error.
// A nil acknowledgement may be returned when using the packet forwarding feature. This signals to core IBC that the acknowledgement will be written asynchronously.
func (LegacyIBCModule) OnRecvPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	return nil
}

// OnAcknowledgementPacket implements the IBCModule interface
func (LegacyIBCModule) OnAcknowledgementPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	return nil
}

// OnTimeoutPacket implements the IBCModule interface
func (LegacyIBCModule) OnTimeoutPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	return nil
}

// OnChanUpgradeInit implements the IBCModule interface
func (LegacyIBCModule) OnChanUpgradeInit(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, proposedVersion string) (string, error) {
	return "", nil
}

// OnChanUpgradeTry implements the IBCModule interface
func (LegacyIBCModule) OnChanUpgradeTry(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, counterpartyVersion string) (string, error) {
	return "", nil
}

// OnChanUpgradeAck implements the IBCModule interface
func (LegacyIBCModule) OnChanUpgradeAck(ctx sdk.Context, portID, channelID, counterpartyVersion string) error {
	return nil
}

// OnChanUpgradeOpen implements the IBCModule interface
func (LegacyIBCModule) OnChanUpgradeOpen(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, proposedVersion string) {
}

// UnmarshalPacketData attempts to unmarshal the provided packet data bytes
// into a FungibleTokenPacketData. This function implements the optional
// PacketDataUnmarshaler interface required for ADR 008 support.
func (LegacyIBCModule) UnmarshalPacketData(ctx sdk.Context, portID, channelID string, bz []byte) (interface{}, error) {
	return nil, nil
}
