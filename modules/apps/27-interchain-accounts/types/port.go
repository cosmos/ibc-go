package types

import (
	"fmt"
	"strings"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// NewControllerPortID creates and returns a new prefixed controller port identifier using the provided owner string and a connection sequence
func NewControllerPortID(owner, connectionSeq string) (string, error) {
	if strings.TrimSpace(owner) == "" {
		return "", sdkerrors.Wrap(ErrInvalidAccountAddress, "owner address cannot be empty")
	}

	// parse only the connection number from the connection sequence string
	seq := strings.Split(connectionSeq, "-")[1]

	return fmt.Sprint(PortPrefix, owner, "-", seq), nil
}
