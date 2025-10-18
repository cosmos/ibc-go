package types

import (
	errorsmod "cosmossdk.io/errors"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

var _ exported.ClientState = (*ClientState)(nil)

// NewClientState creates a new ClientState instance.
func NewClientState(data []byte, checksum []byte, height clienttypes.Height) *ClientState {
	return &ClientState{
		Data:         data,
		Checksum:     checksum,
		LatestHeight: height,
	}
}

// ClientType is Wasm light client.
func (ClientState) ClientType() string {
	return Wasm
}

// Validate performs a basic validation of the client state fields.
func (cs ClientState) Validate() error {
	if len(cs.Data) == 0 {
		return errorsmod.Wrap(ErrInvalidData, "data cannot be empty")
	}

	return ValidateWasmChecksum(cs.Checksum)
}
