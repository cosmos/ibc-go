package wasmtesting

import (
	"github.com/stretchr/testify/require"

	types "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

type WasmEndpoint struct {
	*ibctesting.Endpoint
}

var (
	contractClientState    = []byte{1}
	contractConsensusState = []byte{2}
)

// CreateClient creates an IBC client on the endpoint. It will update the
// clientID for the endpoint if the message is successfully executed.
// NOTE: a solo machine client will be created with an empty diversifier.
func (endpoint *WasmEndpoint) CreateClient() (err error) {
	clientState := types.NewClientState(contractClientState, endpoint.Chain.Coordinator.CodeHash, clienttypes.NewHeight(0, 1))
	consensusState := types.NewConsensusState(contractConsensusState, 0)

	msg, err := clienttypes.NewMsgCreateClient(
		clientState, consensusState, endpoint.Chain.SenderAccount.GetAddress().String(),
	)
	require.NoError(endpoint.Chain.TB, err)

	res, err := endpoint.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	endpoint.ClientID, err = ibctesting.ParseClientIDFromEvents(res.Events)
	require.NoError(endpoint.Chain.TB, err)

	return nil
}
