package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	abci "github.com/tendermint/tendermint/abci/types"
)

func NewValidatorSetChangePacketData(valUpdates []abci.ValidatorUpdate) ValidatorSetChangePacketData {
	return ValidatorSetChangePacketData{
		ValidatorUpdates: valUpdates,
	}
}

// ValidateBasic is used for validating the CCV packet data.
func (vsc ValidatorSetChangePacketData) ValidateBasic() error {
	if len(vsc.ValidatorUpdates) == 0 {
		return sdkerrors.Wrap(ErrInvalidPacketData, "validator updates cannot be empty")
	}
	return nil
}

func (vsc ValidatorSetChangePacketData) GetBytes() []byte {
	valUpdateBytes, _ := vsc.Marshal()
	return valUpdateBytes
}
