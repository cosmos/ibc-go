package keeper_test

import (
	"testing"

	"github.com/cosmos/sandbox-ledger/app"
	"github.com/cosmos/sandbox-ledger/x/tokenfactory/keeper"
	"github.com/cosmos/sandbox-ledger/x/tokenfactory/types"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TestMsgServer_CreateDenom tests the CreateDenom message handler.
func TestMsgServer_CreateDenom(t *testing.T) {
	cases := []struct {
		name     string
		creator  string
		denom    string
		malleate func(wfapp *app.SandboxApp, ctx sdk.Context)
		err      error
	}{
		{
			name:     "success",
			creator:  creatorAddrA,
			denom:    testDenom,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {},
		},
		{
			name:    "failure: duplicate denom",
			creator: creatorAddrA,
			denom:   testDenom,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {
				err := wfapp.TokenFactoryKeeper.CreateDenom(ctx, creatorAddrA, testDenom)
				require.NoError(t, err)
			},
			err: types.ErrDenomExists,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wfapp, ctx := setupIntegrationApp(t)
			ms := keeper.NewMsgServer(wfapp.TokenFactoryKeeper)

			tc.malleate(wfapp, ctx)

			res, err := ms.CreateDenom(ctx, types.NewMsgCreateDenom(tc.creator, tc.denom))
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)

			denoms, err := wfapp.TokenFactoryKeeper.GetDenomsFromCreator(ctx, tc.creator)
			require.NoError(t, err)
			require.Contains(t, denoms, tc.denom)

			md, found := wfapp.BankKeeper.GetDenomMetaData(ctx, tc.denom)
			require.True(t, found)
			require.Equal(t, tc.denom, md.Base)
			require.Len(t, md.DenomUnits, 1)
			require.Equal(t, tc.denom, md.DenomUnits[0].Denom)
			require.EqualValues(t, 0, md.DenomUnits[0].Exponent)
		})
	}
}

// TestMsgServer_Mint tests the Mint message handler.
func TestMsgServer_Mint(t *testing.T) {
	cases := []struct {
		name   string
		from   string
		to     string
		denom  string
		amount int64
		err    error
	}{
		{
			name:   "success",
			from:   creatorAddrA,
			to:     creatorAddrB,
			denom:  testDenom,
			amount: 1_000_000,
		},
		{
			name:   "failure: unauthorized",
			from:   creatorAddrB,
			to:     creatorAddrB,
			denom:  testDenom,
			amount: 1,
			err:    types.ErrUnauthorized,
		},
		{
			name:   "failure: invalid recipient",
			from:   creatorAddrA,
			to:     "invalid",
			denom:  testDenom,
			amount: 1,
			err:    types.ErrInvalidAddress,
		},
		{
			name:   "failure: invalid denom",
			from:   creatorAddrA,
			to:     creatorAddrA,
			denom:  invalidDenom,
			amount: 1,
			err:    types.ErrInvalidDenom,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wfapp, ctx := setupIntegrationApp(t)
			ms := keeper.NewMsgServer(wfapp.TokenFactoryKeeper)

			err := wfapp.TokenFactoryKeeper.CreateDenom(ctx, creatorAddrA, tc.denom)
			require.NoError(t, err)

			amt := sdk.NewCoin(tc.denom, math.NewInt(tc.amount))
			_, err = ms.Mint(ctx, types.NewMsgMint(tc.from, tc.to, amt))

			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestMsgServer_Burn tests the Burn message handler.
func TestMsgServer_Burn(t *testing.T) {
	var (
		denom      string
		mintAmount int64
		burnAmount int64
		burnFrom   string
	)

	cases := []struct {
		name     string
		err      error
		malleate func()
	}{
		{
			name:     "success",
			malleate: func() {},
		},
		{
			name: "failure: unauthorized",
			malleate: func() {
				burnFrom = creatorAddrB
			},
			err: types.ErrUnauthorized,
		},
		{
			name: "failure: nonexistent denom",
			malleate: func() {
				denom = nonExistentDenom
			},
			err: types.ErrDenomNotFound,
		},
		{
			name: "failure: invalid sender",
			malleate: func() {
				burnFrom = "invalid"
			},
			err: types.ErrInvalidAddress,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wfapp, ctx := setupIntegrationApp(t)
			ms := keeper.NewMsgServer(wfapp.TokenFactoryKeeper)

			denom = testDenom
			mintAmount = 1_000_000
			burnAmount = 300_000
			burnFrom = creatorAddrA

			var err error
			err = wfapp.TokenFactoryKeeper.CreateDenom(ctx, burnFrom, denom)
			require.NoError(t, err)
			_, err = ms.Mint(ctx, types.NewMsgMint(burnFrom, burnFrom, sdk.NewCoin(denom, math.NewInt(mintAmount))))
			require.NoError(t, err)

			originalBurnFrom := burnFrom
			originalDenom := denom
			pre := wfapp.BankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(burnFrom), denom)
			require.Equal(t, math.NewInt(mintAmount), pre.Amount)

			tc.malleate()

			_, err = ms.Burn(ctx, types.NewMsgBurn(burnFrom, sdk.NewCoin(denom, math.NewInt(burnAmount))))

			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				post := wfapp.BankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(originalBurnFrom), originalDenom)
				require.Equal(t, pre.Amount.Int64(), post.Amount.Int64())
				return
			}
			require.NoError(t, err)
			post := wfapp.BankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(burnFrom), denom)
			require.Equal(t, pre.Amount.Sub(math.NewInt(burnAmount)), post.Amount)
		})
	}
}

// TestMsgServer_CreateDenom_EmitsEvent tests that CreateDenom emits the correct event
func TestMsgServer_CreateDenom_EmitsEvent(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServer(wfapp.TokenFactoryKeeper)

	_, err := ms.CreateDenom(ctx, types.NewMsgCreateDenom(creatorAddrA, testDenom))
	require.NoError(t, err)

	events := ctx.EventManager().Events()
	var foundEvent bool
	for _, event := range events {
		if event.Type == types.TypeEvtCreateDenom {
			foundEvent = true
			// Verify event attributes
			var hasAdmin, hasDenom bool
			for _, attr := range event.Attributes {
				if attr.Key == types.AttributeKeyAdmin && attr.Value == creatorAddrA {
					hasAdmin = true
				}
				if attr.Key == types.AttributeKeyDenom && attr.Value == testDenom {
					hasDenom = true
				}
			}
			require.True(t, hasAdmin, "event should have admin attribute")
			require.True(t, hasDenom, "event should have denom attribute")
			break
		}
	}
	require.True(t, foundEvent, "CreateDenom should emit tokenfactory_create_denom event")
}

// TestMsgServer_Mint_EmitsEvent tests that Mint emits the correct event
func TestMsgServer_Mint_EmitsEvent(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServer(wfapp.TokenFactoryKeeper)

	err := wfapp.TokenFactoryKeeper.CreateDenom(ctx, creatorAddrA, testDenom)
	require.NoError(t, err)

	// Clear events from CreateDenom
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	amt := sdk.NewCoin(testDenom, math.NewInt(1000000))
	_, err = ms.Mint(ctx, types.NewMsgMint(creatorAddrA, creatorAddrB, amt))
	require.NoError(t, err)

	events := ctx.EventManager().Events()
	var foundEvent bool
	for _, event := range events {
		if event.Type == types.TypeEvtMint {
			foundEvent = true
			// Verify event attributes
			var hasAdmin, hasMintTo, hasDenom, hasAmount bool
			for _, attr := range event.Attributes {
				if attr.Key == types.AttributeKeyAdmin && attr.Value == creatorAddrA {
					hasAdmin = true
				}
				if attr.Key == types.AttributeKeyMintTo && attr.Value == creatorAddrB {
					hasMintTo = true
				}
				if attr.Key == types.AttributeKeyDenom && attr.Value == testDenom {
					hasDenom = true
				}
				if attr.Key == types.AttributeKeyAmount {
					hasAmount = true
				}
			}
			require.True(t, hasAdmin, "event should have admin attribute")
			require.True(t, hasMintTo, "event should have mint_to attribute")
			require.True(t, hasDenom, "event should have denom attribute")
			require.True(t, hasAmount, "event should have amount attribute")
			break
		}
	}
	require.True(t, foundEvent, "Mint should emit tokenfactory_mint event")
}

// TestMsgServer_Burn_EmitsEvent tests that Burn emits the correct event
func TestMsgServer_Burn_EmitsEvent(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServer(wfapp.TokenFactoryKeeper)

	err := wfapp.TokenFactoryKeeper.CreateDenom(ctx, creatorAddrA, testDenom)
	require.NoError(t, err)

	amt := sdk.NewCoin(testDenom, math.NewInt(1000000))
	_, err = ms.Mint(ctx, types.NewMsgMint(creatorAddrA, creatorAddrA, amt))
	require.NoError(t, err)

	// Clear events from CreateDenom and Mint
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	burnAmt := sdk.NewCoin(testDenom, math.NewInt(500000))
	_, err = ms.Burn(ctx, types.NewMsgBurn(creatorAddrA, burnAmt))
	require.NoError(t, err)

	events := ctx.EventManager().Events()
	var foundEvent bool
	for _, event := range events {
		if event.Type == types.TypeEvtBurn {
			foundEvent = true
			// Verify event attributes
			var hasAdmin, hasDenom, hasAmount bool
			for _, attr := range event.Attributes {
				if attr.Key == types.AttributeKeyAdmin && attr.Value == creatorAddrA {
					hasAdmin = true
				}
				if attr.Key == types.AttributeKeyDenom && attr.Value == testDenom {
					hasDenom = true
				}
				if attr.Key == types.AttributeKeyAmount {
					hasAmount = true
				}
			}
			require.True(t, hasAdmin, "event should have admin attribute")
			require.True(t, hasDenom, "event should have denom attribute")
			require.True(t, hasAmount, "event should have amount attribute")
			break
		}
	}
	require.True(t, foundEvent, "Burn should emit tokenfactory_burn event")
}

// TestMsgServer_ChangeAdmin tests the ChangeAdmin message handler.
func TestMsgServer_ChangeAdmin(t *testing.T) {
	cases := []struct {
		name     string
		sender   string
		newAdmin string
		denom    string
		malleate func(wfapp *app.SandboxApp, ctx sdk.Context)
		err      error
	}{
		{
			name:     "success",
			sender:   creatorAddrA,
			newAdmin: creatorAddrB,
			denom:    testDenom,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {},
		},
		{
			name:     "failure: unauthorized",
			sender:   creatorAddrB,
			newAdmin: creatorAddrB,
			denom:    testDenom,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {},
			err:      types.ErrUnauthorized,
		},
		{
			name:     "failure: invalid new admin",
			sender:   creatorAddrA,
			newAdmin: "invalid",
			denom:    testDenom,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {},
			err:      types.ErrInvalidAddress,
		},
		{
			name:     "failure: denom not found",
			sender:   creatorAddrA,
			newAdmin: creatorAddrB,
			denom:    nonExistentDenom,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {},
			err:      types.ErrDenomNotFound,
		},
		{
			name:     "failure: admin already renounced",
			sender:   creatorAddrA,
			newAdmin: creatorAddrB,
			denom:    testDenom,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {
				err := wfapp.TokenFactoryKeeper.RenounceAdmin(ctx, testDenom, creatorAddrA)
				require.NoError(t, err)
			},
			err: types.ErrAdminRenounced,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wfapp, ctx := setupIntegrationApp(t)
			ms := keeper.NewMsgServer(wfapp.TokenFactoryKeeper)

			err := wfapp.TokenFactoryKeeper.CreateDenom(ctx, creatorAddrA, testDenom)
			require.NoError(t, err)

			tc.malleate(wfapp, ctx)

			_, err = ms.ChangeAdmin(ctx, types.NewMsgChangeAdmin(tc.sender, tc.denom, tc.newAdmin))
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)

			// Verify admin changed
			md, err := wfapp.TokenFactoryKeeper.GetAuthorityMetadata(ctx, tc.denom)
			require.NoError(t, err)
			require.Equal(t, tc.newAdmin, md.Admin)
		})
	}
}

// TestMsgServer_RenounceAdmin tests the RenounceAdmin message handler.
func TestMsgServer_RenounceAdmin(t *testing.T) {
	cases := []struct {
		name     string
		sender   string
		denom    string
		malleate func(wfapp *app.SandboxApp, ctx sdk.Context)
		err      error
	}{
		{
			name:     "success",
			sender:   creatorAddrA,
			denom:    testDenom,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {},
		},
		{
			name:     "failure: unauthorized",
			sender:   creatorAddrB,
			denom:    testDenom,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {},
			err:      types.ErrUnauthorized,
		},
		{
			name:     "failure: denom not found",
			sender:   creatorAddrA,
			denom:    nonExistentDenom,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {},
			err:      types.ErrDenomNotFound,
		},
		{
			name:   "failure: already renounced",
			sender: creatorAddrA,
			denom:  testDenom,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {
				err := wfapp.TokenFactoryKeeper.RenounceAdmin(ctx, testDenom, creatorAddrA)
				require.NoError(t, err)
			},
			err: types.ErrAdminRenounced,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wfapp, ctx := setupIntegrationApp(t)
			ms := keeper.NewMsgServer(wfapp.TokenFactoryKeeper)

			err := wfapp.TokenFactoryKeeper.CreateDenom(ctx, creatorAddrA, testDenom)
			require.NoError(t, err)

			tc.malleate(wfapp, ctx)

			_, err = ms.RenounceAdmin(ctx, types.NewMsgRenounceAdmin(tc.sender, tc.denom))
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)

			// Verify admin is empty
			md, err := wfapp.TokenFactoryKeeper.GetAuthorityMetadata(ctx, tc.denom)
			require.NoError(t, err)
			require.Empty(t, md.Admin)
		})
	}
}

// TestMsgServer_RenounceAdmin_BlocksMinting tests that renouncing admin blocks MsgMint
func TestMsgServer_RenounceAdmin_BlocksMinting(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServer(wfapp.TokenFactoryKeeper)

	err := wfapp.TokenFactoryKeeper.CreateDenom(ctx, creatorAddrA, testDenom)
	require.NoError(t, err)

	// Mint works before renouncing
	amt := sdk.NewCoin(testDenom, math.NewInt(1000))
	_, err = ms.Mint(ctx, types.NewMsgMint(creatorAddrA, creatorAddrA, amt))
	require.NoError(t, err)

	// Renounce admin
	_, err = ms.RenounceAdmin(ctx, types.NewMsgRenounceAdmin(creatorAddrA, testDenom))
	require.NoError(t, err)

	// Mint fails after renouncing
	_, err = ms.Mint(ctx, types.NewMsgMint(creatorAddrA, creatorAddrA, amt))
	require.ErrorIs(t, err, types.ErrAdminRenounced)
}

// TestMsgServer_RenounceAdmin_KeepsModuleMinting tests that renouncing admin doesn't block keeper MintTo
func TestMsgServer_RenounceAdmin_KeepsModuleMinting(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServer(wfapp.TokenFactoryKeeper)

	err := wfapp.TokenFactoryKeeper.CreateDenom(ctx, creatorAddrA, testDenom)
	require.NoError(t, err)

	// Renounce admin
	_, err = ms.RenounceAdmin(ctx, types.NewMsgRenounceAdmin(creatorAddrA, testDenom))
	require.NoError(t, err)

	// Keeper MintTo still works (for module usage like IFT)
	to := sdk.MustAccAddressFromBech32(creatorAddrB)
	err = wfapp.TokenFactoryKeeper.MintTo(ctx, testDenom, math.NewInt(1000), to)
	require.NoError(t, err)

	// Verify balance
	balance := wfapp.BankKeeper.GetBalance(ctx, to, testDenom)
	require.Equal(t, math.NewInt(1000), balance.Amount)
}

// TestMsgServer_ChangeAdmin_EmitsEvent tests that ChangeAdmin emits the correct event
func TestMsgServer_ChangeAdmin_EmitsEvent(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServer(wfapp.TokenFactoryKeeper)

	err := wfapp.TokenFactoryKeeper.CreateDenom(ctx, creatorAddrA, testDenom)
	require.NoError(t, err)

	ctx = ctx.WithEventManager(sdk.NewEventManager())

	_, err = ms.ChangeAdmin(ctx, types.NewMsgChangeAdmin(creatorAddrA, testDenom, creatorAddrB))
	require.NoError(t, err)

	events := ctx.EventManager().Events()
	var foundEvent bool
	for _, event := range events {
		if event.Type == types.TypeEvtChangeAdmin {
			foundEvent = true
			var hasAdmin, hasNewAdmin, hasDenom bool
			for _, attr := range event.Attributes {
				if attr.Key == types.AttributeKeyAdmin && attr.Value == creatorAddrA {
					hasAdmin = true
				}
				if attr.Key == types.AttributeKeyNewAdmin && attr.Value == creatorAddrB {
					hasNewAdmin = true
				}
				if attr.Key == types.AttributeKeyDenom && attr.Value == testDenom {
					hasDenom = true
				}
			}
			require.True(t, hasAdmin, "event should have admin attribute")
			require.True(t, hasNewAdmin, "event should have new_admin attribute")
			require.True(t, hasDenom, "event should have denom attribute")
			break
		}
	}
	require.True(t, foundEvent, "ChangeAdmin should emit tokenfactory_change_admin event")
}

// TestMsgServer_RenounceAdmin_EmitsEvent tests that RenounceAdmin emits the correct event
func TestMsgServer_RenounceAdmin_EmitsEvent(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServer(wfapp.TokenFactoryKeeper)

	err := wfapp.TokenFactoryKeeper.CreateDenom(ctx, creatorAddrA, testDenom)
	require.NoError(t, err)

	ctx = ctx.WithEventManager(sdk.NewEventManager())

	_, err = ms.RenounceAdmin(ctx, types.NewMsgRenounceAdmin(creatorAddrA, testDenom))
	require.NoError(t, err)

	events := ctx.EventManager().Events()
	var foundEvent bool
	for _, event := range events {
		if event.Type == types.TypeEvtRenounceAdmin {
			foundEvent = true
			var hasAdmin, hasDenom bool
			for _, attr := range event.Attributes {
				if attr.Key == types.AttributeKeyAdmin && attr.Value == creatorAddrA {
					hasAdmin = true
				}
				if attr.Key == types.AttributeKeyDenom && attr.Value == testDenom {
					hasDenom = true
				}
			}
			require.True(t, hasAdmin, "event should have admin attribute")
			require.True(t, hasDenom, "event should have denom attribute")
			break
		}
	}
	require.True(t, foundEvent, "RenounceAdmin should emit tokenfactory_renounce_admin event")
}
