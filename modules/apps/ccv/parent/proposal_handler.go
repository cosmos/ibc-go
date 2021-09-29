package parent

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/ibc-go/modules/apps/ccv/parent/keeper"
	ccv "github.com/cosmos/ibc-go/modules/apps/ccv/types"
)

// NewCreateChildChainHandler defines the CCV parent proposal handler
func NewCreateChildChainHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *ccv.CreateChildChainProposal:
			return k.CreateChildChainProposal(ctx, c)
		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized ibc proposal content type: %T", c)
		}
	}
}
