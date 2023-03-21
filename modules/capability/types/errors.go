package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	// TODO: these codes have all been incremented as codes 2-8 have already been registered in the SDK and so panics
	// during var declaration.
	ErrInvalidCapabilityName    = errorsmod.Register(ModuleName, 2, "capability name not valid")
	ErrNilCapability            = errorsmod.Register(ModuleName, 3, "provided capability is nil")
	ErrCapabilityTaken          = errorsmod.Register(ModuleName, 4, "capability name already taken")
	ErrOwnerClaimed             = errorsmod.Register(ModuleName, 5, "given owner already claimed capability")
	ErrCapabilityNotOwned       = errorsmod.Register(ModuleName, 6, "capability not owned by module")
	ErrCapabilityNotFound       = errorsmod.Register(ModuleName, 7, "capability not found")
	ErrCapabilityOwnersNotFound = errorsmod.Register(ModuleName, 8, "owners not found for capability")
)
