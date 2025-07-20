---
title: Support the new StackBuilder primitive for Wiring Middlewares in the chain application
sidebar_label: Support StackBuilder Wiring
sidebar_position: 1
slug: /migrations/support-stackbuilder
---

# Migration for Chains wishing to use StackBuilder

The StackBuilder struct is a new primitive for wiring middleware in a simpler and less error-prone manner. It is not a breaking change thus the existing method of wiring middleware still works, though it is highly recommended to transition to the new wiring method.

Refer to the [integration guide](../01-ibc/04-middleware/03-integration.md) to understand how to use this new middleware to improve middleware wiring in the chain application setup.

# Migrations for Application Developers

In order to be wired with the new StackBuilder primitive, applications and middlewares must implement new methods as part of their respective interfaces.

IBC Applications must implement a new `SetICS4Wrapper` which will set the `ICS4Wrapper` through which the application will call `SendPacket` and `WriteAcknowledgement`. It is recommended that IBC applications are initialized first with the IBC ChannelKeeper directly, and then modified with a middleware ICS4Wrapper during the stack wiring. 

```go
// SetICS4Wrapper sets the ICS4Wrapper. This function may be used after
// the module's initialization to set the middleware which is above this
// module in the IBC application stack.
// The ICS4Wrapper **must** be used for sending packets and writing acknowledgements
// to ensure that the middleware can intercept and process these calls.
// Do not use the channel keeper directly to send packets or write acknowledgements
// as this will bypass the middleware.
SetICS4Wrapper(wrapper ICS4Wrapper)
```

Many applications have a stateful keeper that executes the logic for sending packets and writing acknowledgements. In this case, the keeper in the application must be a **pointer** reference so that it can be modified in place after initialization.

The initialization should be modified to no longer take in an addition `ics4Wrapper` as this gets modified later by `SetICS4Wrapper`. The constructor function must also return a **pointer** reference so that it may be modified in-place by the stack builder.

Below is an example IBCModule that supports the stack builder wiring.

E.g.

```go
type IBCModule struct {
	keeper *keeper.Keeper
}

// NewIBCModule creates a new IBCModule given the keeper
func NewIBCModule(k *keeper.Keeper) *IBCModule {
	return &IBCModule{
		keeper: k,
	}
}

// SetICS4Wrapper sets the ICS4Wrapper. This function may be used after
// the module's initialization to set the middleware which is above this
// module in the IBC application stack.
func (im IBCModule) SetICS4Wrapper(wrapper porttypes.ICS4Wrapper) {
	if wrapper == nil {
		panic("ICS4Wrapper cannot be nil")
	}

	im.keeper.WithICS4Wrapper(wrapper)
}

/// Keeper file that has ICS4Wrapper internal to its own struct

// Keeper defines the IBC fungible transfer keeper
type Keeper struct {
	...
	ics4Wrapper   porttypes.ICS4Wrapper

    // Keeper is initialized with ICS4Wrapper
    // being equal to the top-level channelKeeper
    // this can be changed by calling WithICS4Wrapper
    // with a different middleware ICS4Wrapper
	channelKeeper types.ChannelKeeper
	...
}

// WithICS4Wrapper sets the ICS4Wrapper. This function may be used after
// the keepers creation to set the middleware which is above this module
// in the IBC application stack.
func (k *Keeper) WithICS4Wrapper(wrapper porttypes.ICS4Wrapper) {
	k.ics4Wrapper = wrapper
}
```

# Migration for Middleware Developers

Since Middleware is itself implement the IBC application interface, it must also implement `SetICS4Wrapper` in the same way as IBC applications.

Additionally, IBC Middleware has an underlying IBC application that it calls into as well. Previously this application would be set in the middleware upon construction. With the stack builder primitive, the application is only set during upon calling `stack.Build()`. Thus, middleware is additionally responsible for implementing the new method: `SetUnderlyingApplication`:

```go
// SetUnderlyingModule sets the underlying IBC module. This function may be used after
// the middleware's initialization to set the ibc module which is below this middleware.
SetUnderlyingApplication(IBCModule)
```

The initialization should not include the ICS4Wrapper and application as this gets set later. The constructor function for Middlewares **must** be modified to return a **pointer** reference so that it can be modified in place by the stack builder.

Below is an example middleware setup:

```go
// IBCMiddleware implements the ICS26 callbacks
type IBCMiddleware struct {
	app         porttypes.PacketUnmarshalerModule
	ics4Wrapper porttypes.ICS4Wrapper

    // this is a stateful middleware with its own internal keeper
	mwKeeper *keeper.MiddlewareKeeper

	// this is a middleware specific field
	mwField any
}

// NewIBCMiddleware creates a new IBCMiddleware given the keeper and underlying application.
// NOTE: It **must** return a pointer reference so it can be
// modified in place by the stack builder
// NOTE: We do not pass in the underlying app and ICS4Wrapper here as this happens later
func NewIBCMiddleware(
	mwKeeper *keeper.MiddlewareKeeper, mwField any,
) *IBCMiddleware {
    return &IBCMiddleware{
        mwKeeper: mwKeeper,
        mwField, mwField,
    }
}

// SetICS4Wrapper sets the ICS4Wrapper. This function may be used after the
// middleware's creation to set the middleware which is above this module in
// the IBC application stack.
func (im *IBCMiddleware) SetICS4Wrapper(wrapper porttypes.ICS4Wrapper) {
	if wrapper == nil {
		panic("ICS4Wrapper cannot be nil")
	}
	im.mwKeeper.WithICS4Wrapper(wrapper)
}

// SetUnderlyingApplication sets the underlying IBC module. This function may be used after
// the middleware's creation to set the ibc module which is below this middleware.
func (im *IBCMiddleware) SetUnderlyingApplication(app porttypes.IBCModule) {
	if app == nil {
		panic(errors.New("underlying application cannot be nil"))
	}
	if im.app != nil {
		panic(errors.New("underlying application already set"))
	}
	im.app = app
}
```
