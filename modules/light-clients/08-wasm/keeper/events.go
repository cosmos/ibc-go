package keeper

import (
	"encoding/hex"
	"errors"

	"cosmossdk.io/core/event"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

// emitStoreWasmCodeEvent emits a store wasm code event
func emitStoreWasmCodeEvent(em event.Manager, checksum types.Checksum) error {
	return errors.Join(
		em.EmitKV(
			types.EventTypeStoreWasmCode,
			event.NewAttribute(types.AttributeKeyWasmChecksum, hex.EncodeToString(checksum)),
		),
		em.EmitKV(
			sdk.EventTypeMessage,
			event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	)
}

// emitMigrateContractEvent emits a migrate contract event
func emitMigrateContractEvent(em event.Manager, clientID string, checksum, newChecksum types.Checksum) error {
	return errors.Join(
		em.EmitKV(
			types.EventTypeMigrateContract,
			event.NewAttribute(types.AttributeKeyClientID, clientID),
			event.NewAttribute(types.AttributeKeyWasmChecksum, hex.EncodeToString(checksum)),
			event.NewAttribute(types.AttributeKeyNewChecksum, hex.EncodeToString(newChecksum)),
		),
		em.EmitKV(
			sdk.EventTypeMessage,
			event.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	)
}
