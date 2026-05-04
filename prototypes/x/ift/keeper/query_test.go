package keeper_test

import (
	"testing"

	"github.com/cosmos/sandbox-ledger/x/ift/types"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
)

func TestQuery_IFTBridge(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	clientID := createIBCClient(t, ctx, wfapp)

	bridge := types.IFTBridge{
		ClientId:               clientID,
		CounterpartyIftAddress: remoteIFTAddrA,
		IftSendCallConstructor: types.ConstructorEVM,
	}
	err := wfapp.IFTKeeper.IFTBridgeStore.Set(ctx, collections.Join(testDenom, clientID), bridge)
	require.NoError(t, err)

	// Query bridge
	resp, err := wfapp.IFTKeeper.IFTBridge(ctx, &types.QueryIFTBridgeRequest{
		Denom:    testDenom,
		ClientId: clientID,
	})
	require.NoError(t, err)
	require.Equal(t, clientID, resp.Bridge.ClientId)
	require.Equal(t, remoteIFTAddrA, resp.Bridge.CounterpartyIftAddress)
	require.Equal(t, types.ConstructorEVM, resp.Bridge.IftSendCallConstructor)
}



func TestQuery_IFTBridges(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	clientID1 := createIBCClient(t, ctx, wfapp)
	clientID2 := createIBCClient(t, ctx, wfapp)

	bridge1 := types.IFTBridge{
		ClientId:               clientID1,
		CounterpartyIftAddress: remoteIFTAddrA,
		IftSendCallConstructor: types.ConstructorEVM,
	}
	err := wfapp.IFTKeeper.IFTBridgeStore.Set(ctx, collections.Join(testDenom, clientID1), bridge1)
	require.NoError(t, err)

	bridge2 := types.IFTBridge{
		ClientId:               clientID2,
		CounterpartyIftAddress: remoteIFTAddrB,
		IftSendCallConstructor: types.ConstructorCosmos,
	}
	err = wfapp.IFTKeeper.IFTBridgeStore.Set(ctx, collections.Join(testDenom2, clientID2), bridge2)
	require.NoError(t, err)

	// Query all bridges
	resp, err := wfapp.IFTKeeper.IFTBridges(ctx, &types.QueryIFTBridgesRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Bridges, 2)
}

func TestQuery_IFTBridges_Empty(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	// Query all bridges (should be empty)
	resp, err := wfapp.IFTKeeper.IFTBridges(ctx, &types.QueryIFTBridgesRequest{})
	require.NoError(t, err)
	require.Empty(t, resp.Bridges)
}

func TestQuery_IFTBridgesByDenom(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	clientID1 := createIBCClient(t, ctx, wfapp)
	clientID2 := createIBCClient(t, ctx, wfapp)
	clientID3 := createIBCClient(t, ctx, wfapp)

	// Register two bridges for testDenom
	bridge1 := types.IFTBridge{
		ClientId:               clientID1,
		CounterpartyIftAddress: remoteIFTAddrA,
		IftSendCallConstructor: types.ConstructorEVM,
	}
	err := wfapp.IFTKeeper.IFTBridgeStore.Set(ctx, collections.Join(testDenom, clientID1), bridge1)
	require.NoError(t, err)

	bridge2 := types.IFTBridge{
		ClientId:               clientID2,
		CounterpartyIftAddress: remoteIFTAddrB,
		IftSendCallConstructor: types.ConstructorCosmos,
	}
	err = wfapp.IFTKeeper.IFTBridgeStore.Set(ctx, collections.Join(testDenom, clientID2), bridge2)
	require.NoError(t, err)

	// Register one bridge for testDenom2
	bridge3 := types.IFTBridge{
		ClientId:               clientID3,
		CounterpartyIftAddress: remoteIFTAddrA,
		IftSendCallConstructor: types.ConstructorEVM,
	}
	err = wfapp.IFTKeeper.IFTBridgeStore.Set(ctx, collections.Join(testDenom2, clientID3), bridge3)
	require.NoError(t, err)

	// Query bridges for testDenom only
	resp, err := wfapp.IFTKeeper.IFTBridgesByDenom(ctx, &types.QueryIFTBridgesByDenomRequest{
		Denom: testDenom,
	})
	require.NoError(t, err)
	require.Len(t, resp.Bridges, 2)

	// Query bridges for testDenom2 only
	resp, err = wfapp.IFTKeeper.IFTBridgesByDenom(ctx, &types.QueryIFTBridgesByDenomRequest{
		Denom: testDenom2,
	})
	require.NoError(t, err)
	require.Len(t, resp.Bridges, 1)
	require.Equal(t, clientID3, resp.Bridges[0].ClientId)
}

func TestQuery_IFTBridgesByDenom_NotFound(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	// Query bridges for nonexistent denom (should return empty, not error)
	resp, err := wfapp.IFTKeeper.IFTBridgesByDenom(ctx, &types.QueryIFTBridgesByDenomRequest{
		Denom: "nonexistent",
	})
	require.NoError(t, err)
	require.Empty(t, resp.Bridges)
}

func TestQuery_PendingTransfer(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	clientID := createIBCClient(t, ctx, wfapp)
	sequence := uint64(42)

	pending := types.PendingTransfer{
		Denom:    testDenom,
		ClientId: clientID,
		Sequence: sequence,
		Sender:   userAddrA,
		Amount:   math.NewInt(1000000),
	}
	err := wfapp.IFTKeeper.SetPendingTransfer(ctx, clientID, sequence, pending)
	require.NoError(t, err)

	// Query pending transfer
	resp, err := wfapp.IFTKeeper.PendingTransfer(ctx, &types.QueryPendingTransferRequest{
		Denom:    testDenom,
		ClientId: clientID,
		Sequence: sequence,
	})
	require.NoError(t, err)
	require.Equal(t, testDenom, resp.PendingTransfer.Denom)
	require.Equal(t, clientID, resp.PendingTransfer.ClientId)
	require.Equal(t, sequence, resp.PendingTransfer.Sequence)
	require.Equal(t, userAddrA, resp.PendingTransfer.Sender)
	require.True(t, pending.Amount.Equal(resp.PendingTransfer.Amount))
}

func TestQuery_Params(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	// Query params (already set in setup)
	resp, err := wfapp.IFTKeeper.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Params.Authority)
}

func TestQuery_Params_UpdateAndQuery(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	// Update params
	newParams := types.Params{
		Authority: userAddrA,
	}
	err := wfapp.IFTKeeper.ParamsStore.Set(ctx, newParams)
	require.NoError(t, err)

	// Query updated params
	resp, err := wfapp.IFTKeeper.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, userAddrA, resp.Params.Authority)
}
