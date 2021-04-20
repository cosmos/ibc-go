package ibctesting

import (
	"github.com/cosmos/ibc-go/modules/core/keeper"
)

type TestingApp interface {
	GetIBCKeeper() keeper.Keeper
}
