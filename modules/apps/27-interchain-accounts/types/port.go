package types

import (
	"strings"

	errorsmod "cosmossdk.io/errors"
)

// NewControllerPortID creates and returns a new prefixed controller port identifier using the provided owner string
func NewControllerPortID(owner string) (string, error) {
	if strings.TrimSpace(owner) == "" {
		return "", errorsmod.Wrap(ErrInvalidAccountAddress, "owner address cannot be empty")
	}

	ownerWithPrefix := ControllerPortPrefix + owner
	return ownerWithPrefix, nil
}
