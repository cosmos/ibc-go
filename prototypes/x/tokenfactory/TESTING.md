# TokenFactory manual testing guide (single-node)

This guide walks you through manually testing the x/tokenfactory module on a local single-node sandbox using the provided scripts. It covers spinning up the chain, ensuring a wallet is available, and exercising all available tokenfactory CLI tx and query commands in a logical order, with bank balance checks after mint and burn.

References
- Single-node scripts: scripts/single-node.sh, scripts/kill-single-node.sh
- Project quickstart and preloaded accounts: readme.md
- TokenFactory overview and CLI synopsis: x/tokenfactory/README.md


## 1) Prerequisites

- Go 1.25 installed
- jq 
- Built sandboxd (`make build`)


## 2) Start a local single-node

From the repo root:
- make single-node

This runs scripts/single-node.sh, which:
- Initializes a fresh home at scripts/node-data/sandbox-1/n0 (unless continuing an existing state)
- Imports a validator key and a set of named demo keys with funds
- Enables API, gRPC, and Swagger
- Starts the node in the background and prints the home path and a log file location

To stop later:
- make kill-single-node


## 3) Environment variables (recommended)

Export convenient variables for the session. Adjust paths if you run binaries or home elsewhere.

Set core paths and endpoints:
- export BIN=${BIN:-./build/sandboxd}
- export NODE=${NODE:-tcp://localhost:26657}
- export API=${API:-http://localhost:1317}
- export GRPC=${GRPC:-localhost:9090}
- export NODE_HOME=${NODE_HOME:-$(pwd)/scripts/node-data/sandbox-1/n0}
- export KEYRING=${KEYRING:-test}
- export CHAIN_ID=$($BIN status --node $NODE 2>/dev/null | jq -r .NodeInfo.network)
- export BASE_DENOM=$($BIN query staking params --node $NODE -o json | jq -r .params.bond_denom)
- export TX_FLAGS="--node $NODE --chain-id $CHAIN_ID --home $NODE_HOME --keyring-backend $KEYRING --gas auto --gas-adjustment 1.4 --gas-prices 0.05uwfgas -y"

## 4) Keys: ensure you have a funded wallet

List keys in the node’s keyring:
- $BIN keys list --home $NODE_HOME --keyring-backend $KEYRING

The single-node setup imports many pre-funded demo keys (see readme.md). You can use any of them. For example, pick demo-user as the token admin:
- export ADMIN_KEY=demo-user
- export ADMIN_ADDR=$($BIN keys show $ADMIN_KEY --home $NODE_HOME --keyring-backend $KEYRING -a)

Verify it has funds for fees:
- $BIN query bank balances $ADMIN_ADDR --node $NODE -o json | jq

Also choose a recipient account for mint tests (another preloaded key works well):
- export USER_KEY=demo-relayer
- export USER_ADDR=$($BIN keys show $USER_KEY --home $NODE_HOME --keyring-backend $KEYRING -a)


## 5) TokenFactory: queries overview

List available query subcommands:
- $BIN query tokenfactory --help

Expected queries:
- params
- denoms-by-creator [creator-address]
- denom-authority-metadata [denom]
- bridge [denom] [client-id]
- denom-bridges [denom]
- bridges

Useful to run up front:
- $BIN query tokenfactory params --node $NODE -o json | jq


## 6) TokenFactory: tx overview

List available tx subcommands:
- $BIN tx tokenfactory --help

Expected transactions:
- create-denom [denom]
- mint [amount] [mint-to-address]
- burn [amount]
- create-bridge [denom] [client-id] [remote-contract-address]

Notes
- amount must include the full denom for mint/burn (e.g., factory/$ADMIN_ADDR/$DENOM).
- Only the token admin (the creator) can mint and burn.
- Denom must be alphanumeric and between 1 and 20 chars.


## 7) Create a new denom

Pick a unique denom and create it as the admin key:
- export DENOM=tf$(date +%s)
- $BIN tx tokenfactory create-denom $DENOM --from $ADMIN_KEY $TX_FLAGS

The full denom is deterministic:
- export FULL_DENOM=factory/$ADMIN_ADDR/$DENOM

Verify via queries:
- $BIN query tokenfactory denoms-by-creator $ADMIN_ADDR --node $NODE -o json | jq -r .denoms[]
- $BIN query tokenfactory denom-authority-metadata $FULL_DENOM --node $NODE -o json | jq

You should see FULL_DENOM in the creator list, and admin equal to ADMIN_ADDR in the authority metadata.


## 8) Mint tokens and verify balances

Pre-check balances:
- $BIN query bank balances $ADMIN_ADDR --node $NODE -o json | jq
- $BIN query bank balances $USER_ADDR --node $NODE -o json | jq

Mint to yourself (admin):
- $BIN tx tokenfactory mint 1000000$FULL_DENOM $ADMIN_ADDR --from $ADMIN_KEY $TX_FLAGS

Mint to another user:
- $BIN tx tokenfactory mint 600000$FULL_DENOM $USER_ADDR --from $ADMIN_KEY $TX_FLAGS

Post-check balances:
- $BIN query bank balances $ADMIN_ADDR --node $NODE -o json | jq
- $BIN query bank balances $USER_ADDR --node $NODE -o json | jq

Optional: demonstrate sending the new denom via bank send:
- $BIN tx bank send $USER_ADDR $ADMIN_ADDR 100000$FULL_DENOM $TX_FLAGS
- $BIN query bank balances $ADMIN_ADDR --node $NODE -o json | jq
- $BIN query bank balances $USER_ADDR --node $NODE -o json | jq


## 9) Burn tokens and verify balances

Ensure the admin holds some of FULL_DENOM (from the mint above). Then burn from admin:
- $BIN tx tokenfactory burn 250000$FULL_DENOM --from $ADMIN_KEY $TX_FLAGS

Re-check balances to confirm the burn reduced the admin’s FULL_DENOM balance by 250000:
- $BIN query bank balances $ADMIN_ADDR --node $NODE -o json | jq


## 10) Bridge commands

If you want to exercise the bridge mapping CLI, you can set and query a bridge for your denom. The client-id must be a valid IBC client identifier (e.g., 07-tendermint-0 or 08-wasm-0), and the remote contract/address must be a non-empty string.

Create a bridge mapping:
- export CLIENT_ID=07-tendermint-0
- export REMOTE=0x1111111111111111111111111111111111111111
- $BIN tx tokenfactory create-bridge $FULL_DENOM $CLIENT_ID $REMOTE --from $ADMIN_KEY $TX_FLAGS

Query the bridge you just set:
- $BIN query tokenfactory bridge $FULL_DENOM $CLIENT_ID --node $NODE -o json | jq

List bridges by denom and all bridges:
- $BIN query tokenfactory denom-bridges $FULL_DENOM --node $NODE -o json | jq
- $BIN query tokenfactory bridges --node $NODE -o json | jq


## 11) Negative tests (permissions)

Try minting as a non-admin and expect failure:
- $BIN tx tokenfactory mint 1$FULL_DENOM $USER_ADDR --from $USER_KEY $TX_FLAGS

Try burning as a non-admin and expect failure:
- $BIN tx tokenfactory burn 1$FULL_DENOM --from $USER_KEY $TX_FLAGS

## 12) Cleanup

- Stop the node: make kill-single-node
- Optional: Remove the single-node data directory to start from a clean state next time:
  - rm -rf scripts/node-data/sandbox-1
