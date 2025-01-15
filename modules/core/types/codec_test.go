package types

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	gogoprotoany "github.com/cosmos/gogoproto/types/any"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	"github.com/stretchr/testify/require"
)

func TestInterfaceRegistrationOfLocalhost(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	RegisterInterfaces(registry)
	val := &gogoprotoany.Any{
		TypeUrl: "/ibc.lightclients.localhost.v2.ClientState",
		Value:   []byte{},
	}
	require.NoError(t, registry.UnpackAny(val, new(exported.ClientState)))
}
