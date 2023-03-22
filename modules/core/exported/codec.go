package exported

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

// RegisterInterfaces registers the CallbackPacketDataI interface.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterInterface(
		"ibc.core.exported.v1.CallbackPacketDataI",
		(*CallbackPacketDataI)(nil),
	)
}
