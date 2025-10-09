---
title: Address Codec
sidebar_label: Address Codec
sidebar_position: 7
slug: /ibc/apps/address-codec
---

# Custom Address Codec

## Overview

Starting in ibc-go `v10.4.0`, the IBC transfer module uses the application's configured address codec to parse sender and receiver addresses. This enables chains to accept multiple address formats in IBC packets—for example, both standard Cosmos bech32 addresses (`cosmos1...`) and Ethereum hex addresses (`0x...`).

## Interface

The Cosmos SDK defines a simple interface for converting between address representations:

```go
type Codec interface {
  StringToBytes(text string) ([]byte, error)
  BytesToString(bz []byte) (string, error)
}
```

Applications configure a codec implementation on the `AccountKeeper`. The IBC transfer module retrieves this codec via `accountKeeper.AddressCodec()` and uses it throughout packet processing—validating sender addresses when creating packets and parsing receiver addresses when delivering funds.

**Chain independence:** Each chain applies its own codec independently. The sending chain validates senders with its codec, the receiving chain validates receivers with its codec. This works seamlessly across chains with different codec configurations without any protocol changes.

## Implementation

A typical implementation composes the SDK's standard bech32 codec and extends it to parse hex addresses:

```go
type EvmCodec struct {
	bech32Codec address.Codec
}

func (c *EvmCodec) StringToBytes(text string) ([]byte, error) {
	if strings.HasPrefix(text, "0x") {
		// Validate and parse hex address using go-ethereum/common
		if !common.IsHexAddress(text) {
			return nil, errors.New("invalid hex address")
		}
		addr := common.HexToAddress(text)
		return addr.Bytes(), nil
	}
	// Default to bech32 parsing
	return c.bech32Codec.StringToBytes(text)
}

func (c *EvmCodec) BytesToString(bz []byte) (string, error) {
	// Always return bech32 format
	return c.bech32Codec.BytesToString(bz)
}
```

This pattern accepts both address formats as input while consistently outputting bech32. This makes the codec a drop-in replacement for the standard codec—existing tooling continues to work unchanged while users gain the ability to specify hex addresses where convenient.

### Application Wiring

Configure the codec when initializing the AccountKeeper:

```go
addressCodec := utils.NewEvmCodec(bech32Prefix)

app.AccountKeeper = authkeeper.NewAccountKeeper(
	appCodec,
	storeService,
	authtypes.ProtoBaseAccount,
	maccPerms,
	addressCodec,  // Custom codec
	bech32Prefix,
	authority,
)
```

The IBC transfer keeper requires no modification. It automatically uses the codec from `AccountKeeper` through the expected keeper interfaces.

## Usage

Once configured, the chain accepts IBC transfers with receiver addresses in either format:

```bash
# Standard bech32 address
gaiad tx ibc-transfer transfer transfer channel-0 \
cosmos1p9p6h9m8jcn8f7l6h3k2wq9g6yx0l8a9u2n4lr 1000uatom --from sender

# Ethereum hex address
gaiad tx ibc-transfer transfer transfer channel-0 \
0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb 1000uatom --from sender
```

Both formats resolve to the same on-chain account when derived from the same private key. The codec handles conversion to the internal byte representation transparently.

## Reference Implementation

The cosmos/evm repository provides a complete implementation in `utils/address_codec.go` with integration examples in the `evmd` reference chain:

- [**Implementation PR**](https://github.com/cosmos/evm/pull/665)
- [**Example integration**](https://github.com/cosmos/evm/tree/main/evmd)
