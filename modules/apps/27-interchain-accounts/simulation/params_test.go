package simulation_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	controllertypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/controller/types"
	hosttypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/host/types"
	"github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/simulation"
	"github.com/cosmos/ibc-go/v6/testing/simapp"
)

func TestParamChanges(t *testing.T) {
	app := simapp.Setup(false)

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

	paramChanges := simulation.ParamChanges(r, &app.ICAControllerKeeper, &app.ICAHostKeeper)
	require.Len(t, paramChanges, 2)

	for i, p := range paramChanges {
		require.Equal(t, expected[i].composedKey, p.ComposedKey())
		require.Equal(t, expected[i].key, p.Key())
		require.Equal(t, expected[i].simValue, p.SimValue()(r), p.Key())
		require.Equal(t, expected[i].subspace, p.Subspace())
	}

	paramChanges = simulation.ParamChanges(r, &app.ICAControllerKeeper, nil)
	require.Len(t, paramChanges, 1)

	// the second call to paramChanges causing the controller enabled to be changed to true
	expected[0].simValue = "true"

	for _, p := range paramChanges {
		require.Equal(t, expected[0].composedKey, p.ComposedKey())
		require.Equal(t, expected[0].key, p.Key())
		require.Equal(t, expected[0].simValue, p.SimValue()(r), p.Key())
		require.Equal(t, expected[0].subspace, p.Subspace())
	}

	paramChanges = simulation.ParamChanges(r, nil, &app.ICAHostKeeper)
	require.Len(t, paramChanges, 1)

	for _, p := range paramChanges {
		require.Equal(t, expected[1].composedKey, p.ComposedKey())
		require.Equal(t, expected[1].key, p.Key())
		require.Equal(t, expected[1].simValue, p.SimValue()(r), p.Key())
		require.Equal(t, expected[1].subspace, p.Subspace())
	}
}
