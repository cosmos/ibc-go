package types

import (
	abci "github.com/tendermint/tendermint/abci/types"
)

func NewValidatorSetChangePacketData(valUpdates []abci.ValidatorUpdate) ValidatorSetChangePacketData {
	return ValidatorSetChangePacketData{
		ValidatorUpdates: valUpdates,
	}
}

func (vsc ValidatorSetChangePacketData) GetBytes() []byte {
	valUpdateBytes, _ := vsc.Marshal()
	return valUpdateBytes
}
