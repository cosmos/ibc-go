package mock

import (
	"bytes"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v9/modules/core/05-port/types"
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
	ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID string,
	channelID string, counterparty channeltypes.Counterparty, version string,
) (string, error) {
	if strings.TrimSpace(version) == "" {
		version = Version
	}

	if im.IBCApp.OnChanOpenInit != nil {
		return im.IBCApp.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, counterparty, version)
	}

	return version, nil
}

// OnChanOpenTry implements the IBCModule interface.
func (im BlockUpgradeMiddleware) OnChanOpenTry(
	ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID string,
	channelID string, counterparty channeltypes.Counterparty, counterpartyVersion string,
) (version string, err error) {
	if im.IBCApp.OnChanOpenTry != nil {
		return im.IBCApp.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, counterparty, counterpartyVersion)
	}

	return Version, nil
}

// OnChanOpenAck implements the IBCModule interface.
func (im BlockUpgradeMiddleware) OnChanOpenAck(ctx sdk.Context, portID string, channelID string, counterpartyChannelID string, counterpartyVersion string) error {
	if im.IBCApp.OnChanOpenAck != nil {
		return im.IBCApp.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
	}

	return nil
}

// OnChanOpenConfirm implements the IBCModule interface.
func (im BlockUpgradeMiddleware) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	if im.IBCApp.OnChanOpenConfirm != nil {
		return im.IBCApp.OnChanOpenConfirm(ctx, portID, channelID)
	}

	return nil
}

// OnChanCloseInit implements the IBCModule interface.
func (im BlockUpgradeMiddleware) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	if im.IBCApp.OnChanCloseInit != nil {
		return im.IBCApp.OnChanCloseInit(ctx, portID, channelID)
	}

	return nil
}

// OnChanCloseConfirm implements the IBCModule interface.
func (im BlockUpgradeMiddleware) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	if im.IBCApp.OnChanCloseConfirm != nil {
		return im.IBCApp.OnChanCloseConfirm(ctx, portID, channelID)
	}

	return nil
}

// OnSendPacket implements the IBCModule interface.
func (im BlockUpgradeMiddleware) OnSendPacket(ctx sdk.Context, portID string, channelID string, sequence uint64, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, data []byte, signer sdk.AccAddress) error {
	if im.IBCApp.OnSendPacket != nil {
		return im.IBCApp.OnSendPacket(ctx, portID, channelID, sequence, data, signer)
	}

	return nil
}

// OnRecvPacket implements the IBCModule interface.
func (im BlockUpgradeMiddleware) OnRecvPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) exported.RecvPacketResult {
	if im.IBCApp.OnRecvPacket != nil {
		return im.IBCApp.OnRecvPacket(ctx, channelVersion, packet, relayer)
	}

	// set state by claiming capability to check if revert happens return
	capName := GetMockRecvCanaryCapabilityName(packet)
	if _, err := im.IBCApp.ScopedKeeper.NewCapability(ctx, capName); err != nil {
		// application callback called twice on same packet sequence
		// must never occur
		panic(err)
	}

	if bytes.Equal(MockPacketData, packet.GetData()) {
		return exported.RecvPacketResult{
			Status:          exported.Success,
			Acknowledgement: MockAcknowledgement.Acknowledgement(),
		}
	} else if bytes.Equal(MockAsyncPacketData, packet.GetData()) {
		return exported.RecvPacketResult{
			Status: exported.Async,
		}
	}

	return exported.RecvPacketResult{
		Status:          exported.Failure,
		Acknowledgement: MockFailAcknowledgement.Acknowledgement(),
	}
}

// OnAcknowledgementPacket implements the IBCModule interface.
func (im BlockUpgradeMiddleware) OnAcknowledgementPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
	if im.IBCApp.OnAcknowledgementPacket != nil {
		return im.IBCApp.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
	}

	capName := GetMockAckCanaryCapabilityName(packet)
	if _, err := im.IBCApp.ScopedKeeper.NewCapability(ctx, capName); err != nil {
		// application callback called twice on same packet sequence
		// must never occur
		panic(err)
	}

	return nil
}

// OnTimeoutPacket implements the IBCModule interface.
func (im BlockUpgradeMiddleware) OnTimeoutPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	if im.IBCApp.OnTimeoutPacket != nil {
		return im.IBCApp.OnTimeoutPacket(ctx, channelVersion, packet, relayer)
	}

	capName := GetMockTimeoutCanaryCapabilityName(packet)
	if _, err := im.IBCApp.ScopedKeeper.NewCapability(ctx, capName); err != nil {
		// application callback called twice on same packet sequence
		// must never occur
		panic(err)
	}

	return nil
}

// WriteAcknowledgement implements the ICS4 Wrapper interface
func (BlockUpgradeMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	packet exported.PacketI,
	ack []byte,
) error {
	return nil
}

// GetAppVersion returns the application version of the underlying application
func (BlockUpgradeMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return Version, true
}
