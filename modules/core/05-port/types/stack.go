package types

type IBCStackBuilder struct {
	middlewares   []Middleware
	baseModule    IBCModule
	channelKeeper ICS4Wrapper
}

func NewIBCStackBuilder(chanKeeper ICS4Wrapper) *IBCStackBuilder {
	return &IBCStackBuilder{
		channelKeeper: chanKeeper,
	}
}

func (b *IBCStackBuilder) Next(middleware Middleware) *IBCStackBuilder {
	b.middlewares = append(b.middlewares, middleware)
	return b
}

func (b *IBCStackBuilder) Base(baseModule IBCModule) *IBCStackBuilder {
	if baseModule == nil {
		panic("base module cannot be nil")
	}
	if b.baseModule != nil {
		panic("base module already set")
	}
	b.baseModule = baseModule
	return b
}

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
