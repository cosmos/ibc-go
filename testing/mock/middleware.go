package mock

import (
	"bytes"
	"context"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v9/modules/core/05-port/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

const (
	MockBlockUpgrade = "mockblockupgrade"
)

var _ porttypes.Middleware = (*BlockUpgradeMiddleware)(nil)

// BlockUpgradeMiddleware does not implement the UpgradeableModule interface
type BlockUpgradeMiddleware struct {
	appModule *AppModule
	IBCApp    *IBCApp // base application of an IBC middleware stack
}

// NewIBCModule creates a new IBCModule given the underlying mock IBC application and scopedKeeper.
func NewBlockUpgradeMiddleware(appModule *AppModule, app *IBCApp) BlockUpgradeMiddleware {
	appModule.ibcApps = append(appModule.ibcApps, app)
	return BlockUpgradeMiddleware{
		appModule: appModule,
		IBCApp:    app,
	}
}

// OnChanOpenInit implements the IBCModule interface.
func (im BlockUpgradeMiddleware) OnChanOpenInit(
	ctx context.Context, order channeltypes.Order, connectionHops []string, portID string,
	channelID string, chanCap *capabilitytypes.Capability, counterparty channeltypes.Counterparty, version string,
) (string, error) {
	if strings.TrimSpace(version) == "" {
		version = Version
	}

	if im.IBCApp.OnChanOpenInit != nil {
		return im.IBCApp.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, version)
	}

	if chanCap != nil {
		// Claim channel capability passed back by IBC module
		sdkCtx := sdk.UnwrapSDKContext(ctx) // TODO: remove after sdk.Context is removed from core IBC
		if err := im.IBCApp.ScopedKeeper.ClaimCapability(sdkCtx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
			return "", err
		}
	}

	return version, nil
}

// OnChanOpenTry implements the IBCModule interface.
func (im BlockUpgradeMiddleware) OnChanOpenTry(
	ctx context.Context, order channeltypes.Order, connectionHops []string, portID string,
	channelID string, chanCap *capabilitytypes.Capability, counterparty channeltypes.Counterparty, counterpartyVersion string,
) (version string, err error) {
	if im.IBCApp.OnChanOpenTry != nil {
		return im.IBCApp.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, counterpartyVersion)
	}

	if chanCap != nil {
		// Claim channel capability passed back by IBC module
		sdkCtx := sdk.UnwrapSDKContext(ctx) // TODO: remove after sdk.Context is removed from core IBC
		if err := im.IBCApp.ScopedKeeper.ClaimCapability(sdkCtx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
			return "", err
		}
	}

	return Version, nil
}

// OnChanOpenAck implements the IBCModule interface.
func (im BlockUpgradeMiddleware) OnChanOpenAck(ctx context.Context, portID string, channelID string, counterpartyChannelID string, counterpartyVersion string) error {
	if im.IBCApp.OnChanOpenAck != nil {
		return im.IBCApp.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
	}

	return nil
}

// OnChanOpenConfirm implements the IBCModule interface.
func (im BlockUpgradeMiddleware) OnChanOpenConfirm(ctx context.Context, portID, channelID string) error {
	if im.IBCApp.OnChanOpenConfirm != nil {
		return im.IBCApp.OnChanOpenConfirm(ctx, portID, channelID)
	}

	return nil
}

// OnChanCloseInit implements the IBCModule interface.
func (im BlockUpgradeMiddleware) OnChanCloseInit(ctx context.Context, portID, channelID string) error {
	if im.IBCApp.OnChanCloseInit != nil {
		return im.IBCApp.OnChanCloseInit(ctx, portID, channelID)
	}

	return nil
}

// OnChanCloseConfirm implements the IBCModule interface.
func (im BlockUpgradeMiddleware) OnChanCloseConfirm(ctx context.Context, portID, channelID string) error {
	if im.IBCApp.OnChanCloseConfirm != nil {
		return im.IBCApp.OnChanCloseConfirm(ctx, portID, channelID)
	}

	return nil
}

// OnRecvPacket implements the IBCModule interface.
func (im BlockUpgradeMiddleware) OnRecvPacket(ctx context.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) exported.Acknowledgement {
	if im.IBCApp.OnRecvPacket != nil {
		return im.IBCApp.OnRecvPacket(ctx, channelVersion, packet, relayer)
	}

	// set state by claiming capability to check if revert happens return
	capName := GetMockRecvCanaryCapabilityName(packet)
	sdkCtx := sdk.UnwrapSDKContext(ctx) // TODO: remove after sdk.Context is removed from core IBC
	if _, err := im.IBCApp.ScopedKeeper.NewCapability(sdkCtx, capName); err != nil {
		// application callback called twice on same packet sequence
		// must never occur
		panic(err)
	}

	if bytes.Equal(MockPacketData, packet.GetData()) {
		return MockAcknowledgement
	} else if bytes.Equal(MockAsyncPacketData, packet.GetData()) {
		return nil
	}

	return MockFailAcknowledgement
}

// OnAcknowledgementPacket implements the IBCModule interface.
func (im BlockUpgradeMiddleware) OnAcknowledgementPacket(ctx context.Context, channelVersion string, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
	if im.IBCApp.OnAcknowledgementPacket != nil {
		return im.IBCApp.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
	}

	capName := GetMockAckCanaryCapabilityName(packet)
	sdkCtx := sdk.UnwrapSDKContext(ctx) // TODO: remove after sdk.Context is removed from core IBC
	if _, err := im.IBCApp.ScopedKeeper.NewCapability(sdkCtx, capName); err != nil {
		// application callback called twice on same packet sequence
		// must never occur
		panic(err)
	}

	return nil
}

// OnTimeoutPacket implements the IBCModule interface.
func (im BlockUpgradeMiddleware) OnTimeoutPacket(ctx context.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	if im.IBCApp.OnTimeoutPacket != nil {
		return im.IBCApp.OnTimeoutPacket(ctx, channelVersion, packet, relayer)
	}

	capName := GetMockTimeoutCanaryCapabilityName(packet)
	sdkCtx := sdk.UnwrapSDKContext(ctx) // TODO: remove after sdk.Context is removed from core IBC
	if _, err := im.IBCApp.ScopedKeeper.NewCapability(sdkCtx, capName); err != nil {
		// application callback called twice on same packet sequence
		// must never occur
		panic(err)
	}

	return nil
}

// SendPacket implements the ICS4 Wrapper interface
func (BlockUpgradeMiddleware) SendPacket(
	ctx context.Context,
	chanCap *capabilitytypes.Capability,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (uint64, error) {
	return 0, nil
}

// WriteAcknowledgement implements the ICS4 Wrapper interface
func (BlockUpgradeMiddleware) WriteAcknowledgement(
	ctx context.Context,
	chanCap *capabilitytypes.Capability,
	packet exported.PacketI,
	ack exported.Acknowledgement,
) error {
	return nil
}

// GetAppVersion returns the application version of the underlying application
func (BlockUpgradeMiddleware) GetAppVersion(ctx context.Context, portID, channelID string) (string, bool) {
	return Version, true
}
