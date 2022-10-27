<!--
order: 2
-->

# Integration

Learn how to configure the Fee Middleware module with IBC applications. The following document is intended for developers building on top of the Cosmos SDK and only applies for Cosmos SDK chains. {synopsis}

## Pre-requisite Readings

* [IBC middleware development](../../ibc/middleware/develop.md) {prereq}
* [IBC middleware integration](../../ibc/middleware/integration.md) {prereq}

The Fee Middleware module, as the name suggests, plays the role of an IBC middleware and as such must be configured by chain developers to route and handle IBC messages correctly.
For Cosmos SDK chains this setup is done via the `app/app.go` file, where modules are constructed and configured in order to bootstrap the blockchain application.

## Example integration of the Fee Middleware module

```
// app.go

// Register the AppModule for the fee middleware module
ModuleBasics = module.NewBasicManager(
    ...
    ibcfee.AppModuleBasic{},
    ...
)

... 

// Add module account permissions for the fee middleware module
maccPerms = map[string][]string{
    ...
    ibcfeetypes.ModuleName:            nil,
}

...

// Add fee middleware Keeper
type App struct {
    ...

    IBCFeeKeeper ibcfeekeeper.Keeper

    ...
}

...

// Create store keys 
keys := sdk.NewKVStoreKeys(
    ...
    ibcfeetypes.StoreKey,
    ...
)

... 

app.IBCFeeKeeper = ibcfeekeeper.NewKeeper(
	appCodec, keys[ibcfeetypes.StoreKey],
	app.IBCKeeper.ChannelKeeper, // may be replaced with IBC middleware
	app.IBCKeeper.ChannelKeeper,
	&app.IBCKeeper.PortKeeper, app.AccountKeeper, app.BankKeeper,
)


// See the section below for configuring an application stack with the fee middleware module

...

// Register fee middleware AppModule
app.moduleManager = module.NewManager(
    ...
    ibcfee.NewAppModule(app.IBCFeeKeeper),
)

...

// Add fee middleware to begin blocker logic
app.mm.SetOrderBeginBlockers(
    ...
    ibcfeetypes.ModuleName,
    ...
)

// Add fee middleware to end blocker logic
app.mm.SetOrderEndBlockers(
    ...
    ibcfeetypes.ModuleName,
    ...
)

// Add fee middleware to init genesis logic
app.mm.SetOrderInitGenesis(
    ...
    ibcfeetypes.ModuleName,
    ...
)
```

## Configuring an application stack with Fee Middleware

As mentioned in [IBC middleware development](../../ibc/middleware/develop.md) an application stack may be composed of many or no middlewares that nest a base application. 
These layers form the complete set of application logic that enable developers to build composable and flexible IBC application stacks.
For example, an application stack may be just a single base application like `transfer`, however, the same application stack composed with `29-fee` will nest the `transfer` base application
by wrapping it with the Fee Middleware module.


### Transfer

See below for an example of how to create an application stack using `transfer` and `29-fee`.
The following `transferStack` is configured in `app/app.go` and added to the IBC `Router`.
The in-line comments describe the execution flow of packets between the application stack and IBC core.

```go
// Create Transfer Stack
// SendPacket, since it is originating from the application to core IBC:
// transferKeeper.SendPacket -> fee.SendPacket -> channel.SendPacket

// RecvPacket, message that originates from core IBC and goes down to app, the flow is the other way
// channel.RecvPacket -> fee.OnRecvPacket -> transfer.OnRecvPacket

// transfer stack contains (from top to bottom):
// - IBC Fee Middleware
// - Transfer

// create IBC module from bottom to top of stack
var transferStack porttypes.IBCModule
transferStack = transfer.NewIBCModule(app.TransferKeeper)
transferStack = ibcfee.NewIBCMiddleware(transferStack, app.IBCFeeKeeper)

// Add transfer stack to IBC Router
ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferStack)
```

### Interchain Accounts

See below for an example of how to create an application stack using `27-interchain-accounts` and `29-fee`.
The following `icaControllerStack` and `icaHostStack` are configured in `app/app.go` and added to the IBC `Router` with the associated authentication module.
The in-line comments describe the execution flow of packets between the application stack and IBC core.

```go
// Create Interchain Accounts Stack
// SendPacket, since it is originating from the application to core IBC:
// icaAuthModuleKeeper.SendTx -> icaController.SendPacket -> fee.SendPacket -> channel.SendPacket

// initialize ICA module with mock module as the authentication module on the controller side
var icaControllerStack porttypes.IBCModule
icaControllerStack = ibcmock.NewIBCModule(&mockModule, ibcmock.NewMockIBCApp("", scopedICAMockKeeper))
app.ICAAuthModule = icaControllerStack.(ibcmock.IBCModule)
icaControllerStack = icacontroller.NewIBCMiddleware(icaControllerStack, app.ICAControllerKeeper)
icaControllerStack = ibcfee.NewIBCMiddleware(icaControllerStack, app.IBCFeeKeeper)

// RecvPacket, message that originates from core IBC and goes down to app, the flow is:
// channel.RecvPacket -> fee.OnRecvPacket -> icaHost.OnRecvPacket

var icaHostStack porttypes.IBCModule
icaHostStack = icahost.NewIBCModule(app.ICAHostKeeper)
icaHostStack = ibcfee.NewIBCMiddleware(icaHostStack, app.IBCFeeKeeper)

// Add authentication module, controller and host to IBC router
ibcRouter.
    // the ICA Controller middleware needs to be explicitly added to the IBC Router because the
    // ICA controller module owns the port capability for ICA. The ICA authentication module
    // owns the channel capability.
    AddRoute(ibcmock.ModuleName+icacontrollertypes.SubModuleName, icaControllerStack) // ica with mock auth module stack route to ica (top level of middleware stack)
    AddRoute(icacontrollertypes.SubModuleName, icaControllerStack).
    AddRoute(icahosttypes.SubModuleName, icaHostStack).
```
