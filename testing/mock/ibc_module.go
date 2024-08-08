package mock

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v9/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

var (
	_ porttypes.ClassicIBCModule      = (*IBCModule)(nil)
	_ porttypes.PacketDataUnmarshaler = (*IBCModule)(nil)
	_ porttypes.UpgradableModule      = (*IBCModule)(nil)
)

// applicationCallbackError is a custom error type that will be unique for testing purposes.
type applicationCallbackError struct{}

func (applicationCallbackError) Error() string {
	return "mock application callback failed"
}

// IBCModule implements the ICS26 callbacks for testing/mock.
type IBCModule struct {
	appModule *AppModule
	IBCApp    *IBCApp // base application of an IBC middleware stack
}

// NewIBCModule creates a new IBCModule given the underlying mock IBC application and scopedKeeper.
func NewIBCModule(appModule *AppModule, app *IBCApp) IBCModule {
	appModule.ibcApps = append(appModule.ibcApps, app)
	return IBCModule{
		appModule: appModule,
		IBCApp:    app,
	}
}

// OnChanOpenInit implements the IBCModule interface.
func (im IBCModule) OnChanOpenInit(
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
func (im IBCModule) OnChanOpenTry(
	ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID string,
	channelID string, counterparty channeltypes.Counterparty, counterpartyVersion string,
) (version string, err error) {
	if im.IBCApp.OnChanOpenTry != nil {
		return im.IBCApp.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, counterparty, counterpartyVersion)
	}

	return Version, nil
}

// OnChanOpenAck implements the IBCModule interface.
func (im IBCModule) OnChanOpenAck(ctx sdk.Context, portID string, channelID string, counterpartyChannelID string, counterpartyVersion string) error {
	if im.IBCApp.OnChanOpenAck != nil {
		return im.IBCApp.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
	}

	return nil
}

// OnChanOpenConfirm implements the IBCModule interface.
func (im IBCModule) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	if im.IBCApp.OnChanOpenConfirm != nil {
		return im.IBCApp.OnChanOpenConfirm(ctx, portID, channelID)
	}

	return nil
}

// OnChanCloseInit implements the IBCModule interface.
func (im IBCModule) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	if im.IBCApp.OnChanCloseInit != nil {
		return im.IBCApp.OnChanCloseInit(ctx, portID, channelID)
	}

	return nil
}

// OnChanCloseConfirm implements the IBCModule interface.
func (im IBCModule) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	if im.IBCApp.OnChanCloseConfirm != nil {
		return im.IBCApp.OnChanCloseConfirm(ctx, portID, channelID)
	}

	return nil
}

// OnSendPacket implements the IBCModule interface.
func (im IBCModule) OnSendPacket(ctx sdk.Context, portID string, channelID string, sequence uint64, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, data []byte, signer sdk.AccAddress) error {
	if im.IBCApp.OnSendPacket != nil {
		return im.IBCApp.OnSendPacket(ctx, portID, channelID, sequence, data, signer)
	}

	return nil
}

// OnRecvPacket implements the IBCModule interface.
func (im IBCModule) OnRecvPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) exported.RecvPacketResult {
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

	ctx.EventManager().EmitEvent(NewMockRecvPacketEvent())

	if bytes.Equal(MockPacketData, packet.GetData()) {
		return exported.RecvPacketResult{
			Status:          exported.SUCCESS,
			Acknowledgement: MockAcknowledgement.Acknowledgement(),
		}
	} else if bytes.Equal(MockAsyncPacketData, packet.GetData()) {
		return exported.RecvPacketResult{
			Status: exported.ASYNC,
		}
	}

	return exported.RecvPacketResult{
		Status:          exported.FAILURE,
		Acknowledgement: MockFailAcknowledgement.Acknowledgement(),
	}
}

// OnAcknowledgementPacket implements the IBCModule interface.
func (im IBCModule) OnAcknowledgementPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
	if im.IBCApp.OnAcknowledgementPacket != nil {
		return im.IBCApp.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
	}

	capName := GetMockAckCanaryCapabilityName(packet)
	if _, err := im.IBCApp.ScopedKeeper.NewCapability(ctx, capName); err != nil {
		// application callback called twice on same packet sequence
		// must never occur
		panic(err)
	}

	ctx.EventManager().EmitEvent(NewMockAckPacketEvent())

	return nil
}

// OnTimeoutPacket implements the IBCModule interface.
func (im IBCModule) OnTimeoutPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	if im.IBCApp.OnTimeoutPacket != nil {
		return im.IBCApp.OnTimeoutPacket(ctx, channelVersion, packet, relayer)
	}

	capName := GetMockTimeoutCanaryCapabilityName(packet)
	if _, err := im.IBCApp.ScopedKeeper.NewCapability(ctx, capName); err != nil {
		// application callback called twice on same packet sequence
		// must never occur
		panic(err)
	}

	ctx.EventManager().EmitEvent(NewMockTimeoutPacketEvent())

	return nil
}

// OnChanUpgradeInit implements the IBCModule interface
func (im IBCModule) OnChanUpgradeInit(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, proposedVersion string) (string, error) {
	if im.IBCApp.OnChanUpgradeInit != nil {
		return im.IBCApp.OnChanUpgradeInit(ctx, portID, channelID, proposedOrder, proposedConnectionHops, proposedVersion)
	}

	return proposedVersion, nil
}

// OnChanUpgradeTry implements the IBCModule interface
func (im IBCModule) OnChanUpgradeTry(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, counterpartyVersion string) (string, error) {
	if im.IBCApp.OnChanUpgradeTry != nil {
		return im.IBCApp.OnChanUpgradeTry(ctx, portID, channelID, proposedOrder, proposedConnectionHops, counterpartyVersion)
	}

	return counterpartyVersion, nil
}

// OnChanUpgradeAck implements the IBCModule interface
func (im IBCModule) OnChanUpgradeAck(ctx sdk.Context, portID, channelID, counterpartyVersion string) error {
	if im.IBCApp.OnChanUpgradeAck != nil {
		return im.IBCApp.OnChanUpgradeAck(ctx, portID, channelID, counterpartyVersion)
	}

	return nil
}

// OnChanUpgradeOpen implements the IBCModule interface
func (im IBCModule) OnChanUpgradeOpen(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, proposedVersion string) {
	if im.IBCApp.OnChanUpgradeOpen != nil {
		im.IBCApp.OnChanUpgradeOpen(ctx, portID, channelID, proposedOrder, proposedConnectionHops, proposedVersion)
	}
}

// UnmarshalPacketData returns the MockPacketData. This function implements the optional
// PacketDataUnmarshaler interface required for ADR 008 support.
func (IBCModule) UnmarshalPacketData(ctx sdk.Context, portID string, channelID string, bz []byte) (interface{}, string, error) {
	if reflect.DeepEqual(bz, MockPacketData) {
		return MockPacketData, Version, nil
	}
	return nil, "", MockApplicationCallbackError
}

// GetMockRecvCanaryCapabilityName generates a capability name for testing OnRecvPacket functionality.
func GetMockRecvCanaryCapabilityName(packet channeltypes.Packet) string {
	return fmt.Sprintf("%s%s%s%s", MockRecvCanaryCapabilityName, packet.GetDestPort(), packet.GetDestChannel(), strconv.Itoa(int(packet.GetSequence())))
}

// GetMockAckCanaryCapabilityName generates a capability name for OnAcknowledgementPacket functionality.
func GetMockAckCanaryCapabilityName(packet channeltypes.Packet) string {
	return fmt.Sprintf("%s%s%s%s", MockAckCanaryCapabilityName, packet.GetSourcePort(), packet.GetSourceChannel(), strconv.Itoa(int(packet.GetSequence())))
}

// GetMockTimeoutCanaryCapabilityName generates a capability name for OnTimeoutacket functionality.
func GetMockTimeoutCanaryCapabilityName(packet channeltypes.Packet) string {
	return fmt.Sprintf("%s%s%s%s", MockTimeoutCanaryCapabilityName, packet.GetSourcePort(), packet.GetSourceChannel(), strconv.Itoa(int(packet.GetSequence())))
}
