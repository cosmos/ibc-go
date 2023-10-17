package testing

import (
	"crypto/sha256"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

// WasmEndpoint is a wrapper around the ibctesting pkg Endpoint struct.
// It will override any functions which require special handling for the wasm client.
type WasmEndpoint struct {
	*ibctesting.Endpoint
}

var (
	// Represents the code of the wasm contract used in the tests with a mock vm.
	Code                   = []byte("01234567012345670123456701234567")
	contractClientState    = []byte{1}
	contractConsensusState = []byte{2}
)

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
func (endpoint *WasmEndpoint) CreateClient() error {
	codeHash := sha256.Sum256(Code)
	clientState := types.NewClientState(contractClientState, codeHash[:], clienttypes.NewHeight(0, 1))
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
