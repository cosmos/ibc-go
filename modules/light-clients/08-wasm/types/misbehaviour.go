package types

import (
	exported "github.com/cosmos/ibc-go/v7/modules/core/exported"
)

var _ exported.ClientMessage = &Misbehaviour{}

func (m Misbehaviour) ClientType() string {
	return exported.Wasm
}

func (m Misbehaviour) GetClientID() string {
	return m.ClientId
}

func (m Misbehaviour) ValidateBasic() error {
	return nil
}
