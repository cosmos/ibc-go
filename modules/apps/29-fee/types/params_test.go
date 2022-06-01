package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
)

func TestValidateParams(t *testing.T) {
	require.NoError(t, types.DefaultParams().Validate())
	require.NoError(t, types.NewParams("").Validate())
}
