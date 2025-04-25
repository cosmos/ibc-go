package mock

import (
	"bytes"
	"reflect"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

var (
	_ porttypes.IBCModule             = (*IBCModule)(nil)
	_ porttypes.PacketDataUnmarshaler = (*IBCModule)(nil)
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

// OnRecvPacket implements the IBCModule interface.
func (im IBCModule) OnRecvPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) exported.Acknowledgement {
	if im.IBCApp.OnRecvPacket != nil {
		return im.IBCApp.OnRecvPacket(ctx, channelVersion, packet, relayer)
	}

	ctx.EventManager().EmitEvent(NewMockRecvPacketEvent())

	if bytes.Equal(MockPacketData, packet.GetData()) {
		return MockAcknowledgement
	} else if bytes.Equal(MockAsyncPacketData, packet.GetData()) {
		return nil
	}

	return MockFailAcknowledgement
}

// OnAcknowledgementPacket implements the IBCModule interface.
func (im IBCModule) OnAcknowledgementPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
	if im.IBCApp.OnAcknowledgementPacket != nil {
		return im.IBCApp.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
	}

	ctx.EventManager().EmitEvent(NewMockAckPacketEvent())

	return nil
}

// OnTimeoutPacket implements the IBCModule interface.
func (im IBCModule) OnTimeoutPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	if im.IBCApp.OnTimeoutPacket != nil {
		return im.IBCApp.OnTimeoutPacket(ctx, channelVersion, packet, relayer)
	}

	ctx.EventManager().EmitEvent(NewMockTimeoutPacketEvent())

	return nil
}

// UnmarshalPacketData returns the MockPacketData. This function implements the optional
// PacketDataUnmarshaler interface required for ADR 008 support.
func (IBCModule) UnmarshalPacketData(ctx sdk.Context, portID string, channelID string, bz []byte) (any, string, error) {
	if reflect.DeepEqual(bz, MockPacketData) {
		return MockPacketData, Version, nil
	}
	return nil, "", MockApplicationCallbackError
}
