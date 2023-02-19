<!--
order: 1
-->

# `09-localhost`

## Overview

Learn about the 09-localhost light client module. {synopsis}

The 09-localhost light client module implements a localhost loopback client with the ability to send and receive IBC packets to and from the same state machine.

There exists a single sentinel `ClientState` instance with the client identifier `09-localhost`. 

To supplement this, a sentinel `ConnectionEnd` is stored in core IBC state with the connection identifier `connection-localhost`. This enables IBC applications to create channels directly on top of the sentinel connection which leverage the 09-localhost loopback functionality.

