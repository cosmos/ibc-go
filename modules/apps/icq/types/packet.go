package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateBasic performs basic validation of the interchain query packet data.
func (iqpd InterchainQueryPacketData) ValidateBasic() error {
	return nil
}

// GetBytes returns the JSON marshalled interchain query packet data.
func (iqpd InterchainQueryPacketData) GetBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&iqpd))
}
