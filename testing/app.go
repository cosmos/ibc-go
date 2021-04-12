package ibctesting

import (
	"github.com/cosmos/ibc-go/modules/core/keeper"
)

type TestingApp interface {
	IBCKeeper() keeper.Keeper
}
