package client

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/ibc-go/v3/modules/core/02-client/keeper"
	"github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
)

// NewClientProposalHandler defines the 02-client proposal handler
func NewClientProposalHandler(k keeper.Keeper) govv1beta1.Handler {
	return func(ctx sdk.Context, content govv1beta1.Content) error {
		switch c := content.(type) {
		case *types.ClientUpdateProposal:
			return k.ClientUpdateProposal(ctx, c)
		case *types.UpgradeProposal:
			return k.HandleUpgradeProposal(ctx, c)

		default:
			return sdkerrors.Wrapf(sdkerrors.ErrPanic, "unrecognized ibc proposal content type: %T", c)
		}
	}
}
