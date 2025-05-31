package types

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (r *RateLimit) UpdateFlow(direction PacketDirection, amount sdkmath.Int) error {
	switch direction {
	case PACKET_SEND:
		return r.Flow.AddOutflow(amount, *r.Quota)
	case PACKET_RECV:
		return r.Flow.AddInflow(amount, *r.Quota)
	default:
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid packet direction (%s)", direction.String())
	}
}
