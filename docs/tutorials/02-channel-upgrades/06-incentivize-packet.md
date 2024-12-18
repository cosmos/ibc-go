---
title: Incentivize packet
sidebar_label: Incentivize packet
sidebar_position: 6
slug: /channel-upgrades/incentivize-packet
---

# Incentivize an ICS 20 transfer packet

## Register the counterparty payee

All incentivization fees are paid to accounts on the chain from where the IBC packets originate. To ensure that the relayer that delivers the `MsgRecvPacket` on the destination chain is correctly compensated, the counterparty payee address (i.e. the account address of the relayer on the source chain) needs to be registered on the destination chain. Throughout this tutorial the source chain is `chain1` and the destination chain is `chain2`, therefore we need to register the account address of the relayer on chain `chain1` (`RLY_CHAIN1`) on chain `chain2` (where the relayer has the account address `RLY_CHAIN2`):

```bash
simd tx ibc-fee register-counterparty-payee transfer channel-0 $RLY_CHAIN2 $RLY_CHAIN1 \
--from $RLY_CHAIN2 \
--chain-id chain2 \
--keyring-backend test \
--home ../../gm/chain2 \
--node http://localhost:27010
```

Once the above command succeeds, then we can verify which counterparty payee is registered on chain `chain2` for account `RLY_CHAIN2`:

```bash
simd q ibc-fee counterparty-payee channel-0 $RLY_CHAIN2 --node http://localhost:27010
```

```yaml
counterparty_payee: cosmos1vdy5fp0jy2l2ees870a7mls357v7uad6ufzcyz
```

We see that the counterparty payee address matches what we expected (i.e. the `RLY_CHAIN1` address). In this tutorial we are going to send only one packet from chain `chain1` to chain `chain2`, so we only need to register the counterparty payee on chain `chain2`. In real life circumstances relayers relay packets on both directions (i.e. from chain `chain1` to `chain2` and also vice-versa), and thus relayers should register as well on chain `chain1` the counterparty payee address to be compensated for delivering the `MsgRecvPacket` on chain `chain1`.

## Multi-message transaction with single `MsgPayPacketFee` message

We first generate (not execute) an IBC transfer transaction (again `1000samoleans` from `VALIDATOR_CHAIN1` to `VALIDATOR_CHAIN2`):

```bash
simd tx ibc-transfer transfer transfer channel-0 $VALIDATOR_CHAIN2 1000samoleans \
--from $VALIDATOR_CHAIN1 \
--chain-id chain1 \
--keyring-backend test \
--home ../../gm/chain1 \
--node tcp://localhost:27000 \
--generate-only > transfer.json
```

Then we prepend a `MsgPayPacketFee`, sign the transaction and broadcast it. Please note that `jq` is used to manipulate the transaction JSON file, making it a multi-message transaction. In practice, this multi-message transaction would be built using a gRPC or web client, for example, a web-based wallet application could fulfill this role. Note also that the `signer` field uses the address of `VALIDATOR_CHAIN1`.

`
jq '.body.messages |= [{"@type":"/ibc.applications.fee.v1.MsgPayPacketFee","fee": {"recv_fee": [{"denom": "samoleans", "amount": "50"}], "ack_fee": [{"denom": "samoleans", "amount": "25"}], "timeout_fee": [{"denom": "samoleans", "amount": "10"}]}, "source_port_id": "transfer", "source_channel_id": "channel-0", "signer": "cosmos18phmkrpnn6gmpzscf6hnf5zpv06sygxc6f2v92" }] + .' transfer.json > incentivized_transfer.json
`

```bash
simd tx sign incentivized_transfer.json \
--from $VALIDATOR_CHAIN1 \
--chain-id chain1 \
--keyring-backend test \
--home ../../gm/chain1 \
--node tcp://localhost:27000 > signed.json
```

```bash
simd tx broadcast signed.json \
--home ../../gm/chain1 \
--node tcp://localhost:27000
```

We wait for the relayer to relay the packet, and then we query the balance of account `VALIDATOR_CHAIN2` on chain `chain2` and see that it has indeed received an equivalent amount of vouchers for the `1000samoleans` sent by `VALIDATOR_CHAIN1`:

```yaml
simd q bank balances $VALIDATOR_CHAIN2 --node http://localhost:27010
```

```bash
balances:
- amount: "1000"
  denom: ibc/27A6394C3F9FF9C9DCF5DFFADF9BB5FE9A37C7E92B006199894CF1824DF9AC7C
- amount: "100000000"
  denom: samoleans
- amount: "99000000"
  denom: stake
pagination:
  total: "3"
```

We check as well the balance for account `VALIDATOR_CHAIN1` on chain `chain1`:

```bash
simd q bank balances $WALLET_1 --node http://localhost:16657
```

```yaml
./simd q bank balances $VALIDATOR_CHAIN1 --node http://localhost:27000
balances:
- amount: "99998925"
  denom: samoleans
- amount: "99000000"
  denom: stake
pagination:
  total: "2"
```

An amount of `1075samoleans` has been deducted, which is what we expected: `1000samoleans` have been transferred to `VALIDATOR_CHAIN2` and `75stake` have been paid for the receive and acknowledgment fees. The timeout fee has been refunded to `VALIDATOR_CHAIN1` and the relayer address `RLY_CHAIN1` should have gained `75samoleans` for submitting the `MsgRecvPacket` and the `MsgAcknowledgement` messages.
