---
title: Events
sidebar_label: Events
sidebar_position: 6
slug: /ibc/light-clients/wasm/events
---

# Events

The `08-wasm` module emits the following events:

## `MsgStoreCode`

| Type             | Attribute Key  | Attribute Value        |
|------------------|----------------|------------------------|
| store_wasm_code  | wasm_code_hash | {hex.Encode(codeHash)} |
| message          | module         | 08-wasm                |

## `MsgMigrateContract`

| Type             | Attribute Key  | Attribute Value           |
|------------------|----------------|---------------------------|
| migrate_contract | client_id      | {clientId}                |
| migrate_contract | wasm_code_hash | {hex.Encode(codeHash)}    |
| migrate_contract | new_code_hash  | {hex.Encode(newCodeHash)} |
| message          | module         | 08-wasm                   |
