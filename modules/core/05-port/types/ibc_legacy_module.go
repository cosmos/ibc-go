package types

import (
	"slices"
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

var (
	_ ClassicIBCModule        = (*LegacyIBCModule)(nil)
	_ AcknowledgementListener = (*LegacyIBCModule)(nil)
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

func (*LegacyIBCModule) Name() string {
	return "ibc-legacy-module"
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
	for i, cb := range im.reversedCallbacks() {
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
		if wrapper, ok := cb.(VersionWrapper); ok && strings.TrimSpace(version) != "" {
			appVersion, underlyingAppVersion, err := wrapper.UnwrapVersionUnsafe(version)
			if err != nil {
				// middleware disabled
				negotiatedVersions[i] = ""
				continue
			}
			cbVersion, version = appVersion, underlyingAppVersion
		}

		negotiatedVersion, err := cb.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, counterparty, cbVersion)
		if err != nil {
			return "", errorsmod.Wrapf(err, "channel open init callback failed for port ID: %s, channel ID: %s", portID, channelID)
		}
		negotiatedVersions[i] = negotiatedVersion
	}

	return im.reconstructVersion(negotiatedVersions)
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
	for i, cb := range im.reversedCallbacks() {
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
		if wrapper, ok := cb.(VersionWrapper); ok && strings.TrimSpace(counterpartyVersion) != "" {
			appVersion, underlyingAppVersion, err := wrapper.UnwrapVersionUnsafe(counterpartyVersion)
			if err != nil {
				// middleware disabled
				negotiatedVersions[i] = ""
				continue
			}
			cbVersion, counterpartyVersion = appVersion, underlyingAppVersion
		}

		negotiatedVersion, err := cb.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, counterparty, cbVersion)
		if err != nil {
			return "", errorsmod.Wrapf(err, "channel open try callback failed for port ID: %s, channel ID: %s", portID, channelID)
		}
		negotiatedVersions[i] = negotiatedVersion
	}

	return im.reconstructVersion(negotiatedVersions)
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
	for _, cb := range im.reversedCallbacks() {
		cbVersion := counterpartyVersion

		// To maintain backwards compatibility, we must handle counterparty version negotiation.
		// This means the version may have changed, and applications must be allowed to be disabled.
		// Applications should be disabled when receiving an empty counterparty version. Callbacks
		// for all applications must occur to allow disabling.
		if wrapper, ok := cb.(VersionWrapper); ok {
			appVersion, underlyingAppVersion, err := wrapper.UnwrapVersionUnsafe(counterpartyVersion)
			if err != nil {
				cbVersion = "" // disable application
			} else {
				cbVersion, counterpartyVersion = appVersion, underlyingAppVersion
			}
		}

		err := cb.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, cbVersion)
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
	for _, cb := range im.reversedCallbacks() {
		err := cb.OnChanOpenConfirm(ctx, portID, channelID)
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
	for _, cb := range im.reversedCallbacks() {
		if err := cb.OnChanCloseInit(ctx, portID, channelID); err != nil {
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
	for _, cb := range im.reversedCallbacks() {
		if err := cb.OnChanCloseConfirm(ctx, portID, channelID); err != nil {
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
) channeltypes.RecvPacketResult {
	// Example (sync):
	// ResultList {
	// 		fee: { Status: Success, Acknowledgement: feeAck.Bytes() }
	//		transfer: { Status: Success / Failure, Acknowledgement: transferAck.Bytes() }
	// }
	//
	// Example (async):
	// ResultList {
	// 		fee: { Status: Success, Acknowledgement: feeAck.Bytes() }
	//		transfer: { Status: Async, Acknowledgement: nil }
	// }
	//
	// 1. Loop over result list.
	// 2. Check status of each result.
	// 3. If contains async recv result, then write results list to new state key with packet ID.
	// 4. Return async result to ibc core
	//
	// Current assumption:
	// if res.Status == Async, then res.Acknowledgement == nil
	// TODO: add validate func

	// TODO: Remove this implementation and fix tests
	results := im.OnRecvPacketLegacy(ctx, channelVersion, packet, relayer)
	res := im.WrapRecvResults(ctx, packet, results)
	im.OnWriteAcknowledgement(ctx, packet, res)
	return res
}

func (im *LegacyIBCModule) OnRecvPacketLegacy(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) channeltypes.AcknowledgementResults {
	var results []channeltypes.AcknowledgementResult
	for _, cb := range im.reversedCallbacks() {
		cbVersion := channelVersion

		if wrapper, ok := cb.(VersionWrapper); ok {
			cbVersion, channelVersion = wrapper.UnwrapVersionSafe(ctx, packet.DestinationPort, packet.DestinationChannel, cbVersion)
		}

		res := cb.OnRecvPacket(ctx, cbVersion, packet, relayer)
		results = append(results, channeltypes.AcknowledgementResult{RecvPacketResult: res, PortId: cb.Name()})
	}

	return channeltypes.AcknowledgementResults{
		AcknowledgementResults: results,
	}
}

func (im *LegacyIBCModule) OnWriteAcknowledgement(ctx sdk.Context, packet channeltypes.Packet, prevRes channeltypes.RecvPacketResult) {
	for _, cb := range im.cbs {
		if cb, ok := cb.(AcknowledgementListener); ok {
			cb.OnWriteAcknowledgement(ctx, packet, prevRes)
		}
	}
}

func (im *LegacyIBCModule) WrapRecvResults(ctx sdk.Context, packet channeltypes.Packet, results channeltypes.AcknowledgementResults) channeltypes.RecvPacketResult {
	cbs := im.reversedCallbacks()

	var recvResults []channeltypes.RecvPacketResult
	for _, r := range results.AcknowledgementResults {
		recvResults = append(recvResults, r.RecvPacketResult)
	}

	res := recvResults[len(recvResults)-1]
	for i := len(recvResults) - 2; i >= 0; i-- {
		if wrapper, ok := cbs[i].(AcknowledgementWrapper); ok {
			res = wrapper.WrapAcknowledgement(ctx, packet, res, recvResults[i])
		}
	}

	return res
}

// HandleAsyncRecvResults is called by core IBC when handling an async acknowledgement.
//
// It accepts the async acknowledgement (RecvPacketResult), application name (e.g. transfer) and list of already fulfilled results within the application stack.
// The list of ackResults are written to state in OnRecvPacket when handling an async acknowledgement packet result.
//
// 1. Find the index of the caller application within the list of ackResults.
// 2. Reverse the application callbacks (because the results are written in reverse order in OnRecvPacket).
// 3. Calculate the starting callback index (the index of the callback after the caller of WriteRecvPacketResult, see inline).
// 4. Execute the application callbacks and return the optionally wrapped acknowledgement.
func (im *LegacyIBCModule) HandleAsyncRecvResults(ctx sdk.Context, appName string, packet channeltypes.Packet, recvResult channeltypes.RecvPacketResult, ackResults channeltypes.AcknowledgementResults) ([]byte, error) {
	// find the start index for the caller application
	startIndex := slices.IndexFunc(ackResults.AcknowledgementResults, func(res types.AcknowledgementResult) bool {
		return res.PortId == appName
	})

	if startIndex == -1 {
		return nil, errorsmod.Wrapf(types.ErrInvalidAcknowledgement, "acknowledgement result for %s not found", appName)
	}

	callbacks := im.reversedCallbacks()

	//   - in a stack of [transfer, fee, appX, appY, appZ]
	//   - ack results are written in reverse due to the backwards iteration in the legacy module.
	//   - this becomes [appZ, appY, appX, fee, transfer]
	//   - if transfer is calling into this function, the start index will be 4.
	//   - we want to start at fee, (index 3) and iterate through the remainder of the callbacks in reverse order.
	//   - we subtract 1 so we start at 3 (fee), and pipe the result into appX, appY and then appZ.
	startingCallbackIndex := len(callbacks) - startIndex - 1
	for i := startingCallbackIndex; i >= 0; i-- {
		if cb, ok := callbacks[i].(AcknowledgementListener); ok {
			cb.OnWriteAcknowledgement(ctx, packet, recvResult)
		}

		if cb, ok := callbacks[i].(AcknowledgementWrapper); ok {
			recvResult = cb.WrapAcknowledgement(ctx, packet, recvResult, ackResults.AcknowledgementResults[i].RecvPacketResult)
		}
	}

	return recvResult.Acknowledgement, nil
}

// OnAcknowledgementPacket implements the IBCModule interface
func (im *LegacyIBCModule) OnAcknowledgementPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	for _, cb := range im.reversedCallbacks() {
		var (
			cbVersion = channelVersion
			cbAck     = acknowledgement
		)

		if wrapper, ok := cb.(VersionWrapper); ok {
			cbVersion, channelVersion = wrapper.UnwrapVersionSafe(ctx, packet.SourcePort, packet.SourceChannel, cbVersion)
		}

		if wrapper, ok := cb.(AcknowledgementWrapper); ok {
			cbAck, acknowledgement = wrapper.UnwrapAcknowledgement(ctx, packet.SourcePort, packet.SourceChannel, cbAck)
		}

		err := cb.OnAcknowledgementPacket(ctx, cbVersion, packet, cbAck, relayer)
		if err != nil {
			return errorsmod.Wrap(err, "acknowledge packet callback failed")
		}
	}
	return nil
}

// OnTimeoutPacket implements the IBCModule interface
func (im *LegacyIBCModule) OnTimeoutPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	for _, cb := range im.reversedCallbacks() {
		cbVersion := channelVersion

		if wrapper, ok := cb.(VersionWrapper); ok {
			cbVersion, channelVersion = wrapper.UnwrapVersionSafe(ctx, packet.SourcePort, packet.SourceChannel, cbVersion)
		}

		if err := cb.OnTimeoutPacket(ctx, cbVersion, packet, relayer); err != nil {
			return errorsmod.Wrapf(err, "on timeout packet callback failed for packet with source Port ID: %s, source channel ID: %s", packet.SourcePort, packet.SourceChannel)
		}
	}
	return nil
}

// OnChanUpgradeInit implements the IBCModule interface
func (im *LegacyIBCModule) OnChanUpgradeInit(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, proposedVersion string) (string, error) {
	negotiatedVersions := make([]string, len(im.cbs))
	for i, cb := range im.reversedCallbacks() {
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
		if wrapper, ok := cb.(VersionWrapper); ok && strings.TrimSpace(proposedVersion) != "" {
			appVersion, underlyingAppVersion, err := wrapper.UnwrapVersionUnsafe(proposedVersion)
			if err != nil {
				// middleware disabled
				negotiatedVersions[i] = ""
				continue
			}
			cbVersion, proposedVersion = appVersion, underlyingAppVersion
		}

		// in order to maintain backwards compatibility, every callback in the stack must implement the UpgradableModule interface.
		upgradableModule, ok := cb.(UpgradableModule)
		if !ok {
			return "", errorsmod.Wrap(ErrInvalidRoute, "upgrade route not found to module in application callstack")
		}

		negotiatedVersion, err := upgradableModule.OnChanUpgradeInit(ctx, portID, channelID, proposedOrder, proposedConnectionHops, cbVersion)
		if err != nil {
			return "", errorsmod.Wrapf(err, "channel open init callback failed for port ID: %s, channel ID: %s", portID, channelID)
		}
		negotiatedVersions[i] = negotiatedVersion
	}

	return im.reconstructVersion(negotiatedVersions)
}

// OnChanUpgradeTry implements the IBCModule interface
func (im *LegacyIBCModule) OnChanUpgradeTry(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, counterpartyVersion string) (string, error) {
	negotiatedVersions := make([]string, len(im.cbs))

	for i, cb := range im.reversedCallbacks() {
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
		if wrapper, ok := cb.(VersionWrapper); ok && strings.TrimSpace(counterpartyVersion) != "" {
			appVersion, underlyingAppVersion, err := wrapper.UnwrapVersionUnsafe(counterpartyVersion)
			if err != nil {
				// middleware disabled
				negotiatedVersions[i] = ""
				continue
			}
			cbVersion, counterpartyVersion = appVersion, underlyingAppVersion
		}

		// in order to maintain backwards compatibility, every callback in the stack must implement the UpgradableModule interface.
		upgradableModule, ok := cb.(UpgradableModule)
		if !ok {
			return "", errorsmod.Wrap(ErrInvalidRoute, "upgrade route not found to module in application callstack")
		}

		negotiatedVersion, err := upgradableModule.OnChanUpgradeTry(ctx, portID, channelID, proposedOrder, proposedConnectionHops, cbVersion)
		if err != nil {
			return "", errorsmod.Wrapf(err, "channel open init callback failed for port ID: %s, channel ID: %s", portID, channelID)
		}
		negotiatedVersions[i] = negotiatedVersion
	}

	return im.reconstructVersion(negotiatedVersions)
}

// OnChanUpgradeAck implements the IBCModule interface
func (im *LegacyIBCModule) OnChanUpgradeAck(ctx sdk.Context, portID, channelID, counterpartyVersion string) error {
	for _, cb := range im.reversedCallbacks() {
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
		if wrapper, ok := cb.(VersionWrapper); ok && strings.TrimSpace(counterpartyVersion) != "" {
			appVersion, underlyingAppVersion, err := wrapper.UnwrapVersionUnsafe(counterpartyVersion)
			if err != nil {
				// middleware disabled
				continue
			}
			cbVersion, counterpartyVersion = appVersion, underlyingAppVersion
		}

		// in order to maintain backwards compatibility, every callback in the stack must implement the UpgradableModule interface.
		upgradableModule, ok := cb.(UpgradableModule)
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
	for _, cb := range im.reversedCallbacks() {
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
		if wrapper, ok := cb.(VersionWrapper); ok {
			appVersion, underlyingAppVersion, err := wrapper.UnwrapVersionUnsafe(proposedVersion)
			if err != nil {
				cbVersion = "" // disable application
			} else {
				cbVersion, proposedVersion = appVersion, underlyingAppVersion
			}
		}

		// in order to maintain backwards compatibility, every callback in the stack must implement the UpgradableModule interface.
		upgradableModule, ok := cb.(UpgradableModule)
		if !ok {
			panic(errorsmod.Wrap(ErrInvalidRoute, "upgrade route not found to module in application callstack"))
		}

		upgradableModule.OnChanUpgradeOpen(ctx, portID, channelID, proposedOrder, proposedConnectionHops, cbVersion)
	}
}

// UnmarshalPacketData attempts to unmarshal the provided packet data bytes
// into a FungibleTokenPacketData. This function implements the optional
// PacketDataUnmarshaler interface required for ADR 008 support.
func (*LegacyIBCModule) UnmarshalPacketData(ctx sdk.Context, portID, channelID string, bz []byte) (interface{}, error) {
	return nil, nil
}

// reversedCallbacks returns a copy of the callbacks in reverse order.
// the majority of handlers are called in reverse order, so this can be used
// in those cases to prevent needing to iterate backwards over the callbacks.
func (im *LegacyIBCModule) reversedCallbacks() []ClassicIBCModule {
	cbs := slices.Clone(im.cbs)
	slices.Reverse(cbs)
	return cbs
}

// reconstructVersion will generate the channel version by applying any version wrapping as necessary.
// Version wrapping will only occur if the negotiated version is non=empty and the application is a VersionWrapper.
func (im *LegacyIBCModule) reconstructVersion(negotiatedVersions []string) (string, error) {
	// the negotiated versions are expected to be in reverse order, as callbacks are executed in reverse order.
	// in order to ensure that the indices match im.cbs, they must be reversed.
	// the slice is cloned to prevent modifying the input argument.
	negotiatedVersions = slices.Clone(negotiatedVersions)
	slices.Reverse(negotiatedVersions)

	version := negotiatedVersions[0] // base version
	for i := 1; i < len(im.cbs); i++ {
		if strings.TrimSpace(negotiatedVersions[i]) != "" {
			wrapper, ok := im.cbs[i].(VersionWrapper)
			if !ok {
				return "", ibcerrors.ErrInvalidVersion
			}
			version = wrapper.WrapVersion(negotiatedVersions[i], version)
		}
	}
	return version, nil
}
