package types

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

var _ exported.ClientMessage = &ClientMessage{}

// ClientType defines that the client message is a Wasm client consensus algorithm
func (c ClientMessage) ClientType() string {
	return exported.Wasm
}

// ValidateBasic defines a basic validation for the wasm client message.
func (c ClientMessage) ValidateBasic() error {
	if len(c.Data) == 0 {
		return errorsmod.Wrap(ErrInvalidData, "data cannot be empty")
	}

	return nil
}
