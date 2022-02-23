package ibctesting

import (
	"testing"

	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

func ApplyValSetChanges(t *testing.T, valSet *tmtypes.ValidatorSet, valUpdates []abci.ValidatorUpdate) *tmtypes.ValidatorSet {
	updates, err := tmtypes.PB2TM.ValidatorUpdates(valUpdates)
	require.NoError(t, err)

	newVals := valSet.Copy()
	err = newVals.UpdateWithChangeSet(updates)
	require.NoError(t, err)

	return newVals
}
