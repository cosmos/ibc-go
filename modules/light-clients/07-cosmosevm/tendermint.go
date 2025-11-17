package cosmosevm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
)

// getTendermintClientState retrieves the tendermint client state associated with the tmClientID
func (l LightClientModule) getTendermintClientState(ctx sdk.Context, tmClientID string) (*ibctm.ClientState, error) {
	tmClientStateI, found := l.clientKeeper.GetClientState(ctx, tmClientID)
	if !found {
		return nil, ErrTendermintClientNotFound.Wrapf("tendermint client with ID %s not found", tmClientID)
	}

	tmClientState, ok := tmClientStateI.(*ibctm.ClientState)
	if !ok {
		return nil, ErrInvalidTendermintClientState.Wrapf("client state for ID %s is not tendermint client state", tmClientID)
	}

	return tmClientState, nil
}
