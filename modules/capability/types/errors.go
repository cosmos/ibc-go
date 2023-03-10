package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/capability module sentinel errors
var (
	// TODO: these codes have all been incremented as codes 2-8 have already been registered in the SDK and so panics
	// during var declaration.
	ErrInvalidCapabilityName    = sdkerrors.Register(ModuleName, 9, "capability name not valid")
	ErrNilCapability            = sdkerrors.Register(ModuleName, 10, "provided capability is nil")
	ErrCapabilityTaken          = sdkerrors.Register(ModuleName, 11, "capability name already taken")
	ErrOwnerClaimed             = sdkerrors.Register(ModuleName, 12, "given owner already claimed capability")
	ErrCapabilityNotOwned       = sdkerrors.Register(ModuleName, 13, "capability not owned by module")
	ErrCapabilityNotFound       = sdkerrors.Register(ModuleName, 14, "capability not found")
	ErrCapabilityOwnersNotFound = sdkerrors.Register(ModuleName, 15, "owners not found for capability")
)
