# ibc-go

Interblockchain communication protocol (IBC) implementation in Golang built as a SDK module. 

## Components

### Core

The `core/` directory contains the SDK IBC module that SDK based chains must integrate in order to utilize this implementation of IBC.
It handles the core components of IBC including clients, connection, channels, packets, acknowledgements, and timeouts. 

### Applications

Applications can be built as modules to utilize core IBC by fulfilling a set of callbacks. 
Fungible Token Transfers is currently the only supported application module. 

### IBC Light Clients

IBC light clients are on-chain implementations of an off-chain light clients.
This repository currently supports tendermint and solo-machine light clients. 
The localhost client is currently non-functional. 

## Docs

Please see our [documentation](docs/README.md) for more information.


