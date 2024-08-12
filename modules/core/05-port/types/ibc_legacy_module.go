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

// OnChanOpenInit implements the IBCModule interface.
// NOTE: The application callback is skipped if all the following are true:
// - the relayer provided channel version is not empty
// - the callback application is a VersionWrapper
// - the application cannot unwrap the version
func (im *LegacyIBCModule) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	negotiatedVersions := make([]string, len(im.cbs))

	for i := len(im.cbs) - 1; i >= 0; i-- {
		cbVersion := version

		// To maintain backwards compatibility, we must handle two cases:
		// - relayer provides empty version (use default versions)
		// - relayer provides version which chooses to not enable a middleware
		//
		// If an application is a VersionWrapper which means it modifies the version string
		// and the version string is non-empty (don't use default), then the application must
		// attempt to unmarshal the version using the UnwrapVersionUnsafe interface function.
		// If it is unsuccessful, no callback will occur to this application as the version
		// indicates it should be disabled.
		if wrapper, ok := im.cbs[i].(VersionWrapper); ok && strings.TrimSpace(version) != "" {
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

// OnChanOpenTry implements the IBCModule interface.
// NOTE: The application callback is skipped if all the following are true:
// - the relayer provided channel version is not empty
// - the callback application is a VersionWrapper
// - the application cannot unwrap the version
func (im *LegacyIBCModule) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	negotiatedVersions := make([]string, len(im.cbs))

	for i := len(im.cbs) - 1; i >= 0; i-- {
		cbVersion := counterpartyVersion

		// To maintain backwards compatibility, we must handle two cases:
		// - relayer provides empty version (use default versions)
		// - relayer provides version which chooses to not enable a middleware
		//
		// If an application is a VersionWrapper which means it modifies the version string
		// and the version string is non-empty (don't use default), then the application must
		// attempt to unmarshal the version using the UnwrapVersionUnsafe interface function.
		// If it is unsuccessful, no callback will occur to this application as the version
		// indicates it should be disabled.
		if wrapper, ok := im.cbs[i].(VersionWrapper); ok && strings.TrimSpace(counterpartyVersion) != "" {
			appVersion, underlyingAppVersion, err := wrapper.UnwrapVersionUnsafe(counterpartyVersion)
			if err != nil {
				// middleware disabled
				negotiatedVersions[i] = ""
				continue
			}
			cbVersion, counterpartyVersion = appVersion, underlyingAppVersion
		}

		negotiatedVersion, err := im.cbs[i].OnChanOpenTry(ctx, order, connectionHops, portID, channelID, counterparty, cbVersion)
		if err != nil {
			return "", errorsmod.Wrapf(err, "channel open try callback failed for port ID: %s, channel ID: %s", portID, channelID)
		}
		negotiatedVersions[i] = negotiatedVersion
	}

	return reconstructVersion(im.cbs, negotiatedVersions)
}

// OnChanOpenAck implements the IBCModule interface.
// NOTE: The callback will occur for all applications in the callback list.
// If the application is provided an empty string for the counterparty version,
// this indicates the module should be disabled for this portID and channelID.
func (im *LegacyIBCModule) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	for i := len(im.cbs) - 1; i >= 0; i-- {
		cbVersion := counterpartyVersion

		// To maintain backwards compatibility, we must handle counterparty version negotiation.
		// This means the version may have changed, and applications must be allowed to be disabled.
		// Applications should be disabled when receiving an empty counterparty version. Callbacks
		// for all applications must occur to allow disabling.
		if wrapper, ok := im.cbs[i].(VersionWrapper); ok {
			appVersion, underlyingAppVersion, err := wrapper.UnwrapVersionUnsafe(counterpartyVersion)
			if err != nil {
				cbVersion = "" // disable application
			} else {
				cbVersion, counterpartyVersion = appVersion, underlyingAppVersion
			}
		}

		err := im.cbs[i].OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, cbVersion)
		if err != nil {
			return errorsmod.Wrapf(err, "channel open ack callback failed for port ID: %s, channel ID: %s", portID, channelID)
		}
	}

	return nil
}

// OnChanOpenConfirm implements the IBCModule interface
func (im *LegacyIBCModule) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	for i := len(im.cbs) - 1; i >= 0; i-- {
		err := im.cbs[i].OnChanOpenConfirm(ctx, portID, channelID)
		if err != nil {
			return errorsmod.Wrapf(err, "channel open confirm callback failed for port ID: %s, channel ID: %s", portID, channelID)
		}
	}
	return nil
}

// OnChanCloseInit implements the IBCModule interface
func (im *LegacyIBCModule) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	for i := len(im.cbs) - 1; i >= 0; i-- {
		if err := im.cbs[i].OnChanCloseInit(ctx, portID, channelID); err != nil {
			return errorsmod.Wrapf(err, "channel close init callback failed for port ID: %s, channel ID: %s", portID, channelID)
		}
	}
	return nil
}

// OnChanCloseConfirm implements the IBCModule interface
func (im *LegacyIBCModule) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	for i := len(im.cbs) - 1; i >= 0; i-- {
		if err := im.cbs[i].OnChanCloseConfirm(ctx, portID, channelID); err != nil {
			return errorsmod.Wrapf(err, "channel close confirm callback failed for port ID: %s, channel ID: %s", portID, channelID)
		}
	}
	return nil
}

// OnSendPacket implements the IBCModule interface.
func (im *LegacyIBCModule) OnSendPacket(
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
func (im *LegacyIBCModule) OnRecvPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.RecvPacketResult {
	// NOTE: loop in reverse order calling each module's OnRecvPacket callback
	// Collect each recv packet result for the ordered list
	//
	// e.g. Stack: [transfer, 29-fee]
	// 29-fee -> Result
	// transfer -> Result
	//
	// ResultList: [29-fee, transfer]
	var resultList []ibcexported.RecvPacketResult
	for i := len(im.cbs) - 1; i >= 0; i-- {
		cbVersion := channelVersion

		if wrapper, ok := im.cbs[i].(VersionWrapper); ok {
			cbVersion, channelVersion = wrapper.UnwrapVersionSafe(ctx, packet.SourcePort, packet.SourceChannel, cbVersion)
		}

		res := im.cbs[i].OnRecvPacket(ctx, cbVersion, packet, relayer)
		resultList = append(resultList, res)
	}

	// Once we have collected each result, then iterate in order and wrap ackwnoledgement if needed.
	// tldr; reduce resultList to result
	finalResult := resultList[len(resultList)-1]
	for i := len(resultList) - 2; i >= 0; i-- {
		if wrapper, ok := im.cbs[i].(AcknowledgementWrapper); ok {
			finalResult = wrapper.WrapAcknowledgement(ctx, packet, relayer, finalResult)
		}
	}

	return finalResult
}

// OnAcknowledgementPacket implements the IBCModule interface
func (im *LegacyIBCModule) OnAcknowledgementPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	for i := len(im.cbs) - 1; i >= 0; i-- {
		var (
			cbVersion = channelVersion
			cbAck     = acknowledgement
		)

		if wrapper, ok := im.cbs[i].(VersionWrapper); ok {
			cbVersion, channelVersion = wrapper.UnwrapVersionSafe(ctx, packet.SourcePort, packet.SourceChannel, cbVersion)
		}

		if wrapper, ok := im.cbs[i].(AcknowledgementWrapper); ok {
			cbAck, acknowledgement = wrapper.UnwrapAcknowledgement(ctx, packet.SourcePort, packet.SourceChannel, cbAck)
		}

		err := im.cbs[i].OnAcknowledgementPacket(ctx, cbVersion, packet, cbAck, relayer)
		if err != nil {
			return errorsmod.Wrap(err, "acknowledge packet callback failed")
		}
	}
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
func (im *LegacyIBCModule) OnChanUpgradeInit(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, proposedVersion string) (string, error) {
	negotiatedVersions := make([]string, len(im.cbs))

	for i := len(im.cbs) - 1; i >= 0; i-- {
		cbVersion := proposedVersion

		// To maintain backwards compatibility, we must handle two cases:
		// - relayer provides empty version (use default versions)
		// - relayer provides version which chooses to not enable a middleware
		//
		// If an application is a VersionWrapper which means it modifies the version string
		// and the version string is non-empty (don't use default), then the application must
		// attempt to unmarshal the version using the UnwrapVersionUnsafe interface function.
		// If it is unsuccessful, no callback will occur to this application as the version
		// indicates it should be disabled.
		if wrapper, ok := im.cbs[i].(VersionWrapper); ok && strings.TrimSpace(proposedVersion) != "" {
			appVersion, underlyingAppVersion, err := wrapper.UnwrapVersionUnsafe(proposedVersion)
			if err != nil {
				// middleware disabled
				negotiatedVersions[i] = ""
				continue
			}
			cbVersion, proposedVersion = appVersion, underlyingAppVersion
		}

		// in order to maintain backwards compatibility, every callback in the stack must implement the UpgradableModule interface.
		upgradableModule, ok := im.cbs[i].(UpgradableModule)
		if !ok {
			return "", errorsmod.Wrap(ErrInvalidRoute, "upgrade route not found to module in application callstack")
		}

		negotiatedVersion, err := upgradableModule.OnChanUpgradeInit(ctx, portID, channelID, proposedOrder, proposedConnectionHops, cbVersion)
		if err != nil {
			return "", errorsmod.Wrapf(err, "channel open init callback failed for port ID: %s, channel ID: %s", portID, channelID)
		}
		negotiatedVersions[i] = negotiatedVersion
	}

	return reconstructVersion(im.cbs, negotiatedVersions)
}

// OnChanUpgradeTry implements the IBCModule interface
func (im *LegacyIBCModule) OnChanUpgradeTry(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, counterpartyVersion string) (string, error) {
	negotiatedVersions := make([]string, len(im.cbs))

	for i := len(im.cbs) - 1; i >= 0; i-- {
		cbVersion := counterpartyVersion

		// To maintain backwards compatibility, we must handle two cases:
		// - relayer provides empty version (use default versions)
		// - relayer provides version which chooses to not enable a middleware
		//
		// If an application is a VersionWrapper which means it modifies the version string
		// and the version string is non-empty (don't use default), then the application must
		// attempt to unmarshal the version using the UnwrapVersionUnsafe interface function.
		// If it is unsuccessful, no callback will occur to this application as the version
		// indicates it should be disabled.
		if wrapper, ok := im.cbs[i].(VersionWrapper); ok && strings.TrimSpace(counterpartyVersion) != "" {
			appVersion, underlyingAppVersion, err := wrapper.UnwrapVersionUnsafe(counterpartyVersion)
			if err != nil {
				// middleware disabled
				negotiatedVersions[i] = ""
				continue
			}
			cbVersion, counterpartyVersion = appVersion, underlyingAppVersion
		}

		// in order to maintain backwards compatibility, every callback in the stack must implement the UpgradableModule interface.
		upgradableModule, ok := im.cbs[i].(UpgradableModule)
		if !ok {
			return "", errorsmod.Wrap(ErrInvalidRoute, "upgrade route not found to module in application callstack")
		}

		negotiatedVersion, err := upgradableModule.OnChanUpgradeTry(ctx, portID, channelID, proposedOrder, proposedConnectionHops, cbVersion)
		if err != nil {
			return "", errorsmod.Wrapf(err, "channel open init callback failed for port ID: %s, channel ID: %s", portID, channelID)
		}
		negotiatedVersions[i] = negotiatedVersion
	}

	return reconstructVersion(im.cbs, negotiatedVersions)
}

// OnChanUpgradeAck implements the IBCModule interface
func (im *LegacyIBCModule) OnChanUpgradeAck(ctx sdk.Context, portID, channelID, counterpartyVersion string) error {
	for i := len(im.cbs) - 1; i >= 0; i-- {
		cbVersion := counterpartyVersion

		// To maintain backwards compatibility, we must handle two cases:
		// - relayer provides empty version (use default versions)
		// - relayer provides version which chooses to not enable a middleware
		//
		// If an application is a VersionWrapper which means it modifies the version string
		// and the version string is non-empty (don't use default), then the application must
		// attempt to unmarshal the version using the UnwrapVersionUnsafe interface function.
		// If it is unsuccessful, no callback will occur to this application as the version
		// indicates it should be disabled.
		if wrapper, ok := im.cbs[i].(VersionWrapper); ok && strings.TrimSpace(counterpartyVersion) != "" {
			appVersion, underlyingAppVersion, err := wrapper.UnwrapVersionUnsafe(counterpartyVersion)
			if err != nil {
				// middleware disabled
				continue
			}
			cbVersion, counterpartyVersion = appVersion, underlyingAppVersion
		}

		// in order to maintain backwards compatibility, every callback in the stack must implement the UpgradableModule interface.
		upgradableModule, ok := im.cbs[i].(UpgradableModule)
		if !ok {
			return errorsmod.Wrap(ErrInvalidRoute, "upgrade route not found to module in application callstack")
		}

		err := upgradableModule.OnChanUpgradeAck(ctx, portID, channelID, cbVersion)
		if err != nil {
			return errorsmod.Wrapf(err, "channel open init callback failed for port ID: %s, channel ID: %s", portID, channelID)
		}
	}
	return nil
}

// OnChanUpgradeOpen implements the IBCModule interface
func (im *LegacyIBCModule) OnChanUpgradeOpen(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, proposedVersion string) {
	for i := len(im.cbs) - 1; i >= 0; i-- {
		cbVersion := proposedVersion

		// To maintain backwards compatibility, we must handle two cases:
		// - relayer provides empty version (use default versions)
		// - relayer provides version which chooses to not enable a middleware
		//
		// If an application is a VersionWrapper which means it modifies the version string
		// and the version string is non-empty (don't use default), then the application must
		// attempt to unmarshal the version using the UnwrapVersionUnsafe interface function.
		// If it is unsuccessful, no callback will occur to this application as the version
		// indicates it should be disabled.
		if wrapper, ok := im.cbs[i].(VersionWrapper); ok {
			appVersion, underlyingAppVersion, err := wrapper.UnwrapVersionUnsafe(proposedVersion)
			if err != nil {
				cbVersion = "" // disable application
			} else {
				cbVersion, proposedVersion = appVersion, underlyingAppVersion
			}
		}

		// in order to maintain backwards compatibility, every callback in the stack must implement the UpgradableModule interface.
		upgradableModule, ok := im.cbs[i].(UpgradableModule)
		if !ok {
			panic(errorsmod.Wrap(ErrInvalidRoute, "upgrade route not found to module in application callstack"))
		}

		upgradableModule.OnChanUpgradeOpen(ctx, portID, channelID, proposedOrder, proposedConnectionHops, cbVersion)
	}
}

// UnmarshalPacketData attempts to unmarshal the provided packet data bytes
// into a FungibleTokenPacketData. This function implements the optional
// PacketDataUnmarshaler interface required for ADR 008 support.
func (LegacyIBCModule) UnmarshalPacketData(ctx sdk.Context, portID, channelID string, bz []byte) (interface{}, error) {
	return nil, nil
}

// reconstructVersion will generate the channel version by applying any version wrapping as necessary.
// Version wrapping will only occur if the negotiated version is non=empty and the application is a VersionWrapper.
func reconstructVersion(cbs []ClassicIBCModule, negotiatedVersions []string) (string, error) {
	version := negotiatedVersions[0] // base version
	for i := 1; i < len(cbs); i++ {  // iterate over the remaining callbacks
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
