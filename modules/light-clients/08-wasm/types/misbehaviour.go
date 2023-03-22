package types

import (
	"fmt"

	exported "github.com/cosmos/ibc-go/v7/modules/core/exported"
)

var _ exported.ClientMessage = &Misbehaviour{}

func (m Misbehaviour) ClientType() string {
	return exported.Wasm
}

func (m Misbehaviour) ValidateBasic() error {
	if m.Data == nil || len(m.Data) == 0 {
		return fmt.Errorf("data cannot be empty")
	}
	return nil
}
