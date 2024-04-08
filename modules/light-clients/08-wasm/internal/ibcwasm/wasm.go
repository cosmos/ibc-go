package ibcwasm

import (
	"errors"
)

var queryPlugins QueryPluginsI

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
