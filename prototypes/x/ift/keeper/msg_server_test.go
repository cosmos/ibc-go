package keeper_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/cosmos/sandbox-ledger/app"
	"github.com/cosmos/sandbox-ledger/x/ift/keeper"
	"github.com/cosmos/sandbox-ledger/x/ift/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	gmptypes "github.com/cosmos/ibc-go/v11/modules/apps/27-gmp/types"
	channelv2types "github.com/cosmos/ibc-go/v11/modules/core/04-channel/v2/types"
)

// TestMsgServer_RegisterIFTBridge tests the RegisterIFTBridge message handler.
func TestMsgServer_RegisterIFTBridge(t *testing.T) {
	var (
		signer              string
		denom               string
		clientID            string
		counterpartyAddress string
	)

	cases := []struct {
		name     string
		err      error
		malleate func(wfapp *app.SandboxApp, ctx sdk.Context)
	}{
		{
			name:     "success with evm constructor",
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {},
		},
		{
			name: "success with cosmos constructor",
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {
				// Use different denom to avoid duplicate
				denom = testDenom2
				createTokenFactoryDenom(t, ctx, wfapp, adminAddr, denom)
			},
		},
		{
			name: "failure: unauthorized - not authority",
			err:  types.ErrUnauthorized,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {
				signer = userAddrA
			},
		},
		{
			name: "failure: denom not found in token factory",
			err:  types.ErrDenomNotFound,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {
				denom = "nonexistent"
			},
		},
		{
			name: "success: bridge update (override existing)",
			err:  nil,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {
				// Pre-register the bridge with different counterparty address
				bridge := types.IFTBridge{
					ClientId:               clientID,
					CounterpartyIftAddress: "0x1111111111111111111111111111111111111111",
					IftSendCallConstructor: types.ConstructorEVM,
				}
				err := wfapp.IFTKeeper.IFTBridgeStore.Set(ctx, collections.Join(denom, clientID), bridge)
				require.NoError(t, err)
			},
		},
		{
			name: "failure: invalid constructor type",
			err:  types.ErrInvalidConstructorType,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {
				// Will be set via message field
			},
		},
		{
			name: "failure: IBC client not found",
			err:  types.ErrInvalidClientID,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {
				clientID = "nonexistent-client"
			},
		},
		{
			name: "failure: empty counterparty address",
			err:  types.ErrInvalidReceiver,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {
				counterpartyAddress = ""
			},
		},
		{
			name: "failure: invalid EVM counterparty address format",
			err:  types.ErrInvalidReceiver,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {
				counterpartyAddress = "not-a-valid-evm-address"
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wfapp, ctx := setupIntegrationApp(t)
			ms := keeper.NewMsgServerImpl(wfapp.IFTKeeper)

			// Reset defaults
			signer = authtypes.NewModuleAddress(govtypes.ModuleName).String()
			denom = testDenom
			clientID = createIBCClient(t, ctx, wfapp)
			counterpartyAddress = remoteIFTAddrA

			// Create token factory denom
			createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)

			tc.malleate(wfapp, ctx)

			constructorType := types.ConstructorEVM
			if tc.name == "success with cosmos constructor" {
				constructorType = types.ConstructorCosmos
			}
			if tc.name == "failure: invalid constructor type" {
				constructorType = "invalid"
			}

			msg := &types.MsgRegisterIFTBridge{
				Signer:                 signer,
				Denom:                  denom,
				ClientId:               clientID,
				CounterpartyIftAddress: counterpartyAddress,
				IftSendCallConstructor: constructorType,
			}

			_, err := ms.RegisterIFTBridge(ctx, msg)

			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}

			require.NoError(t, err)

			// Verify bridge was stored
			bridge, err := wfapp.IFTKeeper.IFTBridgeStore.Get(ctx, collections.Join(denom, clientID))
			require.NoError(t, err)
			require.Equal(t, clientID, bridge.ClientId)
			require.Equal(t, counterpartyAddress, bridge.CounterpartyIftAddress)
			require.Equal(t, constructorType, bridge.IftSendCallConstructor)
		})
	}
}

// TestMsgServer_RemoveIFTBridge tests the RemoveIFTBridge message handler.
func TestMsgServer_RemoveIFTBridge(t *testing.T) {
	var (
		signer   string
		denom    string
		clientID string
	)

	cases := []struct {
		name     string
		err      error
		malleate func(wfapp *app.SandboxApp, ctx sdk.Context)
	}{
		{
			name:     "success",
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {},
		},
		{
			name: "failure: unauthorized",
			err:  types.ErrUnauthorized,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {
				signer = userAddrA
			},
		},
		{
			name: "failure: bridge not found",
			err:  types.ErrBridgeNotFound,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {
				// Remove the bridge before the test
				err := wfapp.IFTKeeper.IFTBridgeStore.Remove(ctx, collections.Join(denom, clientID))
				require.NoError(t, err)
			},
		},
		{
			name: "failure: has pending transfers",
			err:  types.ErrBridgeHasPendingTransfers,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {
				// Create a pending transfer for this bridge
				pending := types.PendingTransfer{
					Denom:    denom,
					ClientId: clientID,
					Sequence: 1,
					Sender:   userAddrA,
					Amount:   math.NewInt(1000000),
				}
				err := wfapp.IFTKeeper.SetPendingTransfer(ctx, clientID, 1, pending)
				require.NoError(t, err)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wfapp, ctx := setupIntegrationApp(t)
			ms := keeper.NewMsgServerImpl(wfapp.IFTKeeper)

			// Reset defaults
			signer = authtypes.NewModuleAddress(govtypes.ModuleName).String()
			denom = testDenom
			clientID = createIBCClient(t, ctx, wfapp)

			// Create token factory denom
			createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)

			// Pre-register bridge
			bridge := types.IFTBridge{
				ClientId:               clientID,
				CounterpartyIftAddress: remoteIFTAddrA,
				IftSendCallConstructor: types.ConstructorEVM,
			}
			err := wfapp.IFTKeeper.IFTBridgeStore.Set(ctx, collections.Join(denom, clientID), bridge)
			require.NoError(t, err)

			tc.malleate(wfapp, ctx)

			msg := &types.MsgRemoveIFTBridge{
				Signer:   signer,
				Denom:    denom,
				ClientId: clientID,
			}

			_, err = ms.RemoveIFTBridge(ctx, msg)

			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}

			require.NoError(t, err)

			// Verify bridge was removed
			exists, err := wfapp.IFTKeeper.IFTBridgeStore.Has(ctx, collections.Join(denom, clientID))
			require.NoError(t, err)
			require.False(t, exists)
		})
	}
}

// TestMsgServer_IFTTransfer tests the IFTTransfer message handler.
func TestMsgServer_IFTTransfer(t *testing.T) {
	var (
		signer   string
		denom    string
		clientID string
		receiver string
		amount   math.Int
	)

	cases := []struct {
		name     string
		err      error
		malleate func(wfapp *app.SandboxApp, ctx sdk.Context)
	}{
		// Note: success case requires full GMP infrastructure which is tested in e2e tests
		{
			name: "failure: invalid denom - empty",
			err:  types.ErrInvalidDenom,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {
				denom = ""
			},
		},
		{
			name: "failure: invalid client id - empty",
			err:  types.ErrInvalidClientID,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {
				clientID = ""
			},
		},
		{
			name: "failure: invalid receiver - empty",
			err:  types.ErrInvalidReceiver,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {
				receiver = ""
			},
		},
		{
			name: "failure: invalid amount - zero",
			err:  types.ErrInvalidAmount,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {
				amount = math.ZeroInt()
			},
		},
		{
			name: "failure: invalid sender address",
			err:  types.ErrInvalidSigner,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {
				signer = invalidAddr
			},
		},
		{
			name: "failure: bridge not found",
			err:  types.ErrBridgeNotFound,
			malleate: func(wfapp *app.SandboxApp, ctx sdk.Context) {
				// Remove bridge
				err := wfapp.IFTKeeper.IFTBridgeStore.Remove(ctx, collections.Join(denom, clientID))
				require.NoError(t, err)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wfapp, ctx := setupIntegrationApp(t)
			ms := keeper.NewMsgServerImpl(wfapp.IFTKeeper)

			// Reset defaults
			signer = userAddrA
			denom = testDenom
			receiver = remoteIFTAddrA
			amount = math.NewInt(1000000)

			// Setup
			createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)
			clientID = createIBCClient(t, ctx, wfapp)

			// Register bridge
			bridge := types.IFTBridge{
				ClientId:               clientID,
				CounterpartyIftAddress: remoteIFTAddrB,
				IftSendCallConstructor: types.ConstructorEVM,
			}
			err := wfapp.IFTKeeper.IFTBridgeStore.Set(ctx, collections.Join(denom, clientID), bridge)
			require.NoError(t, err)

			// Mint tokens to sender
			senderAddr := sdk.MustAccAddressFromBech32(userAddrA)
			mintTokens(t, ctx, wfapp, testDenom, math.NewInt(2000000), senderAddr)

			tc.malleate(wfapp, ctx)

			timeout := uint64(time.Now().Add(30 * time.Second).Unix())
			msg := &types.MsgIFTTransfer{
				Signer:           signer,
				Denom:            denom,
				ClientId:         clientID,
				Receiver:         receiver,
				Amount:           amount,
				TimeoutTimestamp: timeout,
			}

			resp, err := ms.IFTTransfer(ctx, msg)

			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Positive(t, resp.Sequence)

			// Verify pending transfer was stored
			pending, err := wfapp.IFTKeeper.PendingTransferStore.Get(ctx, collections.Join(clientID, resp.Sequence))
			require.NoError(t, err)
			require.Equal(t, denom, pending.Denom)
			require.Equal(t, clientID, pending.ClientId)
			require.Equal(t, resp.Sequence, pending.Sequence)
			require.Equal(t, signer, pending.Sender)
			require.True(t, amount.Equal(pending.Amount))
		})
	}
}

// TestMsgServer_UpdateParams tests the UpdateParams message handler.
func TestMsgServer_UpdateParams(t *testing.T) {
	cases := []struct {
		name         string
		authority    string
		newAuthority string
		err          error
	}{
		{
			name:         "success",
			authority:    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
			newAuthority: userAddrB,
		},
		{
			name:         "failure: unauthorized",
			authority:    userAddrA,
			newAuthority: userAddrB,
			err:          types.ErrUnauthorized,
		},
		{
			name:         "failure: invalid new authority format",
			authority:    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
			newAuthority: "invalid-address",
			err:          types.ErrInvalidSigner,
		},
		{
			name:         "failure: empty new authority",
			authority:    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
			newAuthority: "",
			err:          types.ErrInvalidSigner,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wfapp, ctx := setupIntegrationApp(t)
			ms := keeper.NewMsgServerImpl(wfapp.IFTKeeper)

			msg := &types.MsgUpdateParams{
				Authority: tc.authority,
				Params: types.Params{
					Authority: tc.newAuthority,
				},
			}

			_, err := ms.UpdateParams(ctx, msg)

			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}

			require.NoError(t, err)

			// Verify params were updated
			params, err := wfapp.IFTKeeper.ParamsStore.Get(ctx)
			require.NoError(t, err)
			require.Equal(t, tc.newAuthority, params.Authority)
		})
	}
}

// TestMsgServer_IFTMint_ValidationErrors tests IFTMint validation errors.
func TestMsgServer_IFTMint_ValidationErrors(t *testing.T) {
	cases := []struct {
		name   string
		denom  string
		amount math.Int
		err    error
	}{
		{
			name:   "failure: invalid denom - empty",
			denom:  "",
			amount: math.NewInt(1000),
			err:    types.ErrInvalidDenom,
		},
		{
			name:   "failure: invalid amount - zero",
			denom:  testDenom,
			amount: math.ZeroInt(),
			err:    types.ErrInvalidAmount,
		},
		{
			name:   "failure: invalid amount - negative",
			denom:  testDenom,
			amount: math.NewInt(-100),
			err:    types.ErrInvalidAmount,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wfapp, ctx := setupIntegrationApp(t)
			ms := keeper.NewMsgServerImpl(wfapp.IFTKeeper)

			msg := &types.MsgIFTMint{
				Signer:   userAddrA,
				Denom:    tc.denom,
				Receiver: userAddrB,
				Amount:   tc.amount,
			}

			_, err := ms.IFTMint(ctx, msg)
			require.ErrorIs(t, err, tc.err)
		})
	}
}

// TestMsgServer_IFTMint_InvalidSigner tests invalid signer addresses
func TestMsgServer_IFTMint_InvalidSigner(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServerImpl(wfapp.IFTKeeper)

	createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)

	msg := &types.MsgIFTMint{
		Signer:   invalidAddr,
		Denom:    testDenom,
		Receiver: userAddrB,
		Amount:   math.NewInt(1000),
	}

	_, err := ms.IFTMint(ctx, msg)
	require.ErrorIs(t, err, types.ErrInvalidSigner)
}

// TestMsgServer_IFTMint_InvalidReceiver tests invalid receiver addresses
func TestMsgServer_IFTMint_InvalidReceiver(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServerImpl(wfapp.IFTKeeper)

	createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)

	msg := &types.MsgIFTMint{
		Signer:   userAddrA,
		Denom:    testDenom,
		Receiver: invalidAddr,
		Amount:   math.NewInt(1000),
	}

	_, err := ms.IFTMint(ctx, msg)
	require.ErrorIs(t, err, types.ErrInvalidReceiver)
}

// TestMsgServer_IFTMint_UnauthorizedSender tests that non-ICS27 accounts cannot mint
func TestMsgServer_IFTMint_UnauthorizedSender(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServerImpl(wfapp.IFTKeeper)

	createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)

	// Use a regular account (not an ICS27 account)
	msg := &types.MsgIFTMint{
		Signer:   userAddrA, // Regular account, not ICS27
		Denom:    testDenom,
		Receiver: userAddrB,
		Amount:   math.NewInt(1000),
	}

	_, err := ms.IFTMint(ctx, msg)
	// Should fail because userAddrA is not an ICS27 account
	require.ErrorIs(t, err, types.ErrUnauthorizedSender)
}

// TestMsgServer_IFTMint_WrongCounterpartySender tests that GMP ICA with wrong sender cannot mint
func TestMsgServer_IFTMint_WrongCounterpartySender(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServerImpl(wfapp.IFTKeeper)

	clientID := createIBCClient(t, ctx, wfapp)
	createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)

	// Register bridge with counterparty address remoteIFTAddrA
	signer := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	_, err := ms.RegisterIFTBridge(ctx, &types.MsgRegisterIFTBridge{
		Signer:                 signer,
		Denom:                  testDenom,
		ClientId:               clientID,
		CounterpartyIftAddress: remoteIFTAddrA,
		IftSendCallConstructor: types.ConstructorEVM,
	})
	require.NoError(t, err)

	// Create a GMP ICA with WRONG sender address (remoteIFTAddrB instead of remoteIFTAddrA)
	wrongSenderAccountID := gmptypes.NewAccountIdentifier(clientID, remoteIFTAddrB, nil)
	icaAddr, err := gmptypes.BuildAddressPredictable(&wrongSenderAccountID)
	require.NoError(t, err)

	// Register the ICA in GMP keeper
	ics27Account := gmptypes.NewICS27Account(icaAddr.String(), &wrongSenderAccountID)
	err = wfapp.GMPKeeper.AccountsByAddress.Set(ctx, icaAddr, ics27Account)
	require.NoError(t, err)

	// Try to mint using the ICA with wrong sender
	msg := &types.MsgIFTMint{
		Signer:   icaAddr.String(),
		Denom:    testDenom,
		Receiver: userAddrB,
		Amount:   math.NewInt(1000),
	}

	_, err = ms.IFTMint(ctx, msg)
	// Should fail because ICA sender doesn't match bridge's CounterpartyIftAddress
	require.ErrorIs(t, err, types.ErrUnauthorizedSender)
}

// TestMsgServer_IFTMint_UnexpectedSalt tests that ICS27 accounts with salt cannot mint
func TestMsgServer_IFTMint_UnexpectedSalt(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServerImpl(wfapp.IFTKeeper)

	clientID := createIBCClient(t, ctx, wfapp)
	createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)

	// Register bridge
	signer := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	_, err := ms.RegisterIFTBridge(ctx, &types.MsgRegisterIFTBridge{
		Signer:                 signer,
		Denom:                  testDenom,
		ClientId:               clientID,
		CounterpartyIftAddress: remoteIFTAddrA,
		IftSendCallConstructor: types.ConstructorEVM,
	})
	require.NoError(t, err)

	// Create a GMP ICA with salt (not allowed for IFT)
	salt := []byte("unexpected-salt")
	accountIDWithSalt := gmptypes.NewAccountIdentifier(clientID, remoteIFTAddrA, salt)
	icaAddr, err := gmptypes.BuildAddressPredictable(&accountIDWithSalt)
	require.NoError(t, err)

	// Register the ICA in GMP keeper
	ics27Account := gmptypes.NewICS27Account(icaAddr.String(), &accountIDWithSalt)
	err = wfapp.GMPKeeper.AccountsByAddress.Set(ctx, icaAddr, ics27Account)
	require.NoError(t, err)

	// Try to mint using the ICA with salt
	msg := &types.MsgIFTMint{
		Signer:   icaAddr.String(),
		Denom:    testDenom,
		Receiver: userAddrB,
		Amount:   math.NewInt(1000),
	}

	_, err = ms.IFTMint(ctx, msg)
	require.ErrorIs(t, err, types.ErrUnexpectedSalt)
}

// TestMsgServer_RegisterIFTBridge_MultipleBridges tests registering multiple bridges for the same denom
func TestMsgServer_RegisterIFTBridge_MultipleBridges(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServerImpl(wfapp.IFTKeeper)

	signer := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	denom := testDenom

	// Create token factory denom
	createTokenFactoryDenom(t, ctx, wfapp, adminAddr, denom)

	// Create two different IBC clients
	clientID1 := createIBCClient(t, ctx, wfapp)
	clientID2 := createIBCClient(t, ctx, wfapp)

	// Register first bridge
	msg1 := &types.MsgRegisterIFTBridge{
		Signer:                 signer,
		Denom:                  denom,
		ClientId:               clientID1,
		CounterpartyIftAddress: remoteIFTAddrA,
		IftSendCallConstructor: types.ConstructorEVM,
	}
	_, err := ms.RegisterIFTBridge(ctx, msg1)
	require.NoError(t, err)

	// Register second bridge for same denom but different client
	msg2 := &types.MsgRegisterIFTBridge{
		Signer:                 signer,
		Denom:                  denom,
		ClientId:               clientID2,
		CounterpartyIftAddress: remoteIFTAddrB,
		IftSendCallConstructor: types.ConstructorCosmos,
	}
	_, err = ms.RegisterIFTBridge(ctx, msg2)
	require.NoError(t, err)

	// Verify both bridges exist
	bridge1, err := wfapp.IFTKeeper.IFTBridgeStore.Get(ctx, collections.Join(denom, clientID1))
	require.NoError(t, err)
	require.Equal(t, remoteIFTAddrA, bridge1.CounterpartyIftAddress)

	bridge2, err := wfapp.IFTKeeper.IFTBridgeStore.Get(ctx, collections.Join(denom, clientID2))
	require.NoError(t, err)
	require.Equal(t, remoteIFTAddrB, bridge2.CounterpartyIftAddress)
}

// TestMsgServer_IFTTransfer_InsufficientBalance tests transfer with insufficient balance
func TestMsgServer_IFTTransfer_InsufficientBalance(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServerImpl(wfapp.IFTKeeper)

	signer := userAddrA
	denom := testDenom
	receiver := remoteIFTAddrA

	// Setup
	createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)
	clientID := createIBCClient(t, ctx, wfapp)

	// Register bridge
	bridge := types.IFTBridge{
		ClientId:               clientID,
		CounterpartyIftAddress: remoteIFTAddrB,
		IftSendCallConstructor: types.ConstructorEVM,
	}
	err := wfapp.IFTKeeper.IFTBridgeStore.Set(ctx, collections.Join(denom, clientID), bridge)
	require.NoError(t, err)

	// Mint only 100 tokens
	senderAddr := sdk.MustAccAddressFromBech32(signer)
	mintTokens(t, ctx, wfapp, testDenom, math.NewInt(100), senderAddr)

	// Try to transfer 1000 tokens (more than balance)
	timeout := uint64(time.Now().Add(30 * time.Second).Unix())
	msg := &types.MsgIFTTransfer{
		Signer:           signer,
		Denom:            denom,
		ClientId:         clientID,
		Receiver:         receiver,
		Amount:           math.NewInt(1000),
		TimeoutTimestamp: timeout,
	}

	_, err = ms.IFTTransfer(ctx, msg)
	// Should fail due to insufficient funds
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrBurnFailed)
}

// TestMsgServer_IFTMint_Success tests successful token minting via GMP ICA
func TestMsgServer_IFTMint_Success(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServerImpl(wfapp.IFTKeeper)

	clientID := createIBCClient(t, ctx, wfapp)
	createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)

	// Register bridge with specific counterparty address
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	_, err := ms.RegisterIFTBridge(ctx, &types.MsgRegisterIFTBridge{
		Signer:                 authority,
		Denom:                  testDenom,
		ClientId:               clientID,
		CounterpartyIftAddress: remoteIFTAddrA,
		IftSendCallConstructor: types.ConstructorEVM,
	})
	require.NoError(t, err)

	// Create a valid GMP ICA with matching counterparty address (no salt)
	accountID := gmptypes.NewAccountIdentifier(clientID, remoteIFTAddrA, nil)
	icaAddr, err := gmptypes.BuildAddressPredictable(&accountID)
	require.NoError(t, err)

	// Register the ICA in GMP keeper
	ics27Account := gmptypes.NewICS27Account(icaAddr.String(), &accountID)
	err = wfapp.GMPKeeper.AccountsByAddress.Set(ctx, icaAddr, ics27Account)
	require.NoError(t, err)

	// Mint tokens via IFTMint
	mintAmount := math.NewInt(1000000)
	receiver, err := sdk.AccAddressFromBech32(userAddrB)
	require.NoError(t, err)

	msg := &types.MsgIFTMint{
		Signer:   icaAddr.String(),
		Denom:    testDenom,
		Receiver: userAddrB,
		Amount:   mintAmount,
	}

	_, err = ms.IFTMint(ctx, msg)
	require.NoError(t, err)

	// Verify tokens were minted to receiver
	balance := wfapp.BankKeeper.GetBalance(ctx, receiver, testDenom)
	require.True(t, balance.Amount.Equal(mintAmount), "expected %s, got %s", mintAmount, balance.Amount)

	// Verify event was emitted
	events := ctx.EventManager().Events()
	var foundMintEvent bool
	for _, event := range events {
		if event.Type == types.EventTypeIFTMintReceived {
			foundMintEvent = true
			break
		}
	}
	require.True(t, foundMintEvent, "IFT mint received event should be emitted")
}

// TestMsgServer_IFTMint_BridgeNotFound tests that IFTMint fails when no bridge exists
func TestMsgServer_IFTMint_BridgeNotFound(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServerImpl(wfapp.IFTKeeper)

	clientID := createIBCClient(t, ctx, wfapp)
	createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)

	// Create a GMP ICA but do NOT register bridge
	accountID := gmptypes.NewAccountIdentifier(clientID, remoteIFTAddrA, nil)
	icaAddr, err := gmptypes.BuildAddressPredictable(&accountID)
	require.NoError(t, err)

	ics27Account := gmptypes.NewICS27Account(icaAddr.String(), &accountID)
	err = wfapp.GMPKeeper.AccountsByAddress.Set(ctx, icaAddr, ics27Account)
	require.NoError(t, err)

	msg := &types.MsgIFTMint{
		Signer:   icaAddr.String(),
		Denom:    testDenom,
		Receiver: userAddrB,
		Amount:   math.NewInt(1000),
	}

	_, err = ms.IFTMint(ctx, msg)
	require.ErrorIs(t, err, types.ErrBridgeNotFound)
}

// TestMsgServer_IFTTransfer_EventPropagation tests that send_packet events from
// the GMP handler are properly propagated to the context's EventManager.
// This is critical for the relayer to detect outgoing packets.
func TestMsgServer_IFTTransfer_EventPropagation(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServerImpl(wfapp.IFTKeeper)

	signer := userAddrA
	denom := testDenom
	receiver := remoteIFTAddrA

	// Setup
	createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)
	clientID := createIBCClient(t, ctx, wfapp)

	// Register bridge
	bridge := types.IFTBridge{
		ClientId:               clientID,
		CounterpartyIftAddress: remoteIFTAddrB,
		IftSendCallConstructor: types.ConstructorEVM,
	}
	err := wfapp.IFTKeeper.IFTBridgeStore.Set(ctx, collections.Join(denom, clientID), bridge)
	require.NoError(t, err)

	// Mint tokens to sender
	senderAddr := sdk.MustAccAddressFromBech32(signer)
	mintTokens(t, ctx, wfapp, testDenom, math.NewInt(2000000), senderAddr)

	timeout := uint64(time.Now().Add(30 * time.Second).Unix())
	msg := &types.MsgIFTTransfer{
		Signer:           signer,
		Denom:            denom,
		ClientId:         clientID,
		Receiver:         receiver,
		Amount:           math.NewInt(1000000),
		TimeoutTimestamp: timeout,
	}

	resp, err := ms.IFTTransfer(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Positive(t, resp.Sequence)

	// Verify that send_packet event is present with encoded_packet_hex attribute
	// This is what the relayer uses to detect outgoing packets
	events := ctx.EventManager().Events()
	var foundSendPacket bool
	var foundEncodedPacketHex bool

	for _, event := range events {
		if event.Type == channelv2types.EventTypeSendPacket {
			foundSendPacket = true
			for _, attr := range event.Attributes {
				if attr.Key == channelv2types.AttributeKeyEncodedPacketHex {
					foundEncodedPacketHex = true
					require.NotEmpty(t, attr.Value, "encoded_packet_hex should not be empty")
					break
				}
			}
			break
		}
	}

	require.True(t, foundSendPacket, "send_packet event should be emitted")
	require.True(t, foundEncodedPacketHex, "send_packet event should have encoded_packet_hex attribute")
}

// TestConstructMintCall_EVM tests the EVM mint call constructor
func TestConstructMintCall_EVM(t *testing.T) {
	wfapp, _ := setupIntegrationApp(t)

	evmReceiver := "0x1234567890abcdef1234567890abcdef12345678"
	amount := math.NewInt(1000000)

	payload, err := types.ConstructMintCall(wfapp.AppCodec(), evmReceiver, amount, types.ConstructorEVM, "", "")
	require.NoError(t, err)
	require.NotEmpty(t, payload)

	// Verify function selector
	expectedSelector := crypto.Keccak256([]byte("iftMint(address,uint256)"))[:4]
	require.Equal(t, expectedSelector, payload[:4], "function selector mismatch")

	// Verify ABI-encoded address (32 bytes: 12 zero bytes + 20 address bytes)
	addressBytes := payload[4:36]
	require.Equal(t, make([]byte, 12), addressBytes[:12], "address should be left-padded with zeros")
	expectedAddr := common.HexToAddress(evmReceiver)
	require.Equal(t, expectedAddr.Bytes(), addressBytes[12:], "address should match")

	// Verify ABI-encoded amount (32 bytes: big-endian uint256)
	amountBytes := payload[36:68]
	decodedAmount := new(big.Int).SetBytes(amountBytes)
	require.Equal(t, amount.BigInt().String(), decodedAmount.String(), "amount should match")
}

// TestConstructMintCall_CosmosTx tests the CosmosTx mint call constructor
func TestConstructMintCall_CosmosTx(t *testing.T) {
	wfapp, _ := setupIntegrationApp(t)

	cosmosReceiver := userAddrB
	amount := math.NewInt(1000000)
	icaAddress := userAddrA

	payload, err := types.ConstructMintCall(wfapp.AppCodec(), cosmosReceiver, amount, types.ConstructorCosmos, testDenom, icaAddress)
	require.NoError(t, err)
	require.NotEmpty(t, payload)

	// Verify we can decode the CosmosTx
	var cosmosTx gmptypes.CosmosTx
	err = cosmosTx.Unmarshal(payload)
	require.NoError(t, err, "should be valid CosmosTx protobuf")
	require.Len(t, cosmosTx.Messages, 1, "CosmosTx should contain exactly one message")

	// Verify the message is a MsgIFTMint
	var mintMsg types.MsgIFTMint
	err = wfapp.AppCodec().Unmarshal(cosmosTx.Messages[0].Value, &mintMsg)
	require.NoError(t, err, "should be valid MsgIFTMint")
	require.Equal(t, icaAddress, mintMsg.Signer)
	require.Equal(t, testDenom, mintMsg.Denom)
	require.Equal(t, cosmosReceiver, mintMsg.Receiver)
	require.True(t, amount.Equal(mintMsg.Amount))
}

// TestConstructMintCall_InvalidConstructor tests invalid constructor type
func TestConstructMintCall_InvalidConstructor(t *testing.T) {
	wfapp, _ := setupIntegrationApp(t)

	_, err := types.ConstructMintCall(wfapp.AppCodec(), userAddrB, math.NewInt(1000), "invalid", testDenom, userAddrA)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid constructor type")
}

// TestMsgServer_RegisterIFTBridge_InvalidClientID tests that registering a bridge with empty client ID fails
func TestMsgServer_RegisterIFTBridge_InvalidClientID(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServerImpl(wfapp.IFTKeeper)

	signer := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)

	msg := &types.MsgRegisterIFTBridge{
		Signer:                 signer,
		Denom:                  testDenom,
		ClientId:               "",
		CounterpartyIftAddress: remoteIFTAddrA,
		IftSendCallConstructor: types.ConstructorEVM,
	}

	_, err := ms.RegisterIFTBridge(ctx, msg)
	require.ErrorIs(t, err, types.ErrInvalidClientID)
}

// TestMsgServer_IFTMint_DenomNotInTokenFactory tests that IFTMint fails when denom doesn't exist in token factory
func TestMsgServer_IFTMint_DenomNotInTokenFactory(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServerImpl(wfapp.IFTKeeper)

	clientID := createIBCClient(t, ctx, wfapp)
	nonExistentDenom := "nonexistent"

	// Register bridge directly (bypassing RegisterIFTBridge validation)
	bridge := types.IFTBridge{
		ClientId:               clientID,
		CounterpartyIftAddress: remoteIFTAddrA,
		IftSendCallConstructor: types.ConstructorEVM,
	}
	err := wfapp.IFTKeeper.IFTBridgeStore.Set(ctx, collections.Join(nonExistentDenom, clientID), bridge)
	require.NoError(t, err)

	// Create GMP ICA
	accountID := gmptypes.NewAccountIdentifier(clientID, remoteIFTAddrA, nil)
	icaAddr, err := gmptypes.BuildAddressPredictable(&accountID)
	require.NoError(t, err)

	ics27Account := gmptypes.NewICS27Account(icaAddr.String(), &accountID)
	err = wfapp.GMPKeeper.AccountsByAddress.Set(ctx, icaAddr, ics27Account)
	require.NoError(t, err)

	msg := &types.MsgIFTMint{
		Signer:   icaAddr.String(),
		Denom:    nonExistentDenom,
		Receiver: userAddrB,
		Amount:   math.NewInt(1000),
	}

	_, err = ms.IFTMint(ctx, msg)
	require.ErrorIs(t, err, types.ErrDenomNotFound)
}

// TestMsgServer_RemoveIFTBridge_EmptyDenom tests that removing bridge with empty denom fails
func TestMsgServer_RemoveIFTBridge_EmptyDenom(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServerImpl(wfapp.IFTKeeper)

	signer := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	clientID := createIBCClient(t, ctx, wfapp)

	msg := &types.MsgRemoveIFTBridge{
		Signer:   signer,
		Denom:    "",
		ClientId: clientID,
	}

	_, err := ms.RemoveIFTBridge(ctx, msg)
	require.ErrorIs(t, err, types.ErrBridgeNotFound)
}

// TestMsgServer_IFTTransfer_SolanaEncoding tests that Solana transfers use protobuf encoding
func TestMsgServer_IFTTransfer_SolanaEncoding(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)
	ms := keeper.NewMsgServerImpl(wfapp.IFTKeeper)

	signer := userAddrA
	denom := testDenom
	receiver := solanaReceiverAddr

	// Setup
	createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)
	clientID := createIBCClient(t, ctx, wfapp)

	// Register Solana bridge with JSON constructor
	solanaConstructor := `{"solana":{"gmp_program_id":"` + solanaGMPProgramID + `","mint_pubkey":"` + solanaMintPubkey + `"}}`
	bridge := types.IFTBridge{
		ClientId:               clientID,
		CounterpartyIftAddress: solanaIFTProgramID,
		IftSendCallConstructor: solanaConstructor,
	}
	err := wfapp.IFTKeeper.IFTBridgeStore.Set(ctx, collections.Join(denom, clientID), bridge)
	require.NoError(t, err)

	// Mint tokens to sender
	senderAddr := sdk.MustAccAddressFromBech32(signer)
	mintTokens(t, ctx, wfapp, testDenom, math.NewInt(2000000), senderAddr)

	timeout := uint64(time.Now().Add(30 * time.Second).Unix())
	msg := &types.MsgIFTTransfer{
		Signer:           signer,
		Denom:            denom,
		ClientId:         clientID,
		Receiver:         receiver,
		Amount:           math.NewInt(1000000),
		TimeoutTimestamp: timeout,
	}

	resp, err := ms.IFTTransfer(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Positive(t, resp.Sequence)

	// Verify send_packet event exists (encoding is used internally by GMP)
	events := ctx.EventManager().Events()
	var foundSendPacket bool
	for _, event := range events {
		if event.Type == channelv2types.EventTypeSendPacket {
			foundSendPacket = true
			break
		}
	}
	require.True(t, foundSendPacket, "send_packet event should be emitted for Solana transfer")
}

// TestMsgServer_IFTTransfer_TimeoutValidation tests timeout validation
func TestMsgServer_IFTTransfer_TimeoutValidation(t *testing.T) {
	cases := []struct {
		name            string
		timeoutOffset   time.Duration
		expectErr       bool
		expectedErrType error
	}{
		{
			name:            "timeout in past",
			timeoutOffset:   -1 * time.Hour,
			expectErr:       true,
			expectedErrType: types.ErrInvalidTimeout,
		},
		{
			name:            "timeout equals block time",
			timeoutOffset:   0,
			expectErr:       true,
			expectedErrType: types.ErrInvalidTimeout,
		},
		{
			name:          "timeout 1 second in future",
			timeoutOffset: 1 * time.Second,
			expectErr:     false,
		},
		{
			name:          "timeout 15 minutes in future",
			timeoutOffset: 15 * time.Minute,
			expectErr:     false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wfapp, ctx := setupIntegrationApp(t)
			ms := keeper.NewMsgServerImpl(wfapp.IFTKeeper)

			signer := userAddrA
			denom := testDenom
			receiver := remoteIFTAddrA

			createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)
			clientID := createIBCClient(t, ctx, wfapp)

			bridge := types.IFTBridge{
				ClientId:               clientID,
				CounterpartyIftAddress: remoteIFTAddrB,
				IftSendCallConstructor: types.ConstructorEVM,
			}
			err := wfapp.IFTKeeper.IFTBridgeStore.Set(ctx, collections.Join(denom, clientID), bridge)
			require.NoError(t, err)

			senderAddr := sdk.MustAccAddressFromBech32(signer)
			mintTokens(t, ctx, wfapp, testDenom, math.NewInt(2000000), senderAddr)

			timeout := uint64(ctx.BlockTime().Add(tc.timeoutOffset).Unix())
			msg := &types.MsgIFTTransfer{
				Signer:           signer,
				Denom:            denom,
				ClientId:         clientID,
				Receiver:         receiver,
				Amount:           math.NewInt(1000000),
				TimeoutTimestamp: timeout,
			}

			_, err = ms.IFTTransfer(ctx, msg)

			if tc.expectErr {
				require.Error(t, err)
				if tc.expectedErrType != nil {
					require.ErrorIs(t, err, tc.expectedErrType)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
