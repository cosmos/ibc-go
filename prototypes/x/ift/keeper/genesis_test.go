package keeper_test

import (
	"testing"

	"github.com/cosmos/ibc-go/prototypes/x/ift/types"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
)

func TestGenesis_DefaultGenesis(t *testing.T) {
	genesis := types.DefaultGenesis()

	require.NotNil(t, genesis)
	require.NotNil(t, genesis.Params)
	require.Empty(t, genesis.Params.Authority)
	require.Empty(t, genesis.Bridges)
	require.Empty(t, genesis.PendingTransfers)
}

func TestGenesis_Validate(t *testing.T) {
	cases := []struct {
		name    string
		genesis types.GenesisState
		err     error
	}{
		{
			name:    "valid default genesis",
			genesis: *types.DefaultGenesis(),
		},
		{
			name: "valid genesis with bridges",
			genesis: types.GenesisState{
				Params: types.Params{Authority: adminAddr},
				Bridges: []types.GenesisBridge{
					{
						Denom: testDenom,
						Bridge: types.IFTBridge{
							ClientId:               "attestations-0",
							CounterpartyIftAddress: remoteIFTAddrA,
							IftSendCallConstructor: types.ConstructorEVM,
						},
					},
				},
				PendingTransfers: []types.PendingTransfer{},
			},
		},
		{
			name: "valid genesis with pending transfers",
			genesis: types.GenesisState{
				Params:  types.Params{Authority: adminAddr},
				Bridges: []types.GenesisBridge{},
				PendingTransfers: []types.PendingTransfer{
					{
						Denom:    testDenom,
						ClientId: "attestations-0",
						Sequence: 1,
						Sender:   userAddrA,
						Amount:   math.NewInt(1000),
					},
				},
			},
		},
		{
			name: "invalid bridge - empty denom",
			genesis: types.GenesisState{
				Params: types.Params{Authority: adminAddr},
				Bridges: []types.GenesisBridge{
					{
						Denom: "",
						Bridge: types.IFTBridge{
							ClientId:               "attestations-0",
							CounterpartyIftAddress: remoteIFTAddrA,
							IftSendCallConstructor: types.ConstructorEVM,
						},
					},
				},
			},
			err: types.ErrInvalidDenom,
		},
		{
			name: "invalid bridge - empty client id",
			genesis: types.GenesisState{
				Params: types.Params{Authority: adminAddr},
				Bridges: []types.GenesisBridge{
					{
						Denom: testDenom,
						Bridge: types.IFTBridge{
							ClientId:               "",
							CounterpartyIftAddress: remoteIFTAddrA,
							IftSendCallConstructor: types.ConstructorEVM,
						},
					},
				},
			},
			err: types.ErrInvalidClientID,
		},
		{
			name: "invalid pending transfer - empty denom",
			genesis: types.GenesisState{
				Params:  types.Params{Authority: adminAddr},
				Bridges: []types.GenesisBridge{},
				PendingTransfers: []types.PendingTransfer{
					{
						Denom:    "",
						ClientId: "attestations-0",
						Sequence: 1,
						Sender:   userAddrA,
						Amount:   math.NewInt(1000),
					},
				},
			},
			err: types.ErrInvalidDenom,
		},
		{
			name: "invalid pending transfer - empty client id",
			genesis: types.GenesisState{
				Params:  types.Params{Authority: adminAddr},
				Bridges: []types.GenesisBridge{},
				PendingTransfers: []types.PendingTransfer{
					{
						Denom:    testDenom,
						ClientId: "",
						Sequence: 1,
						Sender:   userAddrA,
						Amount:   math.NewInt(1000),
					},
				},
			},
			err: types.ErrInvalidClientID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.genesis.Validate()
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGenesis_InitGenesis(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	clientID1 := createIBCClient(t, ctx, wfapp)
	clientID2 := createIBCClient(t, ctx, wfapp)

	genesis := types.GenesisState{
		Params: types.Params{Authority: adminAddr},
		Bridges: []types.GenesisBridge{
			{
				Denom: testDenom,
				Bridge: types.IFTBridge{
					ClientId:               clientID1,
					CounterpartyIftAddress: remoteIFTAddrA,
					IftSendCallConstructor: types.ConstructorEVM,
				},
			},
			{
				Denom: testDenom2,
				Bridge: types.IFTBridge{
					ClientId:               clientID2,
					CounterpartyIftAddress: remoteIFTAddrB,
					IftSendCallConstructor: types.ConstructorCosmos,
				},
			},
		},
		PendingTransfers: []types.PendingTransfer{
			{
				Denom:    testDenom,
				ClientId: clientID1,
				Sequence: 1,
				Sender:   userAddrA,
				Amount:   math.NewInt(1000),
			},
			{
				Denom:    testDenom,
				ClientId: clientID1,
				Sequence: 2,
				Sender:   userAddrB,
				Amount:   math.NewInt(2000),
			},
		},
	}

	// Initialize genesis
	wfapp.IFTKeeper.InitGenesis(ctx, genesis)

	// Verify params
	params, err := wfapp.IFTKeeper.ParamsStore.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, adminAddr, params.Authority)

	// Verify bridges
	bridge1, err := wfapp.IFTKeeper.IFTBridgeStore.Get(ctx, collections.Join(testDenom, clientID1))
	require.NoError(t, err)
	require.Equal(t, remoteIFTAddrA, bridge1.CounterpartyIftAddress)
	require.Equal(t, types.ConstructorEVM, bridge1.IftSendCallConstructor)

	bridge2, err := wfapp.IFTKeeper.IFTBridgeStore.Get(ctx, collections.Join(testDenom2, clientID2))
	require.NoError(t, err)
	require.Equal(t, remoteIFTAddrB, bridge2.CounterpartyIftAddress)
	require.Equal(t, types.ConstructorCosmos, bridge2.IftSendCallConstructor)

	// Verify pending transfers
	pending1, err := wfapp.IFTKeeper.PendingTransferStore.Get(ctx, collections.Join(clientID1, uint64(1)))
	require.NoError(t, err)
	require.Equal(t, userAddrA, pending1.Sender)
	require.True(t, math.NewInt(1000).Equal(pending1.Amount))

	pending2, err := wfapp.IFTKeeper.PendingTransferStore.Get(ctx, collections.Join(clientID1, uint64(2)))
	require.NoError(t, err)
	require.Equal(t, userAddrB, pending2.Sender)
	require.True(t, math.NewInt(2000).Equal(pending2.Amount))
}

func TestGenesis_ExportGenesis(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	clientID1 := createIBCClient(t, ctx, wfapp)
	clientID2 := createIBCClient(t, ctx, wfapp)

	// Set up state
	err := wfapp.IFTKeeper.ParamsStore.Set(ctx, types.Params{Authority: adminAddr})
	require.NoError(t, err)

	bridge1 := types.IFTBridge{
		ClientId:               clientID1,
		CounterpartyIftAddress: remoteIFTAddrA,
		IftSendCallConstructor: types.ConstructorEVM,
	}
	err = wfapp.IFTKeeper.IFTBridgeStore.Set(ctx, collections.Join(testDenom, clientID1), bridge1)
	require.NoError(t, err)

	bridge2 := types.IFTBridge{
		ClientId:               clientID2,
		CounterpartyIftAddress: remoteIFTAddrB,
		IftSendCallConstructor: types.ConstructorCosmos,
	}
	err = wfapp.IFTKeeper.IFTBridgeStore.Set(ctx, collections.Join(testDenom2, clientID2), bridge2)
	require.NoError(t, err)

	pending := types.PendingTransfer{
		Denom:    testDenom,
		ClientId: clientID1,
		Sequence: 42,
		Sender:   userAddrA,
		Amount:   math.NewInt(5000),
	}
	err = wfapp.IFTKeeper.SetPendingTransfer(ctx, clientID1, 42, pending)
	require.NoError(t, err)

	// Export genesis
	exportedGenesis := wfapp.IFTKeeper.ExportGenesis(ctx)

	// Verify exported state
	require.Equal(t, adminAddr, exportedGenesis.Params.Authority)
	require.Len(t, exportedGenesis.Bridges, 2)
	require.Len(t, exportedGenesis.PendingTransfers, 1)

	// Verify pending transfer
	require.Equal(t, testDenom, exportedGenesis.PendingTransfers[0].Denom)
	require.Equal(t, clientID1, exportedGenesis.PendingTransfers[0].ClientId)
	require.Equal(t, uint64(42), exportedGenesis.PendingTransfers[0].Sequence)
	require.Equal(t, userAddrA, exportedGenesis.PendingTransfers[0].Sender)
	require.True(t, math.NewInt(5000).Equal(exportedGenesis.PendingTransfers[0].Amount))
}

func TestGenesis_InitExportRoundTrip(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	clientID := createIBCClient(t, ctx, wfapp)

	originalGenesis := types.GenesisState{
		Params: types.Params{Authority: adminAddr},
		Bridges: []types.GenesisBridge{
			{
				Denom: testDenom,
				Bridge: types.IFTBridge{
					ClientId:               clientID,
					CounterpartyIftAddress: remoteIFTAddrA,
					IftSendCallConstructor: types.ConstructorEVM,
				},
			},
		},
		PendingTransfers: []types.PendingTransfer{
			{
				Denom:    testDenom,
				ClientId: clientID,
				Sequence: 1,
				Sender:   userAddrA,
				Amount:   math.NewInt(12345),
			},
		},
	}

	// Initialize with genesis
	wfapp.IFTKeeper.InitGenesis(ctx, originalGenesis)

	// Export genesis
	exportedGenesis := wfapp.IFTKeeper.ExportGenesis(ctx)

	// Verify round-trip
	require.Equal(t, originalGenesis.Params.Authority, exportedGenesis.Params.Authority)
	require.Len(t, exportedGenesis.Bridges, 1)
	require.Equal(t, originalGenesis.Bridges[0].Denom, exportedGenesis.Bridges[0].Denom)
	require.Equal(t, originalGenesis.Bridges[0].Bridge.ClientId, exportedGenesis.Bridges[0].Bridge.ClientId)
	require.Equal(t, originalGenesis.Bridges[0].Bridge.CounterpartyIftAddress, exportedGenesis.Bridges[0].Bridge.CounterpartyIftAddress)
	require.Equal(t, originalGenesis.Bridges[0].Bridge.IftSendCallConstructor, exportedGenesis.Bridges[0].Bridge.IftSendCallConstructor)

	require.Len(t, exportedGenesis.PendingTransfers, 1)
	require.Equal(t, originalGenesis.PendingTransfers[0].Denom, exportedGenesis.PendingTransfers[0].Denom)
	require.Equal(t, originalGenesis.PendingTransfers[0].ClientId, exportedGenesis.PendingTransfers[0].ClientId)
	require.Equal(t, originalGenesis.PendingTransfers[0].Sequence, exportedGenesis.PendingTransfers[0].Sequence)
	require.Equal(t, originalGenesis.PendingTransfers[0].Sender, exportedGenesis.PendingTransfers[0].Sender)
	require.True(t, originalGenesis.PendingTransfers[0].Amount.Equal(exportedGenesis.PendingTransfers[0].Amount))
}

func TestGenesis_EmptyGenesis(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	// Export genesis from fresh state (should match default genesis structure)
	exportedGenesis := wfapp.IFTKeeper.ExportGenesis(ctx)

	// Should have default params set during setupIntegrationApp
	require.NotEmpty(t, exportedGenesis.Params.Authority)
	require.Empty(t, exportedGenesis.Bridges)
	require.Empty(t, exportedGenesis.PendingTransfers)
}
