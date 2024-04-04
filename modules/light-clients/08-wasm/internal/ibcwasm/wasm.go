package ibcwasm

import (
	"errors"
)

var (
	queryRouter  QueryRouter
	queryPlugins QueryPluginsI
)

// SetQueryRouter sets the custom wasm query router for the 08-wasm module.
// Panics if the queryRouter is nil.
func SetQueryRouter(router QueryRouter) {
	if router == nil {
		panic(errors.New("query router must not be nil"))
	}
	queryRouter = router
}

// GetQueryRouter returns the custom wasm query router for the 08-wasm module.
func GetQueryRouter() QueryRouter {
	return queryRouter
}

// SetQueryPlugins sets the current query plugins
func SetQueryPlugins(plugins QueryPluginsI) {
	if plugins == nil {
		panic(errors.New("query plugins must not be nil"))
	}
	queryPlugins = plugins
}

// GetQueryPlugins returns the current query plugins
func GetQueryPlugins() QueryPluginsI {
	return queryPlugins
}
