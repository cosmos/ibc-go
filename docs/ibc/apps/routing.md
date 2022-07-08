<!--
order: 6
-->

# Routing

## Pre-requisites Readings

- [IBC Overview](../overview.md)) {prereq}
- [IBC default integration](../integration.md) {prereq}

Learn how to hook a route to the IBC router for the custom IBC module. {synopsis}

As mentioned above, modules must implement the `IBCModule` interface (which contains both channel
handshake callbacks and packet handling callbacks). The concrete implementation of this interface
must be registered with the module name as a route on the IBC `Router`.

```go
// app.go
func NewApp(...args) *App {
// ...

// Create static IBC router, add module routes, then set and seal it
ibcRouter := port.NewRouter()

ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferModule)
// Note: moduleCallbacks must implement IBCModule interface
ibcRouter.AddRoute(moduleName, moduleCallbacks)

// Setting Router will finalize all routes by sealing router
// No more routes can be added
app.IBCKeeper.SetRouter(ibcRouter)

// ...
}
```
