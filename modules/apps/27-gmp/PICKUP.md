# GMP pickup document

`27-gmp` application is feature complete, however, it is not yet production ready. This document outlines the remaining tasks to make it production ready.

- `BuildAddressPredictable` should be audited and tested for security. Specifcally, what happens if the address was created by sending the address some coins first?
- `DeserializeCosmosTx` supports both protobuf encoding and protojson encoding. However, the support for this is hacky since it tries to decode the tx using both methods and returns the one that works. It would be better to have a more robust way to determine the encoding of the tx.
- Add unit tests and integration tests for the `27-gmp` module. Right now, there are no tests for this module.
- End to end tests should be added to ensure the entire flow works as expected. End to end tests should be added to this repository if Cosmos to Cosmos calls are to be supported, otherwise, the end to end tests in the `solidity-ibc-go` repository should be sufficient.
