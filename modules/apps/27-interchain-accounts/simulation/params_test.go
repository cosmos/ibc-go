package simulation_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	controllertypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
	hosttypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/host/types"
	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/simulation"
)

func TestParamChanges(t *testing.T) {
	s := rand.NewSource(1)
	r := rand.New(s)

	expected := []struct {
		composedKey string
		key         string
		simValue    string
		subspace    string
	}{
		{fmt.Sprintf("%s/%s", controllertypes.SubModuleName, controllertypes.KeyControllerEnabled), string(controllertypes.KeyControllerEnabled), "false", controllertypes.SubModuleName},
		{fmt.Sprintf("%s/%s", hosttypes.SubModuleName, hosttypes.KeyHostEnabled), string(hosttypes.KeyHostEnabled), "true", hosttypes.SubModuleName},
	}

	paramChanges := simulation.ParamChanges(r)

	require.Len(t, paramChanges, 2)

	for i, p := range paramChanges {
		require.Equal(t, expected[i].composedKey, p.ComposedKey())
		require.Equal(t, expected[i].key, p.Key())
		require.Equal(t, expected[i].simValue, p.SimValue()(r), p.Key())
		require.Equal(t, expected[i].subspace, p.Subspace())
	}
}
