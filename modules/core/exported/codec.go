package exported

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterInterface(
		"ibc.core.exported.v1.CallbackPacketDataI",
		(*CallbackPacketDataI)(nil),
	)
}
