# Token Factory Module

The Token Factory module enables permissionless creation, minting, and burning of tokens on the sandbox.
Tokens are address namespaced, thus the creation is permissionless.

## Overview

This module provides the following key features:

- **Permissionless Creation**: Anyone can create a new token denomination
- **Creator Authority**: Token creators automatically become admins of their tokens
- **Admin-Only Operations**: Only token admins can mint and burn their tokens
- **Namespace Protection**: Tokens are namespaced by creator address to prevent conflicts

## Key Concepts

### Token Denomination Format

Tokens created through this module follow the format:

- 1-20 characters long
- Only alphanumerical

Example: `mydenom`

### Authority Model

- When you create a token, you become its **admin**
- Only the admin can mint and burn tokens
- Each token has independent admin control
- No global authority or governance control

## Messages

### MsgCreateDenom

Creates a new token denomination.

```protobuf
message MsgCreateDenom {
  string sender = 1;
  string denom = 2;
}
```

### MsgMint

Mints tokens to a specified address.

```protobuf
message MsgMint {
  string from = 1;
  string address = 2;
  cosmos.base.v1beta1.Coin amount = 3;
}
```

### MsgBurn

Burns tokens from the admin's balance.

```protobuf
message MsgBurn {
  string from = 1;
  cosmos.base.v1beta1.Coin amount = 2;
}
```

### MsgCreateBridge

Creates or overwrites a bridge mapping for a tokenfactory denom and IBC client ID, and stores the derived ICA address.

```protobuf
message MsgCreateBridge {
  string from = 1;
  string denom = 2;
  string client_id = 3;
  string remote_contract_address = 4;
}
```

## Queries

### DenomAuthorityMetadata

Query the authority metadata for a specific denom:

```bash
sandboxd query tokenfactory denom-authority-metadata uwfdeposit
```

### DenomsByCreator

Query all denoms created by a specific creator:

```bash
sandboxd query tokenfactory denoms-by-creator wf1abc123...
```

### Params

Query module parameters:

```bash
sandboxd query tokenfactory params
```

### Bridge

Query a single bridge by denom and client-id:

```bash
sandboxd query tokenfactory bridge uwfdeposit 07-tendermint-0
```

### DenomBridges

List bridges for a given denom:

```bash
sandboxd query tokenfactory denom-bridges uwfdeposit
```

### Bridges

List all bridges:

```bash
sandboxd query tokenfactory bridges
```

## Usage Examples

### Creating a Token

```bash
sandboxd tx tokenfactory create-denom uwfdeposit \
  --from alice \
  --chain-id sandbox-1 \
  --gas auto \
  --home path_to_home
```

### Minting Tokens

```bash
sandboxd tx tokenfactory mint wf1recipient... 1000000uwfdeposit \
  --from alice \
  --chain-id sandbox-1 \
  --gas auto \
  --home path_to_home
```

### Burning Tokens

```bash
sandboxd tx tokenfactory burn 500000uwfdeposit \
  --from alice \
  --chain-id sandbox-1 \
  --gas auto \
  --home path_to_home
```

### Creating a Bridge

```bash
sandboxd tx tokenfactory create-bridge uwfdeposit 07-tendermint-0 0x1111111111111111111111111111111111111111 \
  --from alice \
  --chain-id sandbox-1 \
  --gas auto \
  --home path_to_home
```

## Module Parameters

This module currently has no configurable parameters.

## Integration

The module is designed to work seamlessly with:

- Cosmos SDK Bank module for token transfers
- Standard wallet interfaces
- Bridge mapping support for cross-chain integrations (set via create-bridge; query via bridge/denom-bridges/bridges)

## Security Features

- **Authority Validation**: All mint/burn operations verify admin permissions
- **Namespace Protection**: Creator address prevents naming collisions
- **Input Validation**: Comprehensive validation of all message parameters
- **State Consistency**: Atomic operations ensure consistent state

## Events

The module emits the following events on successful operations:

- tokenfactory_create_denom
  - denom
  - admin: creator address
- tokenfactory_mint
  - denom, amount, admin, to
- tokenfactory_burn
  - denom, amount, admin
- tokenfactory_create_bridge
  - denom, admin, client_id, remote_contract_address, ica_address
