package cosmosevm

import (
	errorsmod "cosmossdk.io/errors"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

var _ exported.ClientState = (*ClientState)(nil)

// NewClientState creates a new ClientState instance
func NewClientState(tendermintClientID string) *ClientState {
	return &ClientState{
		TendermintClientId: tendermintClientID,
	}
}

// ClientType implements the exported.ClientState interface.
func (ClientState) ClientType() string {
	return exported.CosmosEvm
}

// Validate implements the exported.ClientState interface.
func (cs ClientState) Validate() error {
	clientType, _, err := clienttypes.ParseClientIdentifier(cs.TendermintClientId)
	if err != nil {
		return err
	}
	if clientType != exported.Tendermint {
		return errorsmod.Wrapf(ErrInvalidClientType, "expected 07-tendermint client type in TendermintClientId, got %s", clientType)
	}
	return nil
}
