package types

import (
	"fmt"
	"strings"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	// ControllerPortFormat is the expected port identifier format to which controller chains must conform
	// See (TODO: Link to spec when updated)
	ControllerPortFormat = "<app-version>.<controller-conn-seq>.<host-conn-seq>.<owner>"
)

// NewControllerPortID creates and returns a new controller port identifier in the expected format
func NewControllerPortID(owner string) (string, error) {
	if strings.TrimSpace(owner) == "" {
		return "", sdkerrors.Wrap(ErrInvalidAccountAddress, "owner address cannot be empty")
	}

	return fmt.Sprint(PortPrefix, Delimiter, owner), nil
}
