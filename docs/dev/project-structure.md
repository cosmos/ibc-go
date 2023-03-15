# Project structure

If you're not familiar with the overall module structure from the SDK modules, please check this [document](https://github.com/cosmos/cosmos-sdk/blob/main/docs/docs/building-modules/11-structure.md) as prerequisite reading.

Every Interchain Standard (ICS) has been developed in its own package. The development team separated the IBC TAO (Transport, Authentication, Ordering) ICS specifications from the IBC application level specification. The following sections describe the architecture of the most relevant directories that comprise this repository.

## `modules` 

This folder contains implementations for the IBC TAO (`core`), IBC applications (`apps`) and light clients (`light-clients`). 

### `core`

- `02-client`: This package is an implementation for Cosmos SDK-based chains of [ICS 02](https://github.com/cosmos/ibc/tree/main/spec/core/ics-002-client-semantics). This implementation defines the types and methods needed to operate light clients tracking other chain's consensus state.
- `03-connection`: This package is an implementation for Cosmos SDK-based chains of [ICS 03](https://github.com/cosmos/ibc/tree/main/spec/core/ics-003-connection-semantics). This implementation defines the types and methods necessary to perform connection handshake between two chains.
- `04-channel`: This package is an implementation for Cosmos SDK-based chains of [ICS 04](https://github.com/cosmos/ibc/tree/main/spec/core/ics-004-channel-and-packet-semantics). This implementation defines the types and methods necessary to perform channel handshake between two chains and ensure correct packet sending flow.
- `05-port`: This package is an implementation for Cosmos SDK-based chains of [ICS 05](https://github.com/cosmos/ibc/tree/main/spec/core/ics-005-port-allocation). This implements the port allocation system by which modules can bind to uniquely named ports.
- `23-commitment`: This package is an implementation for Cosmos SDK-based chains of [ICS 23](https://github.com/cosmos/ibc/tree/main/spec/core/ics-023-vector-commitments). This implementation defines the functions required to prove inclusion or non-inclusion of particular values at particular paths in state.
- `24-host`: This package is an implementation for Cosmos SDK-based chains of [ICS 24](https://github.com/cosmos/ibc/tree/main/spec/core/ics-024-host-requirements).

### `apps`

- `transfer`: This is the Cosmos SDK implementation of the [ICS 20](https://github.com/cosmos/ibc/tree/main/spec/app/ics-020-fungible-token-transfer) protocol, which enables cross-chain fungible token transfers. For more information, read the [module's docs](../apps/transfer/overview.md)
- `27-interchain-accounts`: This is the Cosmos SDK implementation of the [ICS 27](https://github.com/cosmos/ibc/tree/main/spec/app/ics-027-interchain-accounts) protocol, which enables cross-chain account management built upon IBC. For more information, read the [module's documentation](../apps/interchain-accounts/overview.md).
- `29-fee`: This is the Cosmos SDK implementation of the [ICS 29](https://github.com/cosmos/ibc/tree/main/spec/app/ics-029-fee-payment) middleware, which handles packet incentivisation and fee distribution on top of any ICS application protocol, enabling fee payment to relayer operators. For more information, read the [module's documentation](../middleware/ics29-fee/overview.md).

### `light-clients`

- `06-solomachine`: This package implement the types for the Solo Machine light client specified in [ICS 06](https://github.com/cosmos/ibc/tree/main/spec/client/ics-006-solo-machine-client).
- `07-tendermint`: This package implement the types for the Tendermint consensus light client as specified in [ICS 07](https://github.com/cosmos/ibc/tree/main/spec/client/ics-007-tendermint-client).

## `proto`

This folder contains all the Protobuf files used for

- common message type definitions,
- message type definitions related to genesis state,
- `Query` service and related message type definitions,
- `Msg` service and related message type definitions.

## `testing`

This package contains the implementation of the testing package used in unit and integration tests. Please read the [package's documentation](../../testing/README.md) for more information.
