package wasm

import (
	"github.com/cosmos/ibc-go/modules/core/exported"
	v5 "github.com/cosmos/ibc-go/v5/modules/core/exported"
)

var (
	_ exported.Misbehaviour = &Misbehaviour{}
)

func (m Misbehaviour) ClientType() string {
	return v5.Wasm
}

func (m Misbehaviour) GetClientID() string {
	return m.ClientId
}

func (m Misbehaviour) ValidateBasic() error {
	return nil
}
