package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// Hooks specifies methods to enable writing custom logic which executes
// after specific client lifecycle methods.
type Hooks interface {
	// OnClientCreated is executed upon client creation.
	OnClientCreated(ctx sdk.Context, clientID string) error
	// OnClientUpdated is executed when a client is updated.
	OnClientUpdated(ctx sdk.Context, clientID string, consensusHeights []exported.Height) error
	// OnClientUpgraded is executed when a client is upgraded.
	OnClientUpgraded(ctx sdk.Context, clientID string) error
}

var _ Hooks = &MultiHooks{}

func NewMultiHooks(hooks ...Hooks) MultiHooks {
	return MultiHooks{hooks: hooks}
}

type MultiHooks struct {
	hooks []Hooks
}

func (m MultiHooks) OnClientCreated(ctx sdk.Context, clientID string) error {
	for _, h := range m.hooks {
		if err := h.OnClientCreated(ctx, clientID); err != nil {
			return err
		}
	}
	return nil
}

func (m MultiHooks) OnClientUpdated(ctx sdk.Context, clientID string, consensusHeights []exported.Height) error {
	for _, h := range m.hooks {
		if err := h.OnClientUpdated(ctx, clientID, consensusHeights); err != nil {
			return err
		}
	}
	return nil
}

func (m MultiHooks) OnClientUpgraded(ctx sdk.Context, clientID string) error {
	for _, h := range m.hooks {
		if err := h.OnClientUpgraded(ctx, clientID); err != nil {
			return err
		}
	}
	return nil
}
