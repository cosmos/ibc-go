package ibctesting

import (
	"fmt"
	"time"

	"github.com/stretchr/testify/require"

	tmtypes "github.com/cometbft/cometbft/types"

	wasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
)

// ConstructUpdateWasmClientHeader will construct a valid 08-wasm Header with a zero height
// to update the light client on the source chain.
func (chain *TestChain) ConstructUpdateWasmClientHeader(counterparty *TestChain, clientID string) (*wasmtypes.ClientMessage, clienttypes.Height, error) {
	return chain.ConstructUpdateWasmClientHeaderWithTrustedHeight(counterparty, clientID, clienttypes.ZeroHeight())
}

// ConstructUpdateWasmClientHeaderWithTrustedHeight will construct a valid 08-wasm Header
// to update the light client on the source chain.
func (chain *TestChain) ConstructUpdateWasmClientHeaderWithTrustedHeight(
	counterparty *TestChain,
	clientID string,
	trustedHeight clienttypes.Height,
) (*wasmtypes.ClientMessage, clienttypes.Height, error) {
	tmHeader, err := chain.ConstructUpdateTMClientHeaderWithTrustedHeight(counterparty, clientID, trustedHeight)
	if err != nil {
		return nil, clienttypes.ZeroHeight(), err
	}

	tmWasmHeaderData, err := chain.Codec.MarshalInterface(tmHeader)
	if err != nil {
		return nil, clienttypes.ZeroHeight(), err
	}

	height, ok := tmHeader.GetHeight().(clienttypes.Height)
	if !ok {
		return nil, clienttypes.ZeroHeight(), fmt.Errorf("error casting exported height to clienttypes height")
	}

	wasmHeader := wasmtypes.ClientMessage{
		Data: tmWasmHeaderData,
	}

	return &wasmHeader, height, nil
}

func (chain *TestChain) CreateWasmClientHeader(
	chainID string,
	blockHeight int64,
	trustedHeight clienttypes.Height,
	timestamp time.Time,
	tmValSet, nextVals, tmTrustedVals *tmtypes.ValidatorSet,
	signers map[string]tmtypes.PrivValidator,
) *wasmtypes.ClientMessage {
	tmHeader := chain.CreateTMClientHeader(chainID, blockHeight, trustedHeight, timestamp, tmValSet, nextVals, tmTrustedVals, signers)
	tmWasmHeaderData, err := chain.Codec.MarshalInterface(tmHeader)
	require.NoError(chain.TB, err)
	return &wasmtypes.ClientMessage{
		Data: tmWasmHeaderData,
	}
}
