# Overview

Learn about the `09-localhost` light client module. {synopsis}

The `09-localhost` light client module implements a localhost loopback client with the ability to send and receive IBC packets on the same state machine.

There exists a single localhost `ClientState` instance with the client identifier `09-localhost`. 
To supplement this, a sentinel `ConnectionEnd` is stored in state with the connection identifier `connection-localhost`. This enables custom IBC applications to create channels directly on top of the sentinel connection which leverage localhost loopback functionality.

