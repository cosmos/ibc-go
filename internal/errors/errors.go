package errors

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

var (
	// ErrTxDecode is returned if we cannot parse a transaction.
	ErrTxDecode = errorsmod.Register(exported.ModuleName, 2, "tx parse error")

	// ErrInvalidSequence is used the sequence number (nonce) is incorrect
	// for the signature.
	ErrInvalidSequence = errorsmod.Register(exported.ModuleName, 3, "invalid sequence")

	// ErrUnauthorized is used whenever a request without sufficient
	// authorization is handled.
	ErrUnauthorized = errorsmod.Register(exported.ModuleName, 4, "unauthorized")

	// ErrInsufficientFunds is used when the account cannot pay requested amount.
	ErrInsufficientFunds = errorsmod.Register(exported.ModuleName, 5, "insufficient funds")

	// ErrUnknownRequest is used when the request body.
	ErrUnknownRequest = errorsmod.Register(exported.ModuleName, 6, "unknown request")

	// ErrInvalidAddress is used when an address is found to be invalid.
	ErrInvalidAddress = errorsmod.Register(exported.ModuleName, 7, "invalid address")

	// ErrInvalidPubKey is used when a public key is considered to be invalid.
	ErrInvalidPubKey = errorsmod.Register(exported.ModuleName, 8, "invalid pubkey")

	// ErrUnknownAddress is used when an address is unknown.
	ErrUnknownAddress = errorsmod.Register(exported.ModuleName, 9, "unknown address")

	// ErrInvalidCoins is used when sdk.Coins are invalid.
	ErrInvalidCoins = errorsmod.Register(exported.ModuleName, 10, "invalid coins")

	// ErrOutOfGas is used when there is not enough gas.
	ErrOutOfGas = errorsmod.Register(exported.ModuleName, 11, "out of gas")

	// ErrMemoTooLarge is used when the memo field is too large.
	ErrMemoTooLarge = errorsmod.Register(exported.ModuleName, 12, "memo too large")

	// ErrInsufficientFee is used when an insufficient fee is provided.
	ErrInsufficientFee = errorsmod.Register(exported.ModuleName, 13, "insufficient fee")

	// ErrTooManySignatures is used when too many signatures are present.
	ErrTooManySignatures = errorsmod.Register(exported.ModuleName, 14, "maximum number of signatures exceeded")

	// ErrNoSignatures is used when no signatures are provided.
	ErrNoSignatures = errorsmod.Register(exported.ModuleName, 15, "no signatures supplied")

	// ErrJSONMarshal defines an ABCI typed JSON marshalling error.
	ErrJSONMarshal = errorsmod.Register(exported.ModuleName, 16, "failed to marshal JSON bytes")

	// ErrJSONUnmarshal defines an ABCI typed JSON unmarshalling error.
	ErrJSONUnmarshal = errorsmod.Register(exported.ModuleName, 17, "failed to unmarshal JSON bytes")

	// ErrInvalidRequest defines an ABCI typed error where the request contains
	// invalid data.
	ErrInvalidRequest = errorsmod.Register(exported.ModuleName, 18, "invalid request")

	// ErrTxInMempoolCache defines an ABCI typed error where a tx already exists
	// in the mempool.
	ErrTxInMempoolCache = errorsmod.Register(exported.ModuleName, 19, "tx already in mempool")

	// ErrMempoolIsFull defines an ABCI typed error where the mempool is full.
	ErrMempoolIsFull = errorsmod.Register(exported.ModuleName, 20, "mempool is full")

	// ErrTxTooLarge defines an ABCI typed error where tx is too large.
	ErrTxTooLarge = errorsmod.Register(exported.ModuleName, 21, "tx too large")

	// ErrKeyNotFound defines an error when the key doesn't exist
	ErrKeyNotFound = errorsmod.Register(exported.ModuleName, 22, "key not found")

	// ErrWrongPassword defines an error when the key password is invalid.
	ErrWrongPassword = errorsmod.Register(exported.ModuleName, 23, "invalid account password")

	// ErrorInvalidSigner defines an error when the tx intended signer does not match the given signer.
	ErrorInvalidSigner = errorsmod.Register(exported.ModuleName, 24, "tx intended signer does not match the given signer")

	// ErrorInvalidGasAdjustment defines an error for an invalid gas adjustment
	ErrorInvalidGasAdjustment = errorsmod.Register(exported.ModuleName, 25, "invalid gas adjustment")

	// ErrInvalidHeight defines an error for an invalid height
	ErrInvalidHeight = errorsmod.Register(exported.ModuleName, 26, "invalid height")

	// ErrInvalidVersion defines a general error for an invalid version
	ErrInvalidVersion = errorsmod.Register(exported.ModuleName, 27, "invalid version")

	// ErrInvalidChainID defines an error when the chain-id is invalid.
	ErrInvalidChainID = errorsmod.Register(exported.ModuleName, 28, "invalid chain-id")

	// ErrInvalidType defines an error an invalid type.
	ErrInvalidType = errorsmod.Register(exported.ModuleName, 29, "invalid type")

	// ErrTxTimeoutHeight defines an error for when a tx is rejected out due to an
	// explicitly set timeout height.
	ErrTxTimeoutHeight = errorsmod.Register(exported.ModuleName, 30, "tx timeout height")

	// ErrUnknownExtensionOptions defines an error for unknown extension options.
	ErrUnknownExtensionOptions = errorsmod.Register(exported.ModuleName, 31, "unknown extension options")

	// ErrWrongSequence defines an error where the account sequence defined in
	// the signer info doesn't match the account's actual sequence number.
	ErrWrongSequence = errorsmod.Register(exported.ModuleName, 32, "incorrect account sequence")

	// ErrPackAny defines an error when packing a protobuf message to Any fails.
	ErrPackAny = errorsmod.Register(exported.ModuleName, 33, "failed packing protobuf message to Any")

	// ErrUnpackAny defines an error when unpacking a protobuf message from Any fails.
	ErrUnpackAny = errorsmod.Register(exported.ModuleName, 34, "failed unpacking protobuf message from Any")

	// ErrLogic defines an internal logic error, e.g. an invariant or assertion
	// that is violated. It is a programmer error, not a user-facing error.
	ErrLogic = errorsmod.Register(exported.ModuleName, 35, "internal logic error")

	// ErrConflict defines a conflict error, e.g. when two goroutines try to access
	// the same resource and one of them fails.
	ErrConflict = errorsmod.Register(exported.ModuleName, 36, "conflict")

	// ErrNotSupported is returned when we call a branch of a code which is currently not
	// supported.
	ErrNotSupported = errorsmod.Register(exported.ModuleName, 37, "feature not supported")

	// ErrNotFound defines an error when requested entity doesn't exist in the state.
	ErrNotFound = errorsmod.Register(exported.ModuleName, 38, "not found")

	// ErrIO should be used to wrap internal errors caused by external operation.
	// Examples: not DB domain error, file writing etc...
	ErrIO = errorsmod.Register(exported.ModuleName, 39, "Internal IO error")

	// ErrAppConfig defines an error occurred if min-gas-prices field in BaseConfig is empty.
	ErrAppConfig = errorsmod.Register(exported.ModuleName, 40, "error in app.toml")

	// ErrInvalidGasLimit defines an error when an invalid GasWanted value is
	// supplied.
	ErrInvalidGasLimit = errorsmod.Register(exported.ModuleName, 41, "invalid gas limit")

	// ErrPanic should only be set when we recovering from a panic
	ErrPanic = errorsmod.ErrPanic
)
