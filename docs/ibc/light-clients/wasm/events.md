<!--
order: 6
-->

# Events

The `08-wasm` module emits the following events:

## `MsgStoreCode`

| Type            | Attribute Key  | Attribute Value        |
|-----------------|----------------|------------------------|
| store_wasm_code | wasm_code_hash | {hex.Encode(codeHash)} |
| message         | module         | ibc_client             |
