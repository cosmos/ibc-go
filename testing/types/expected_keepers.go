package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// IBCTestingStakingKeeper defines the expected staking keeper interface used in the
// IBC testing package
type IBCTestingStakingKeeper interface {
	GetValidators(ctx sdk.Context, maxRetrieve uint32) (validators []stakingtypes.Validator)
	Delegate(
		ctx sdk.Context,
		delAddr sdk.AccAddress,
		bondAmt math.Int,
		tokenSrc stakingtypes.BondStatus,
		validator stakingtypes.Validator,
		subtractAccount bool,
	) (newShares sdk.Dec, err error)
	GetHistoricalInfo(ctx sdk.Context, height int64) (stakingtypes.HistoricalInfo, bool)
}
