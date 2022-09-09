package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterInterfaces registers the interchain accounts controller message types using the provided InterfaceRegistry
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
<<<<<<< HEAD
		&MsgRegisterAccount{},
		&MsgSubmitTx{},
=======
		&MsgRegisterInterchainAccount{},
		&MsgSendTx{},
>>>>>>> a4be561 (chore: rename `SubmitTx` to `SendTx` (#2255))
	)
}
