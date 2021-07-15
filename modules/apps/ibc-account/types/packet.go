package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (packet IBCAccountPacketData) GetBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&packet))
}
