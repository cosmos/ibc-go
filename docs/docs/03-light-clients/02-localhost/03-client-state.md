---
title: ClientState
sidebar_label: ClientState
sidebar_position: 3
slug: /ibc/light-clients/localhost/client-state
---


# `ClientState`

Even though the 09-localhost light client is a stateless client, it still has the concept of a `ClientState` that follows the
blockchains own latest height automatically. The `ClientState` is constructed on demand when required.

The 09-localhost `ClientState` maintains a single field used to track the latest sequence of the state machine i.e. the height of the blockchain.

```go
type ClientState struct {
  // the latest height of the blockchain
  LatestHeight clienttypes.Height
}
```

The 09-localhost `ClientState` is available from the 02-client submodule in core IBC, and does not need to be initialized.

It is possible to disable the localhost client by removing the `09-localhost` entry from the `allowed_clients` list through governance.

## Client updates

The 09-localhost `ClientState` is stateless, so no client updates are required. The `ClientState` is constructed with the latest height when queried.
It will always follow latest height of the blockchain.

