---
title: Events
sidebar_label: Events
sidebar_position: 6
slug: /ibc/light-clients/wasm/events
---

# Events

The `08-wasm` module emits the following events:

## `MsgStoreCode`

| Type            | Attribute Key  | Attribute Value        |
|-----------------|----------------|------------------------|
| store_wasm_code | wasm_code_hash | {hex.Encode(codeHash)} |
| message         | module         | ibc_client             |
