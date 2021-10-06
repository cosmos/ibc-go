package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (packet InterchainAccountPacketData) GetBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&packet))
}
