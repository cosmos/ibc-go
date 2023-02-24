<!--
order: 1
-->

# `09-localhost`

## Overview

Learn about the 09-localhost light client module. {synopsis}

The 09-localhost light client module implements a localhost loopback client with the ability to send and receive IBC packets to and from the same state machine.

### Context

In a multichain environment, application developers will be used to developing cross-chain applications through IBC. From their point of view, whether or not they are interacting with multiple modules on the same chain or on different chains should not matter. The localhost client module enables a unified interface to interact with different applications on a single chain, using the familiar IBC application layer semantics.

### Implementation

There exists a single sentinel `ClientState` instance with the client identifier `09-localhost`.

To supplement this, a sentinel `ConnectionEnd` is stored in core IBC state with the connection identifier `connection-localhost`. This enables IBC applications to create channels directly on top of the sentinel connection which leverage the 09-localhost loopback functionality.
