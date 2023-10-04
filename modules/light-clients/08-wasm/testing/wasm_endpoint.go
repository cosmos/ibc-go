package wasmtesting

import (
	"github.com/stretchr/testify/require"

	types "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

type WasmEndpoint struct {
	*ibctesting.Endpoint
}

var (
	CodeHash               = []byte("01234567012345670123456701234567")
	contractClientState    = []byte{1}
	contractConsensusState = []byte{2}
)

// CreateClient creates an wasm client on a mock cometbft chain.
// The client and consensus states are represented by byte slices
// and the starting height is 1.
func (endpoint *WasmEndpoint) CreateClient() (err error) {
	clientState := types.NewClientState(contractClientState, CodeHash, clienttypes.NewHeight(0, 1))
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
