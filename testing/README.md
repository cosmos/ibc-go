# IBC Testing Package 

**NOTE:** The IBC testing package is undergoing a significant refactor. This README reflects the new changes and thus may not be useful yet. This note will be removed when the refactor is complete.

## Components

The testing package comprises of four parts constructed as a stack.
- coordinator
- chain
- path
- endpoint

A coordinator sits at the height level and contains all the chains which have been initialized.
It also stores and updates the current global time. The time is manually incremented by a `TimeIncrement`. 
This allows all the chains to remain in synchrony to prevent update issues if the counterparty is perceived to
be in the future. The coordinator also contains functions to do basic setup of clients, connections, and channels
between two chains. 

A chain is an SDK application (as represented by an app.go file). Inside the chain is an `TestingApp` which allows
the chain to simulate block production and transaction processing. The chain contains by default a single tendermint
validator. A chain is used to process SDK messages.

A path connects two channel endpoints. It contains all the information needed to relay between two endpoints. 

An endpoint represents a channel (and its associated client and connections) on some specific chain. It contains
references to the chain it is on and the counterparty chain it is connected to. The endpoint contains functions
to interact with initialization and updates of its associated clients, connections, and channels. 

In general:
- endpoints are used for initialization on one side of an IBC connection
- paths are used to relay packets
- chains are used to commit IBC messages
- coordinator is used to setup a path between two chains 


