# Repository guidelines for coding agents

This file provides guidance for automated agents contributing to this repository.

## Repository Overview

- This is a Cosmos SDK Go implementation of the Inter Blockchain Communication protocol.
- It implements both IBC v1 and IBC v2.
- The project is written in Go, and implements a set of Cosmos SDK modules that are organized under `modules/`.
  - `core/` implements IBC Core:
    - Common components for both IBC v1 and v2:
      - keeper: Root IBC module msg server
      - 02-client: light-client handling and routing
      - 05-port: application port binding and routing
      - 23-commitment: merkle tree types for provable commitments (such as packet commitments)
      - 24-host: host state-machine and related keys
    - IBC v1 only:
      - 03-connection: connection handling and routing, including connection setup handshakes
      - 04-channel: channel handling and routing, including channel setup handshakes
    - IBC v2 only:
      - api: port module router
      - 02-client/v2
      - 04-channel/v2
  - `apps/` contains application level modules and middlewares:
  - `light-clients/` provides implementations of IBC light clients.
- Protobuf definitions live under `proto/`.
- Unit and integration tests reside throughout the repo and rely on the `testing/` package.
- End to end tests live under `e2e/`, but agents are **not** expected to run them.

## Development Workflow

1. **Formatting and linting**
  - Run `make lint` to lint all modules
  - Run `make lint-fix` to automatically fix lint issues via `golangci-lint`.
  - Run `make format` to format Go code with `gofumpt`.
2. **Testing**
   - Execute all unit and integration tests with `make test-unit`.
   - Do not run the e2e tests under `e2e/`.
3. **After making changes to dependencies**
  - Run `make tidy-all` to tidy dependencies across all modules

## Commit Messages

- Follow the Conventional Commits specification. Examples of valid types are
  `feat`, `fix`, `docs`, `test`, `deps`, and `chore`.
- Breaking changes must use the `(api)!` or `(statemachine)!` suffix.
- Include the proposed commit message in the pull request description.

Refer to `docs/dev/pull-requests.md` for more details on commit conventions and
pull request guidelines.
