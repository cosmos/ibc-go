package types

import (
	"context"

	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

type LiteClientModule interface {
	GetCounterparty(ctx context.Context, clientId string) string
	VerifyMembership(
		ctx context.Context,
		clientId string,
		height exported.Height,
		delayTimePeriod uint64,
		delayBlockPeriod uint64,
		proof []byte,
		path exported.Path,
		value []byte,
	) error
	VerifyNonMembership(
		ctx context.Context,
		clientId string,
		height exported.Height,
		delayTimePeriod uint64,
		delayBlockPeriod uint64,
		proof []byte,
		path exported.Path,
	) error
}
