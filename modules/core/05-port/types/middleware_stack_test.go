package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v10/testing/mock"
)

func TestMiddlewareStackQuery(t *testing.T) {
	// Test ModuleName method on individual modules
	// Note: We can't easily create real keepers in unit tests without full app setup
	// So we'll test the ModuleName functionality with a simpler approach

	// Create router and test basic functionality
	router := porttypes.NewRouter()

	// Create a mock module for testing
	appModule := &mock.AppModule{}
	ibcApp := &mock.IBCApp{}
	mockModule := mock.NewIBCModule(appModule, ibcApp)

	// Test ModuleName method
	require.Equal(t, "mock", mockModule.ModuleName())

	// Add to router
	router.AddRoute("test", mockModule)

	// Test middleware stack query
	middlewareStack := router.GetMiddlewareStack("test")
	require.NotNil(t, middlewareStack)
	require.Len(t, middlewareStack, 1) // Currently only returns top-level module
	require.Equal(t, "mock", middlewareStack[0])
}

func TestRouterMiddlewareStack(t *testing.T) {
	router := porttypes.NewRouter()

	// Test with non-existent port
	stack := router.GetMiddlewareStack("nonexistent")
	require.Nil(t, stack)

	// Test with mock module
	appModule := &mock.AppModule{}
	ibcApp := &mock.IBCApp{}
	mockModule := mock.NewIBCModule(appModule, ibcApp)
	router.AddRoute("simple", mockModule)

	stack = router.GetMiddlewareStack("simple")
	require.NotNil(t, stack)
	require.Len(t, stack, 1)
	require.Equal(t, "mock", stack[0])
}
