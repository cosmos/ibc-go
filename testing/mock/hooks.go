package mock

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

var _ clienttypes.Hooks = &NoopHooks{}

type NoopHooks struct{}

func (n NoopHooks) OnClientCreated(ctx sdk.Context, clientId string) error {
	return nil
}

func (n NoopHooks) OnClientUpdated(ctx sdk.Context, clientID string, consensusHeights []exported.Height) error {
	return nil
}

func (n NoopHooks) OnClientUpgraded(ctx sdk.Context, clientID string) error {
	return nil
}
