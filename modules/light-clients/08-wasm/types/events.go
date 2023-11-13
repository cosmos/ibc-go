package types

// IBC 08-wasm events
const (
	// EventTypeStoreWasmCode defines the event type for bytecode storage
	EventTypeStoreWasmCode = "store_wasm_code"
	// EventTypeMigrateContract defines the event type for a contract migration
	EventTypeMigrateContract = "migrate_contract"

	// AttributeKeyWasmCodeHash denotes the code hash of the wasm code that was stored or migrated
	AttributeKeyWasmCodeHash = "wasm_code_hash"
	// AttributeKeyClientID denotes the client identifier of the wasm client
	AttributeKeyClientID = "client_id"
	// AttributeKeyNewCodeHash denotes the code hash of the new wasm code.
	AttributeKeyNewCodeHash = "new_code_hash"

	AttributeValueCategory = ModuleName
)
