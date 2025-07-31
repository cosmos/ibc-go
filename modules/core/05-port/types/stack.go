package types

// IBCStackBuilder is a builder for creating an IBC application stack.
// It allows for the creation of an IBC stack with with a set of middlewares on top of a base IBC application.
// The stack is built by adding, from bottom to top, the IBC application and all subsequent middlewares to the stack.
// For instance, to create a stack like the following:
// * RecvPacket: IBC core -> RateLimit -> PFM -> Callbacks -> Transfer (AddRoute)
// * SendPacket: Transfer -> Callbacks -> PFM -> RateLimit -> IBC core (ICS4Wrapper)
// porttypes.NewIBCStackBuilder(app.IBCKeeper.ChannelKeeper).
// .Base(app.TransferKeeper).
// .Next(callbacks.NewIBCMiddleware(..)).
// .Next(packetforward.NewIBCMiddleware(..)).
// .Next(ratelimiting.NewIBCMiddleware(..)).
// .Build()
//
// ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferStack.Build())
type IBCStackBuilder struct {
	middlewares   []StackMiddleware
	baseModule    StackIBCModule
	channelKeeper ICS4Wrapper
}

// StackIBCModule is an interface for IBC applications and middlewares to support
// the IBCStackBuilder.
type StackIBCModule interface {
	IBCModule

	// SetICS4Wrapper sets the ICS4Wrapper. This function may be used after
	// the module's initialization to set the middleware which is above this
	// module in the IBC application stack.
	// The ICS4Wrapper **must** be used for sending packets and writing acknowledgements
	// to ensure that the middleware can intercept and process these calls.
	// Do not use the channel keeper directly to send packets or write acknowledgements
	// as this will bypass the middleware.
	SetICS4Wrapper(wrapper ICS4Wrapper)
}

// StackMiddleware is an interface for middlewares to support the IBCStackBuilder.
type StackMiddleware interface {
	Middleware
	StackIBCModule

	// SetUnderlyingModule sets the underlying IBC module. This function may be used after
	// the middleware's initialization to set the ibc module which is below this middleware.
	SetUnderlyingApplication(IBCModule)
}

// NewIBCStackBuilder creates a new IBCStackBuilder
func NewIBCStackBuilder(chanKeeper ICS4Wrapper) *IBCStackBuilder {
	return &IBCStackBuilder{
		channelKeeper: chanKeeper,
	}
}

// Next adds a middleware to the stack, bottom to top.
func (b *IBCStackBuilder) Next(middleware StackMiddleware) *IBCStackBuilder {
	b.middlewares = append(b.middlewares, middleware)
	return b
}

// Base sets the base IBC module for the stack.
func (b *IBCStackBuilder) Base(baseModule StackIBCModule) *IBCStackBuilder {
	if baseModule == nil {
		panic("base module cannot be nil")
	}
	if b.baseModule != nil {
		panic("base module already set")
	}
	b.baseModule = baseModule
	return b
}

// Build creates the IBC stack in the order of the middlewares, from bottom to top.
// The stack is returned as an IBCModule.
func (b *IBCStackBuilder) Build() IBCModule {
	if b.baseModule == nil {
		panic("base module cannot be nil")
	}
	if len(b.middlewares) == 0 {
		panic("middlewares cannot be empty")
	}
	if b.channelKeeper == nil {
		panic("channel keeper cannot be nil")
	}

	// Build the stack by moving up the middleware list
	// and setting the underlying application for each middleware
	// and the ICS4wrapper for the underlying module.
	underlyingModule := b.baseModule
	for i := range len(b.middlewares) {
		b.middlewares[i].SetUnderlyingApplication(underlyingModule)
		underlyingModule.SetICS4Wrapper(b.middlewares[i])
		underlyingModule = b.middlewares[i]
	}

	// set the top level channel keeper as the ICS4Wrapper
	// for the lop level middleware
	b.middlewares[len(b.middlewares)-1].SetICS4Wrapper(b.channelKeeper)

	return b.middlewares[len(b.middlewares)-1]
}
