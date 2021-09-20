package mock

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	channeltypes "github.com/cosmos/ibc-go/v2/modules/core/04-channel/types"
)

type MockMiddleware struct {
	OnChanOpenInit func(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID string,
		channelID string, chanCap *capabilitytypes.Capability, counterparty channeltypes.Counterparty, version string,
	) error
	OnChanOpenTry func(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID string,
		channelID string, chanCap *capabilitytypes.Capability, counterparty channeltypes.Counterparty, version, counterpartyVersion string,
	) error
	OnChanOpenAck     func(sdk.Context, string, string, string) error
	OnChanOpenConfirm func(sdk.Context, string, string) error
}
