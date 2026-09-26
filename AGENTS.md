# Repository instructions for coding agents

This file applies to the entire repository. Add a nested `AGENTS.md` only when a
subtree has genuinely different commands or rules; keep shared guidance here.

## Source of truth

- Treat the `Makefile`, scripts it calls, and `.github/workflows/` as the source
  of truth for build and verification commands. If prose documentation differs,
  follow the executable configuration and report the mismatch.
- Use the Go version declared by the relevant `go.mod`; do not hard-code a
  separate toolchain version in instructions or automation.
- Preserve existing user changes. Check `git status --short` before editing and
  do not clean up unrelated files.
- Keep changes focused. Do not mix feature work, bug fixes, dependency updates,
  and unrelated refactors in one diff.

## Repository map

- This is a multi-module Go implementation of IBC. IBC v1 and v2 coexist, so
  confirm which protocol version a change affects.
- Current Go module roots are `./`, `simapp/`,
  `modules/light-clients/08-wasm/`, and `e2e/`. The tidy, lint, and unit-test
  Make targets discover these modules recursively.
- Production code lives under `modules/`: `core/` implements IBC core,
  `apps/` contains applications and middleware, and `light-clients/` contains
  light-client implementations.
- Shared test infrastructure lives under `testing/`; the runnable test chain is
  under `simapp/`; protobuf sources live under `proto/`.
- Documentation lives under `docs/`. Docker-backed end-to-end tests live under
  `e2e/` and are not part of the normal local test workflow.

## Working method

1. Read the package, its tests, and the nearest relevant documentation before
   editing.
2. Trace callers with `rg`, including the corresponding v1 or v2 path when one
   exists, and fix shared causes at the narrowest common layer.
3. Reuse established repository patterns and dependencies; add abstractions or
   dependencies only when the existing code cannot solve the concrete problem.
4. Do not hand-edit generated files such as `*.pb.go` or `*.pb.gw.go`; change
   their source and regenerate them.
5. Add the smallest regression test that proves changed behavior. Use the
   `testing/` helpers for cross-chain or keeper integration behavior.

## Protocol and Go rules

- State-machine code must be deterministic. Do not let map iteration order,
  wall-clock time, randomness, or host-specific behavior affect state changes.
- Validate packet, relayer, protobuf, and message input before mutating state.
  Return errors for invalid external input; reserve panics for broken internal
  invariants.
- Preserve store keys, protobuf field numbers, wire formats, event semantics,
  and exported APIs unless a breaking change is explicitly intended. Never
  reuse a removed protobuf field number.
- For stored-state changes, provide a migration and test it from representative
  legacy state. For behavior shared by IBC v1 and v2, test each affected path.
- Follow `docs/dev/go-style-guide.md`: use `gofumpt`, wrap errors with `%w`,
  prefer table-driven tests when useful, and use `testing.TB.Context()` as a
  test's initial context.

## Verification

During development, run the narrowest useful test from the owning module root,
for example `go test -mod=readonly ./path/to/package` or add `-run TestName`.
After the final edit, run the checks for the changed surface:

| Changed surface | Required checks |
| --- | --- |
| Go source or tests | `make format`, `make build`, `make lint`, `make test-unit` |
| Any `go.mod` or `go.sum` | `make tidy-all`, then the Go checks above; review every module-file change |
| Protobuf | `make proto-all`, `make proto-check-breaking`, then the Go checks above |
| REST/gRPC service surface | Also run `make proto-swagger-gen` and verify the generated Swagger diff |
| `docs/**` | `make lint-docs` and `make build-docs`; add `make check-docs-links` when links change |

- Use `make lint`; do not substitute an ad hoc `golangci-lint` invocation.
- `make test-unit` covers repository Go modules while excluding Docker-backed
  end-to-end execution. Do not run `e2e/Makefile` targets unless explicitly
  requested.
- Protobuf commands require Docker. If local build prerequisites for Ledger are
  unavailable, use the CI-equivalent `LEDGER_ENABLED=false make build` and say
  so in the handoff.
- For instruction-only or other Markdown changes outside `docs/`,
  `git diff --check` is sufficient unless the change affects executable commands.
- Report each command run and any skipped or failed check; never claim a check
  passed without running it.

## Pull requests and commits

- Do not commit, push, rebase, or open a pull request unless explicitly asked.
- Non-minor work should link an accepted issue or specification. Architecture
  changes without an existing specification require an ADR under
  `docs/architecture/`.
- Update relevant docs and the Unreleased changelog for user-visible, API, or
  state-machine changes. Include tests and godoc for new exported behavior.
- Use Conventional Commits. Breaking changes use `type(api)!` or
  `type(statemachine)!`; API breaking takes precedence when both apply.
- Before handoff, self-review `git diff --check` and `git diff` for accidental
  generated, dependency, or unrelated changes.

## Code review rules

- Flag nondeterministic state transitions. Safe path: sort inputs and derive
  time and height from the SDK context.
- Flag writes performed before all external input is validated. Safe path:
  validate first, then mutate state.
- Flag protobuf, store-layout, or wire-format changes without compatibility
  analysis and migration coverage. Safe path: preserve identifiers or add the
  explicit migration and breaking-change classification.
- Flag a v1-only or v2-only behavior change when the sibling path implements the
  same feature and the asymmetry is unexplained.
- Flag bug fixes without a regression test, and stored-state migrations without
  a legacy-state test.
