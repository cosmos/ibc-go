---
title: Open transfer channel
sidebar_label: Open transfer channel
sidebar_position: 4
slug: /channel-upgrades/open-channel
---

# Open an ICS 20 transfer channel

The relayer needs to submit transactions on both blockchains, so we run the following command to add the keys for the accounts on `chain1` and `chain2` that the relayer can use to submit transactions:

```bash
gm hermes keys
```

The relayer also needs a [configuration file](https://github.com/informalsystems/hermes/blob/master/config.toml). In this tutorial we will have the configuration file in the same folder as the relayer binary and specify it using the `--config` flag in each command.

You can generate a default configuration by running:

```bash
gm hermes config
```

This tutorial has been completed with the following configuration file:

```yaml
[global]
log_level = 'trace'

[telemetry]
enabled = true
host = '127.0.0.1'
port = 3001

# Specify the mode to be used by the relayer. [Required]
[mode]

# Specify the client mode.
[mode.clients]

# Whether or not to enable the client workers. [Required]
enabled = true

# Whether or not to enable periodic refresh of clients. [Default: true]
# Note: Even if this is disabled, clients will be refreshed automatically if
#      there is activity on a connection or channel they are involved with.
refresh = true

# Whether or not to enable misbehaviour detection for clients. [Default: false]
misbehaviour = true

# Specify the connections mode.
[mode.connections]

# Whether or not to enable the connection workers for handshake completion. [Required]
enabled = true

[mode.channels]
enabled = true

# Specify the packets mode.
[mode.packets]

# Whether or not to enable the packet workers. [Required]
enabled = true

clear_interval = 1

[[chains]]
id = 'chain1'
type = 'CosmosSdk'
rpc_addr = 'http://localhost:27000'
grpc_addr = 'http://localhost:27002'
event_source = { mode = 'push', url = 'ws://127.0.0.1:27000/websocket', batch_delay = '500ms' }
rpc_timeout = '15s'
account_prefix = 'cosmos'
key_name = 'wallet'
store_prefix = 'ibc'
gas_price = { price = 0.001, denom = 'stake' }
max_gas = 1000000
clock_drift = '5s'
trusting_period = '14days'
trust_threshold = { numerator = '1', denominator = '3' }

[[chains]]
id = 'chain2'
type = 'CosmosSdk'
rpc_addr = 'http://localhost:27010'
grpc_addr = 'http://localhost:27012'
event_source = { mode = 'push', url = 'ws://127.0.0.1:27010/websocket', batch_delay = '500ms' }
rpc_timeout = '15s'
account_prefix = 'cosmos'
key_name = 'wallet'
store_prefix = 'ibc'
gas_price = { price = 0.001, denom = 'stake' }
max_gas = 1000000
clock_drift = '5s'
trusting_period = '14days'
trust_threshold = { numerator = '1', denominator = '3' }
```

With both blockchains running, we can run the following command in hermes to establish a connection and an ICS 20 transfer channel:

```bash
hermes --config config.toml create channel --a-chain chain1 \
--b-chain chain2 \
--a-port transfer \
--b-port transfer \
--new-client-connection
```

When both the connection and channel handshakes complete, the output on the console looks like this:

```bash
SUCCESS Channel {
  ordering: Unordered,
  a_side: ChannelSide {
    chain: BaseChainHandle {
      chain_id: ChainId {
        id: "chain1",
        version: 0,
      },
      runtime_sender: Sender { .. },
    },
    client_id: ClientId(
      "07-tendermint-0",
    ),
    connection_id: ConnectionId(
      "connection-0",
    ),
    port_id: PortId(
      "transfer",
    ),
    channel_id: Some(
      ChannelId(
        "channel-0",
      ),
    ),
    version: None,
  },
  b_side: ChannelSide {
    chain: BaseChainHandle {
      chain_id: ChainId {
        id: "chain2",
        version: 0,
      },
      runtime_sender: Sender { .. },
    },
    client_id: ClientId(
      "07-tendermint-0",
    ),
    connection_id: ConnectionId(
      "connection-0",
    ),
    port_id: PortId(
      "transfer",
    ),
    channel_id: Some(
      ChannelId(
        "channel-0",
      ),
    ),
    version: None,
  },
  connection_delay: 0ns,
}
```
