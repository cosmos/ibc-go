package localhost

import (
	errorsmod "cosmossdk.io/errors"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

var _ exported.ClientState = (*ClientState)(nil)

func (cs *ClientState) ClientType() string {
	return exported.Localhost
}

func (cs *ClientState) Validate() error {
	if cs.LatestHeight.RevisionHeight == 0 {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidHeight, "local revision height cannot be zero")
	}
	return nil
}
