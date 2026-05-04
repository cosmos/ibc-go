package keeper_test

import (
	"fmt"
	"testing"
	"time"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/sandbox-ledger/app"
	"github.com/cosmos/sandbox-ledger/testutil"
	"github.com/cosmos/sandbox-ledger/x/tokenfactory/types"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log/v2"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/runtime"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"

	cmtabcitypes "github.com/cometbft/cometbft/abci/types"
)

const (
	// Addresses
	creatorAddrA   = "cosmos1nsh9vj9znccakn6xwlhlwx92acdt79yeqrkz4y"
	creatorAddrB   = "cosmos1uu635yk0hz3cvrypnryrggltrjq7975jrmeg97"
	invalidAddress = "invalid"

	// Token denoms
	testDenom        = "testtoken"
	nonExistentDenom = "doesnotexist"
	invalidDenom     = "not-so-alphanumeric"
)

func setupIntegrationApp(tb testing.TB) (*app.SandboxApp, sdk.Context) {
	tb.Helper()
	// Ensure bech32 prefixes match wf* test addresses
	testutil.SafeSetAddressPrefixes()

	db := dbm.NewMemDB()
	wfapp := app.NewApp(log.NewNopLogger(), db, nil, false, simtestutil.NewAppOptionsWithFlagHome(tb.TempDir()), baseapp.SetChainID("chain-id"))
	wfapp.SetInitChainer(func(ctx sdk.Context, _ *cmtabcitypes.RequestInitChain) (*cmtabcitypes.ResponseInitChain, error) {
		for _, mod := range wfapp.ModuleManager.OrderInitGenesis {
			if m, ok := wfapp.ModuleManager.Modules[mod].(module.HasGenesis); ok {
				m.InitGenesis(ctx, wfapp.AppCodec(), m.DefaultGenesis(wfapp.AppCodec()))
			}
		}

		return &cmtabcitypes.ResponseInitChain{}, nil
	})

	// set baseApp param store
	consensusParamsKeeper := consensusparamkeeper.NewKeeper(wfapp.AppCodec(), runtime.NewKVStoreService(wfapp.GetKey(consensusparamtypes.StoreKey)), authtypes.NewModuleAddress("gov").String(), runtime.EventService{})
	wfapp.SetParamStore(consensusParamsKeeper.ParamsStore)

	if err := wfapp.LoadLatestVersion(); err != nil {
		panic(fmt.Errorf("failed to load application version from store: %w", err))
	}

	_, err := wfapp.InitChain(&cmtabcitypes.RequestInitChain{ChainId: "chain-id", ConsensusParams: simtestutil.DefaultConsensusParams})
	require.NoError(tb, err)

	// use deliver-state context backed by BaseApp finalizeBlockState
	ctx := wfapp.NewContext(false).
		WithChainID("chain-id").
		WithBlockTime(time.Now()).
		WithBlockHeight(1)

	// Ensure module params exist (not strictly necessary if queries don’t touch them)
	require.NoError(tb, wfapp.TokenFactoryKeeper.SetParams(ctx, types.DefaultParams()))

	return wfapp, ctx
}

// TestKeeper_MintTo tests the MintTo function directly
func TestKeeper_MintTo(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	creator, err := sdk.AccAddressFromBech32(creatorAddrA)
	require.NoError(t, err)

	recipient, err := sdk.AccAddressFromBech32(creatorAddrB)
	require.NoError(t, err)

	// Create denom first
	err = wfapp.TokenFactoryKeeper.CreateDenom(ctx, creatorAddrA, testDenom)
	require.NoError(t, err)

	t.Run("successful mint", func(t *testing.T) {
		amount := math.NewInt(1000000)
		err := wfapp.TokenFactoryKeeper.MintTo(ctx, testDenom, amount, recipient)
		require.NoError(t, err)

		// Verify balance
		balance := wfapp.BankKeeper.GetBalance(ctx, recipient, testDenom)
		require.True(t, balance.Amount.Equal(amount))
	})

	t.Run("mint more tokens", func(t *testing.T) {
		amount := math.NewInt(500000)
		err := wfapp.TokenFactoryKeeper.MintTo(ctx, testDenom, amount, creator)
		require.NoError(t, err)

		// Verify balance
		balance := wfapp.BankKeeper.GetBalance(ctx, creator, testDenom)
		require.True(t, balance.Amount.Equal(amount))
	})

	t.Run("invalid denom format", func(t *testing.T) {
		amount := math.NewInt(1000)
		err := wfapp.TokenFactoryKeeper.MintTo(ctx, invalidDenom, amount, recipient)
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrInvalidDenom)
	})

	t.Run("empty denom", func(t *testing.T) {
		amount := math.NewInt(1000)
		err := wfapp.TokenFactoryKeeper.MintTo(ctx, "", amount, recipient)
		require.Error(t, err)
	})
}

// TestKeeper_BurnFrom tests the BurnFrom function directly
func TestKeeper_BurnFrom(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	holder, err := sdk.AccAddressFromBech32(creatorAddrA)
	require.NoError(t, err)

	// Create denom and mint some tokens first
	err = wfapp.TokenFactoryKeeper.CreateDenom(ctx, creatorAddrA, testDenom)
	require.NoError(t, err)

	mintAmount := math.NewInt(1000000)
	err = wfapp.TokenFactoryKeeper.MintTo(ctx, testDenom, mintAmount, holder)
	require.NoError(t, err)

	t.Run("successful burn", func(t *testing.T) {
		burnAmount := math.NewInt(400000)
		err := wfapp.TokenFactoryKeeper.BurnFrom(ctx, testDenom, burnAmount, holder)
		require.NoError(t, err)

		// Verify balance decreased
		balance := wfapp.BankKeeper.GetBalance(ctx, holder, testDenom)
		expected := mintAmount.Sub(burnAmount)
		require.True(t, balance.Amount.Equal(expected))
	})

	t.Run("burn remaining balance", func(t *testing.T) {
		balance := wfapp.BankKeeper.GetBalance(ctx, holder, testDenom)
		err := wfapp.TokenFactoryKeeper.BurnFrom(ctx, testDenom, balance.Amount, holder)
		require.NoError(t, err)

		// Verify balance is zero
		newBalance := wfapp.BankKeeper.GetBalance(ctx, holder, testDenom)
		require.True(t, newBalance.Amount.IsZero())
	})

	t.Run("insufficient balance", func(t *testing.T) {
		burnAmount := math.NewInt(1000)
		err := wfapp.TokenFactoryKeeper.BurnFrom(ctx, testDenom, burnAmount, holder)
		require.Error(t, err)
	})

	t.Run("invalid denom format", func(t *testing.T) {
		burnAmount := math.NewInt(1000)
		err := wfapp.TokenFactoryKeeper.BurnFrom(ctx, invalidDenom, burnAmount, holder)
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrInvalidDenom)
	})
}

// TestKeeper_HasDenom tests the HasDenom function
func TestKeeper_HasDenom(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	t.Run("denom does not exist", func(t *testing.T) {
		exists := wfapp.TokenFactoryKeeper.HasDenom(ctx, testDenom)
		require.False(t, exists)
	})

	t.Run("denom exists after creation", func(t *testing.T) {
		err := wfapp.TokenFactoryKeeper.CreateDenom(ctx, creatorAddrA, testDenom)
		require.NoError(t, err)

		exists := wfapp.TokenFactoryKeeper.HasDenom(ctx, testDenom)
		require.True(t, exists)
	})

	t.Run("invalid denom format returns false", func(t *testing.T) {
		exists := wfapp.TokenFactoryKeeper.HasDenom(ctx, invalidDenom)
		require.False(t, exists)
	})

	t.Run("empty denom returns false", func(t *testing.T) {
		exists := wfapp.TokenFactoryKeeper.HasDenom(ctx, "")
		require.False(t, exists)
	})

	t.Run("non-existent valid denom returns false", func(t *testing.T) {
		exists := wfapp.TokenFactoryKeeper.HasDenom(ctx, nonExistentDenom)
		require.False(t, exists)
	})
}
