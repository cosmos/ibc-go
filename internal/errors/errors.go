package errors

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

const codespace = exported.ModuleName

var (
	// ErrInvalidSequence is used the sequence number (nonce) is incorrect
	// for the signature.
	ErrInvalidSequence = errorsmod.Register(codespace, 3, "invalid sequence")

	// ErrUnauthorized is used whenever a request without sufficient
	// authorization is handled.
	ErrUnauthorized = errorsmod.Register(codespace, 4, "unauthorized")

	// ErrInsufficientFunds is used when the account cannot pay requested amount.
	ErrInsufficientFunds = errorsmod.Register(codespace, 5, "insufficient funds")

	// ErrUnknownRequest is used when the request body.
	ErrUnknownRequest = errorsmod.Register(codespace, 6, "unknown request")

	// ErrInvalidAddress is used when an address is found to be invalid.
	ErrInvalidAddress = errorsmod.Register(codespace, 7, "invalid address")

	// ErrInvalidCoins is used when sdk.Coins are invalid.
	ErrInvalidCoins = errorsmod.Register(codespace, 10, "invalid coins")

	// ErrOutOfGas is used when there is not enough gas.
	ErrOutOfGas = errorsmod.Register(codespace, 11, "out of gas")

	// ErrInvalidRequest defines an ABCI typed error where the request contains
	// invalid data.
	ErrInvalidRequest = errorsmod.Register(codespace, 18, "invalid request")

	// ErrInvalidHeight defines an error for an invalid height
	ErrInvalidHeight = errorsmod.Register(codespace, 26, "invalid height")

	// ErrInvalidVersion defines a general error for an invalid version
	ErrInvalidVersion = errorsmod.Register(codespace, 27, "invalid version")

	// ErrInvalidChainID defines an error when the chain-id is invalid.
	ErrInvalidChainID = errorsmod.Register(codespace, 28, "invalid chain-id")

	// ErrInvalidType defines an error an invalid type.
	ErrInvalidType = errorsmod.Register(codespace, 29, "invalid type")

	// ErrPackAny defines an error when packing a protobuf message to Any fails.
	ErrPackAny = errorsmod.Register(codespace, 33, "failed packing protobuf message to Any")

	// ErrUnpackAny defines an error when unpacking a protobuf message from Any fails.
	ErrUnpackAny = errorsmod.Register(codespace, 34, "failed unpacking protobuf message from Any")

	// ErrLogic defines an internal logic error, e.g. an invariant or assertion
	// that is violated. It is a programmer error, not a user-facing error.
	ErrLogic = errorsmod.Register(codespace, 35, "internal logic error")

	// ErrNotFound defines an error when requested entity doesn't exist in the state.
	ErrNotFound = errorsmod.Register(codespace, 38, "not found")
)
