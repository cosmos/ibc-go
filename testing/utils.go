package ibctesting

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	cmttypes "github.com/cometbft/cometbft/types"
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
