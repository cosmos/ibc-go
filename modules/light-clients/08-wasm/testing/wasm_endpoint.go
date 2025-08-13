package testing

import (
	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

// WasmEndpoint is a wrapper around the ibctesting pkg Endpoint struct.
// It will override any functions which require special handling for the wasm client.
type WasmEndpoint struct {
	*ibctesting.Endpoint
}

// NewWasmEndpoint returns a wasm endpoint with the default ibctesting pkg
// Endpoint embedded.
func NewWasmEndpoint(chain *ibctesting.TestChain) *WasmEndpoint {
	return &WasmEndpoint{
		Endpoint: ibctesting.NewDefaultEndpoint(chain),
	}
}

// CreateClient creates an wasm client on a mock cometbft chain.
// The client and consensus states are represented by byte slices
// and the starting height is 1.
func (ep *WasmEndpoint) CreateClient() error {
	checksum, err := types.CreateChecksum(Code)
	require.NoError(ep.Chain.TB, err)

	wrappedClientStateBz := clienttypes.MustMarshalClientState(ep.Chain.App.AppCodec(), CreateMockTendermintClientState(clienttypes.NewHeight(1, 5)))
	wrappedClientConsensusStateBz := clienttypes.MustMarshalConsensusState(ep.Chain.App.AppCodec(), MockTendermintClientConsensusState)

	clientState := types.NewClientState(wrappedClientStateBz, checksum, clienttypes.NewHeight(0, 1))
	consensusState := types.NewConsensusState(wrappedClientConsensusStateBz)

	msg, err := clienttypes.NewMsgCreateClient(
		clientState, consensusState, ep.Chain.SenderAccount.GetAddress().String(),
	)
	require.NoError(ep.Chain.TB, err)

	res, err := ep.Chain.SendMsgs(msg)
	if err != nil {
		return err
	}

	ep.ClientID, err = ibctesting.ParseClientIDFromEvents(res.Events)
	require.NoError(ep.Chain.TB, err)

	return nil
}
