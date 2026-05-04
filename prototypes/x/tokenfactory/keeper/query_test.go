package keeper_test

import (
	"testing"

	"github.com/cosmos/ibc-go/prototypes/x/tokenfactory/keeper"
	"github.com/cosmos/ibc-go/prototypes/x/tokenfactory/types"
	"github.com/stretchr/testify/require"
)

func TestQuery_DenomAuthorityMetadata(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServer(wfapp.TokenFactoryKeeper)
	denom := testDenom

	_, err := ms.CreateDenom(ctx, types.NewMsgCreateDenom(creatorAddrA, denom))
	require.NoError(t, err)

	res, err := wfapp.TokenFactoryKeeper.DenomAuthorityMetadata(ctx, &types.QueryDenomAuthorityMetadataRequest{Denom: denom})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, creatorAddrA, res.AuthorityMetadata.Admin)

	res, err = wfapp.TokenFactoryKeeper.DenomAuthorityMetadata(ctx, &types.QueryDenomAuthorityMetadataRequest{Denom: "uwfdeposit"})
	require.Error(t, err)
	require.Nil(t, res)
}

func TestQuery_DenomsByCreator(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServer(wfapp.TokenFactoryKeeper)

	denomA1 := "denom_a1"
	denomA2 := "denom_a2"
	denomB1 := "denom_b1"

	_, err := ms.CreateDenom(ctx, types.NewMsgCreateDenom(creatorAddrA, denomA1))
	require.NoError(t, err)
	_, err = ms.CreateDenom(ctx, types.NewMsgCreateDenom(creatorAddrA, denomA2))
	require.NoError(t, err)
	_, err = ms.CreateDenom(ctx, types.NewMsgCreateDenom(creatorAddrB, denomB1))
	require.NoError(t, err)

	res, err := wfapp.TokenFactoryKeeper.DenomsByCreator(ctx, &types.QueryDenomsByCreatorRequest{Creator: creatorAddrA})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.Denoms, 2)
	require.Contains(t, res.Denoms, denomA1)
	require.Contains(t, res.Denoms, denomA2)

	res, err = wfapp.TokenFactoryKeeper.DenomsByCreator(ctx, &types.QueryDenomsByCreatorRequest{Creator: "wf1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq"})
	require.NoError(t, err)
	require.Empty(t, res.Denoms)
}

func TestQueryParams(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	params := types.DefaultParams()
	require.NoError(t, wfapp.TokenFactoryKeeper.SetParams(ctx, params))

	response, err := wfapp.TokenFactoryKeeper.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, params, response.Params)
}
