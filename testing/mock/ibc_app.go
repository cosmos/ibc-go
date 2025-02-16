package mock

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// IBCApp contains IBC application module callbacks as defined in 05-port.
type IBCApp struct {
	PortID string

	OnChanOpenInit func(
		ctx sdk.Context,
		order channeltypes.Order,
		connectionHops []string,
		portID string,
		channelID string,
		counterparty channeltypes.Counterparty,
		version string,
	) (string, error)

	OnChanOpenTry func(
		ctx sdk.Context,
		order channeltypes.Order,
		connectionHops []string,
		portID,
		channelID string,
		counterparty channeltypes.Counterparty,
		counterpartyVersion string,
	) (version string, err error)

	OnChanOpenAck func(
		ctx sdk.Context,
		portID,
		channelID string,
		counterpartyChannelID string,
		counterpartyVersion string,
	) error

	OnChanOpenConfirm func(
		ctx sdk.Context,
		portID,
		channelID string,
	) error

	OnChanCloseInit func(
		ctx sdk.Context,
		portID,
		channelID string,
	) error

	OnChanCloseConfirm func(
		ctx sdk.Context,
		portID,
		channelID string,
	) error

	// OnRecvPacket must return an acknowledgement that implements the Acknowledgement interface.
	// In the case of an asynchronous acknowledgement, nil should be returned.
	// If the acknowledgement returned is successful, the state changes on callback are written,
	// otherwise the application state changes are discarded. In either case the packet is received
	// and the acknowledgement is written (in synchronous cases).
	OnRecvPacket func(
		ctx sdk.Context,
		channelVersion string,
		packet channeltypes.Packet,
		relayer sdk.AccAddress,
	) exported.Acknowledgement

	OnAcknowledgementPacket func(
		ctx sdk.Context,
		channelVersion string,
		packet channeltypes.Packet,
		acknowledgement []byte,
		relayer sdk.AccAddress,
	) error

	OnTimeoutPacket func(
		ctx sdk.Context,
		channelVersion string,
		packet channeltypes.Packet,
		relayer sdk.AccAddress,
	) error

	OnChanUpgradeInit func(
		ctx sdk.Context,
		portID, channelID string,
		order channeltypes.Order,
		connectionHops []string,
		version string,
	) (string, error)

	OnChanUpgradeTry func(
		ctx sdk.Context,
		portID, channelID string,
		order channeltypes.Order,
		connectionHops []string,
		counterpartyVersion string,
	) (string, error)

	OnChanUpgradeAck func(
		ctx sdk.Context,
		portID,
		channelID,
		counterpartyVersion string,
	) error

	OnChanUpgradeOpen func(
		ctx sdk.Context,
		portID,
		channelID string,
		order channeltypes.Order,
		connectionHops []string,
		version string,
	)
}

// NewIBCApp returns a IBCApp. An empty PortID indicates the mock app doesn't bind/claim ports.
func NewIBCApp(portID string) *IBCApp {
	return &IBCApp{
		PortID: portID,
	}
}
