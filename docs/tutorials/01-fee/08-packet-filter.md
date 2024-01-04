---
title: Hermes Packet Filtering by Fee
sidebar_label: Hermes Packet Filtering by Fee (Optional)
sidebar_position: 8
slug: /fee/packet-filter
---

# Hermes Packet Filtering by Fee

Hermes provides a way for relayers to only relay packets that have a fee greater than a certain amount.
In this section, we will configure this option to only relay packets that have a fee greater than 30 `token`.
Currently, Hermes only supports filtering by the receive packet fee.
Filtering by acknowledgement packet fee or timeout packet fee is not supported.

## Configure Hermes

To configure Hermes to filter packets by fee, we need to add the following to the `config.toml` file:

```toml
[chains.packet_filter]
policy = 'allow'
list = [
  ['ica*', '*'],
  ['transfer', '*'],
]

[chains.packet_filter.min_fees.'*']
recv = [ { amount = 30, denom = 'token' } ]
```

Here is a full example of the `config.toml` file:

```toml reference title="hermes/filtered_config.toml"
https://github.com/srdtrk/cosmoverse2023-ibc-fee-demo/blob/1ddac03efdf6d403126c3f5ad067fd708e2e410a/hermes/filtered_config.toml
```

You can copy this using the following command:

```bash
cp hermes/filtered_config.toml ~/.hermes/config.toml
```

## Test the Application

To test the application, we launch the chains and the relayer as we did in the previous sections.
This requires four terminals, run the following commands in each of the first three:

```bash title="Terminal 1"
ignite chain serve -c earth.yml --reset-once
```

```bash title="Terminal 2"
ignite chain serve -c mars.yml --reset-once
```

```bash title="Terminal 3"
cd react
npm run dev
```

The last terminal will be used to run the relayer. First, we will create the client, connection, and channel between the two chains by running:

```bash title="Terminal 4"
hermes create channel --channel-version '{"fee_version":"ics29-1","app_version":"ics20-1"}' \
--a-chain earth --b-chain mars \
--a-port transfer --b-port transfer \
--new-client-connection --yes
```

Once the operation above is completed, we can run the relayer with the following command:

```bash title="Terminal 4"
hermes start
```

To test the application, you can now send packets from the Earth chain to the Mars chain.
If the fee is less than 30 `token`, the relayer will not relay the packet, and it will eventually timeout.
Note that due to our frontend implementation, the amount of fee that needs to be entered to the frontend is at least 60 `token`.
Don't forget to quit all the processes after the test is done.
