package ibctesting

import (
	"encoding/hex"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/cometbft/cometbft/abci/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/ibc-go/e2e/testsuite"
)

// ApplyValSetChanges takes in cmttypes.ValidatorSet and []abci.ValidatorUpdate and will return a new cmttypes.ValidatorSet which has the
// provided validator updates applied to the provided validator set.
func ApplyValSetChanges(tb testing.TB, valSet *cmttypes.ValidatorSet, valUpdates []abci.ValidatorUpdate) *cmttypes.ValidatorSet {
	tb.Helper()
	updates, err := cmttypes.PB2TM.ValidatorUpdates(valUpdates)
	require.NoError(tb, err)

	// must copy since validator set will mutate with UpdateWithChangeSet
	newVals := valSet.Copy()
	err = newVals.UpdateWithChangeSet(updates)
	require.NoError(tb, err)

	return newVals
}

// GenerateString generates a random string of the given length in bytes
func GenerateString(length uint) string {
	bytes := make([]byte, length)
	for i := range bytes {
		bytes[i] = charset[rand.Intn(len(charset))]
	}
	return string(bytes)
}

// UnmarshalMsgResponse parse out msg responses from a transaction result
func UnmarshalMsgResponse(cdc *codec.LegacyAmino, resp abci.ExecTxResult, msgs ...codec.ProtoMarshaler) error {
	// Convert the response data to sdk.TxResponse
	txResp := sdk.TxResponse{
		Data: hex.EncodeToString(resp.Data),
	}

	// Use UnmarshalMsgResponses from testsuite package
	return testsuite.UnmarshalMsgResponses(txResp, msgs...)
}