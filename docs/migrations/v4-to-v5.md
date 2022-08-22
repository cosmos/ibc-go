# Migrating from v4 to v5

This document is intended to highlight significant changes which may require more information than presented in the CHANGELOG.
Any changes that must be done by a user of ibc-go should be documented here.

There are four sections based on the four potential user groups of this document:
- Chains
- IBC Apps
- Relayers
- IBC Light Clients

**Note:** ibc-go supports golang semantic versioning and therefore all imports must be updated to bump the version number on major releases.
```go
github.com/cosmos/ibc-go/v4 -> github.com/cosmos/ibc-go/v5
```

## Chains

### Ante decorator

The `AnteDecorator` type in `core/ante` has been renamed to `RedundantRelayDecorator` (and the corresponding constructor function to `NewRedundantRelayDecorator`). Therefore in the function that creates the instance of the `sdk.AnteHandler` type (e.g. `NewAnteHandler`) the change would be like this:

```diff
func NewAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	// parameter validation

	anteDecorators := []sdk.AnteDecorator{
      // other ante decorators
-     ibcante.NewAnteDecorator(opts.IBCkeeper),
+     ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
	}

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}
```

## IBC Apps

### Core

The `key` parameter of the `NewKeeper` function in `modules/core/keeper` is now of type `storetypes.StoreKey` (where `storetypes` is an import alias for `"github.com/cosmos/cosmos-sdk/store/types"`):

```diff
func NewKeeper(
   cdc codec.BinaryCodec,
-  key sdk.StoreKey,
+  key storetypes.StoreKey,
   paramSpace paramtypes.Subspace,
   stakingKeeper clienttypes.StakingKeeper, 
   upgradeKeeper clienttypes.UpgradeKeeper,
   scopedKeeper capabilitykeeper.ScopedKeeper,
) *Keeper
```

The `RegisterRESTRoutes` function in `modules/core` has been removed.

### ICS03 - Connection

The `key` parameter of the `NewKeeper` function in `modules/core/03-connection/keeper` is now of type `storetypes.StoreKey` (where `storetypes` is an import alias for `"github.com/cosmos/cosmos-sdk/store/types"`):

```diff
func NewKeeper(
   cdc codec.BinaryCodec,
-  key sdk.StoreKey,
+  key storetypes.StoreKey,
   paramSpace paramtypes.Subspace, 
   ck types.ClientKeeper
) Keeper
```

### ICS04 - Channel 

The function `NewPacketId` in `modules/core/04-channel/types` has been renamed to `NewPacketID`:

```diff
-  func NewPacketId(
+  func NewPacketID(
  portID, 
  channelID string, 
  seq uint64
) PacketId 
```

The `key` parameter of the `NewKeeper` function in `modules/core/04-channel/keeper` is now of type `storetypes.StoreKey` (where `storetypes` is an import alias for `"github.com/cosmos/cosmos-sdk/store/types"`):

```diff
func NewKeeper(
   cdc codec.BinaryCodec,
-  key sdk.StoreKey,
+  key storetypes.StoreKey,
   clientKeeper types.ClientKeeper,
   connectionKeeper types.ConnectionKeeper,
   portKeeper types.PortKeeper, 
   scopedKeeper capabilitykeeper.ScopedKeeper,
) Keeper 
```

### ICS20 - Transfer

The `key` parameter of the `NewKeeper` function in `modules/apps/transfer/keeper` is now of type `storetypes.StoreKey` (where `storetypes` is an import alias for `"github.com/cosmos/cosmos-sdk/store/types"`):

```diff
func NewKeeper(
   cdc codec.BinaryCodec,
-  key sdk.StoreKey,
+  key storetypes.StoreKey, 
   paramSpace paramtypes.Subspace,
   ics4Wrapper types.ICS4Wrapper, 
   channelKeeper types.ChannelKeeper,
   portKeeper types.PortKeeper,
   authKeeper types.AccountKeeper,
   bankKeeper types.BankKeeper,
   scopedKeeper capabilitykeeper.ScopedKeeper,
) Keeper
```

The `amount` parameter of function `GetTransferCoin` in `modules/apps/transfer/types` is now of type `math.Int` (`"cosmossdk.io/math"`):

```diff
func GetTransferCoin(
   portID, channelID, baseDenom string,
-  amount sdk.Int
+  amount math.Int
) sdk.Coin
```

The `RegisterRESTRoutes` function in `modules/apps/transfer` has been removed.

### ICS27 - Interchain Accounts

The `key` parameter of the `NewKeeper` functions in 

- `modules/apps/27-interchain-accounts/controller/keeper` 
- and `modules/apps/27-interchain-accounts/host/keeper` 

 is now of type `storetypes.StoreKey` (where `storetypes` is an import alias for `"github.com/cosmos/cosmos-sdk/store/types"`):

```diff
// NewKeeper creates a new interchain accounts controller Keeper instance
func NewKeeper(
   cdc codec.BinaryCodec,
-  key sdk.StoreKey,
+  key storetypes.StoreKey,
   paramSpace paramtypes.Subspace,
   ics4Wrapper icatypes.ICS4Wrapper,
   channelKeeper icatypes.ChannelKeeper,
   portKeeper icatypes.PortKeeper,
   scopedKeeper capabilitykeeper.ScopedKeeper,
   msgRouter *baseapp.MsgServiceRouter,
) Keeper  
```

```diff
// NewKeeper creates a new interchain accounts host Keeper instance
func NewKeeper(
   cdc codec.BinaryCodec,
-  key sdk.StoreKey,
+  key storetypes.StoreKey,
   paramSpace paramtypes.Subspace,
   channelKeeper icatypes.ChannelKeeper,
   portKeeper icatypes.PortKeeper,
   accountKeeper icatypes.AccountKeeper,
   scopedKeeper capabilitykeeper.ScopedKeeper,
   msgRouter *baseapp.MsgServiceRouter,
) Keeper 
```

The `RegisterRESTRoutes` function in `modules/apps/27-interchain-accounts` has been removed.

####

The response of a message execution on the host chain is constructed now like this:

```
&codectypes.Any{
  TypeUrl: sdk.MsgTypeURL(msg),
  Value:   msgResponse,
}
```

See [ADR-03](../architecture/adr-003-ics27-acknowledgement.md/#next-major-version-format) for more information.

### ICS29 - Fee Middleware

The `key` parameter of the `NewKeeper` function in `modules/apps/29-fee` is now of type `storetypes.StoreKey` (where `storetypes` is an import alias for `"github.com/cosmos/cosmos-sdk/store/types"`):

```diff
func NewKeeper(
   cdc codec.BinaryCodec,
-  key sdk.StoreKey,
+  key storetypes.StoreKey,
   paramSpace paramtypes.Subspace,
   ics4Wrapper types.ICS4Wrapper,
   channelKeeper types.ChannelKeeper,
   portKeeper types.PortKeeper,
   authKeeper types.AccountKeeper,
   bankKeeper types.BankKeeper,
) Keeper 
```

The `RegisterRESTRoutes` function in `modules/apps/29-fee` has been removed.

### IBC testing package

The `MockIBCApp` type has been renamed to `IBCApp` (and the corresponding constructor function to `NewIBCApp`). This has resulted therefore in:

- The `IBCApp` field of the `*IBCModule` in `testing/mock` to change its type as well to `*IBCApp`.
- The `app` parameter to `*NewIBCModule` in `testing/mock` to change its type as well to `*IBCApp`.

The `MockEmptyAcknowledgement` field has been renamed to `EmptyAcknowledgement` (and the corresponding constructor function to `NewEmptyAcknowledgement`).

The return type of the function `LastCommitID` of the `TestingApp` interface in `testing` has changed to `storetypes.CommitID` (where `storetypes` is an import alias for `"github.com/cosmos/cosmos-sdk/store/types"`):

```diff
type TestingApp interface {
   abci.Application
   
   // ibc-go additions
   GetBaseApp() *baseapp.BaseApp
   GetStakingKeeper() stakingkeeper.Keeper
   GetIBCKeeper() *keeper.Keeper
   GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper
   GetTxConfig() client.TxConfig

   // Implemented by SimApp
   AppCodec() codec.Codec
  
   // Implemented by BaseApp
-  LastCommitID() sdk.CommitID
+  LastCommitID() storetypes.CommitID
   LastBlockHeight() int64
}
```

The `powerReduction` parameter of the function `SetupWithGenesisValSet` in `testing` is now of type `math.Int` (`"cosmossdk.io/math"`):

```diff
func SetupWithGenesisValSet(
   t *testing.T,
   valSet *tmtypes.ValidatorSet,
   genAccs []authtypes.GenesisAccount,
   chainID string,
-  powerReduction sdk.Int,
+  powerReduction math.Int,
   balances ...banktypes.Balance
) TestingApp
```

The `accAmt` parameter of the functions

- `AddTestAddrsFromPubKeys` ,
- `AddTestAddrs`
- and `AddTestAddrsIncremental`

in `testing/simapp` are now of type `math.Int` (`"cosmossdk.io/math"`):

```diff
func AddTestAddrsFromPubKeys(
   app *SimApp,
   ctx sdk.Context,
   pubKeys []cryptotypes.PubKey,
-  accAmt sdk.Int,
+  accAmt math.Int
) 
func addTestAddrs(
   app *SimApp,
   ctx sdk.Context,
   accNum int,
-  accAmt sdk.Int,
+  accAmt math.Int,
   strategy GenerateAccountStrategy
) []sdk.AccAddress
func AddTestAddrsIncremental(
   app *SimApp,
   ctx sdk.Context,
   accNum int,
-  accAmt sdk.Int,
+  accAmt math.Int
) []sdk.AccAddress
```

The `RegisterRESTRoutes` function in `testing/mock` has been removed.

## Relayers

- No relevant changes were made in this release.

## IBC Light Clients

### ICS02 - Client

The `key` parameter of the `NewKeeper` function in `modules/core/02-client/keeper` is now of type `storetypes.StoreKey` (where `storetypes` is an import alias for `"github.com/cosmos/cosmos-sdk/store/types"`):

```diff
func NewKeeper(
   cdc codec.BinaryCodec,
-  key sdk.StoreKey,
+  key storetypes.StoreKey,
   paramSpace paramtypes.Subspace,
   sk types.StakingKeeper,
   uk types.UpgradeKeeper
) Keeper
```
