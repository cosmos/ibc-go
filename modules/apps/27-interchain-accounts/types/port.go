package types

import (
	"fmt"
	"strings"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const MaximumOwnerLength = 2048 // maximum length of the owner in bytes (value chosen arbitrarily)

// NewControllerPortID creates and returns a new prefixed controller port identifier using the provided owner string
func NewControllerPortID(owner string) (string, error) {
	if strings.TrimSpace(owner) == "" {
		return "", sdkerrors.Wrap(ErrInvalidAccountAddress, "owner address cannot be empty")
	}

	if len(owner) > MaximumOwnerLength {
		return "", sdkerrors.Wrapf(ErrInvalidAccountAddress, "owner address must not exceed %d bytes", MaximumOwnerLength)
	}

	return fmt.Sprint(PortPrefix, owner), nil
}
