package types

import (
	errorsmod "cosmossdk.io/errors"
)

// IFT sentinel errors
var (
	ErrInvalidSigner             = errorsmod.Register(ModuleName, 1, "invalid signer")
	ErrUnauthorized              = errorsmod.Register(ModuleName, 2, "unauthorized")
	ErrBridgeNotFound            = errorsmod.Register(ModuleName, 3, "bridge not found")
	ErrPendingTransferNotFound   = errorsmod.Register(ModuleName, 5, "pending transfer not found")
	ErrInvalidDenom              = errorsmod.Register(ModuleName, 6, "invalid denom")
	ErrInvalidClientID           = errorsmod.Register(ModuleName, 7, "invalid client id")
	ErrInvalidReceiver           = errorsmod.Register(ModuleName, 8, "invalid receiver")
	ErrInvalidAmount             = errorsmod.Register(ModuleName, 9, "invalid amount")
	ErrInvalidConstructorType    = errorsmod.Register(ModuleName, 10, "invalid constructor type")
	ErrConstructMintCallFailed   = errorsmod.Register(ModuleName, 11, "failed to construct mint call")
	ErrSendCallFailed            = errorsmod.Register(ModuleName, 12, "failed to send ICS27-GMP call")
	ErrMintFailed                = errorsmod.Register(ModuleName, 13, "failed to mint tokens")
	ErrBurnFailed                = errorsmod.Register(ModuleName, 14, "failed to burn tokens")
	ErrDenomNotFound             = errorsmod.Register(ModuleName, 15, "denom not found in token factory")
	ErrUnauthorizedSender        = errorsmod.Register(ModuleName, 16, "unauthorized sender")
	ErrUnexpectedSalt            = errorsmod.Register(ModuleName, 17, "unexpected salt in account identifier")
	ErrInvalidPacketData         = errorsmod.Register(ModuleName, 18, "invalid packet data")
	ErrCallbackValidationFailed  = errorsmod.Register(ModuleName, 19, "callback validation failed")
	ErrBridgeHasPendingTransfers = errorsmod.Register(ModuleName, 20, "bridge has pending transfers")
	ErrInvalidTimeout            = errorsmod.Register(ModuleName, 21, "invalid timeout")
	ErrInvalidEVMAddress         = errorsmod.Register(ModuleName, 22, "invalid EVM address")
	ErrZeroAddress               = errorsmod.Register(ModuleName, 23, "zero address not allowed")
	ErrABIPackFailed             = errorsmod.Register(ModuleName, 24, "failed to pack ABI arguments")
	ErrInvalidSolanaAddress      = errorsmod.Register(ModuleName, 25, "invalid Solana address")
	ErrNotSolanaConstructor      = errorsmod.Register(ModuleName, 26, "not a solana constructor")
)
