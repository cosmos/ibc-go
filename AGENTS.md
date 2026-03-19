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

1. **Pre-push verification**
   - Before committing or pushing, run in order: `make tidy-all`, `make build`, `make lint`, `make test-unit`
   - Fix any failures locally before pushing
   - Use `make lint` (or repo equivalent) — never run golangci-lint directly with custom paths; match CI
   - Other repos: same pattern (tidy → build → lint → test) using that repo's Makefile targets
2. **Multi-repo verification** (when changes span ibc-go, wasmd, interchaintest, evm)
   - At the end of each step, verify **lint and build** pass for all four repos
   - ibc-go: `make lint` and `make build`
   - wasmd: `make format` then `make lint`, and `make build`
   - interchaintest: `golangci-lint run ./...` and `go build ./...`
   - evm: `make lint-go` and `make build` (or equivalent)
3. **Formatting and linting**
   - Run `make lint` to lint all modules
   - Run `make lint-fix` to automatically fix lint issues via `golangci-lint`
   - Run `make format` to format Go code with `gofumpt`
4. **Testing**
   - Execute all unit and integration tests with `make test-unit`
   - Do not run the e2e tests under `e2e/`
5. **After making changes to dependencies**
   - Run `make tidy-all` to tidy dependencies across all modules

## Commit Messages

- Follow the Conventional Commits specification. Examples of valid types are
  `feat`, `fix`, `docs`, `test`, `deps`, and `chore`.
- Breaking changes must use the `(api)!` or `(statemachine)!` suffix.
- Include the proposed commit message in the pull request description.

Refer to `docs/dev/pull-requests.md` for more details on commit conventions and
pull request guidelines.
