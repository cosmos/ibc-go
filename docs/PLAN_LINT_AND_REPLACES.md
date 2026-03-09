# Plan: Lint Fixes + Replace Local Paths with Commit Hashes

## Replace Strategy

- **Keep local replaces** for in-repo testing (e.g. e2e, evmd) — same-repo references stay as `../`
- **Use commit hashes** for anything outside the repo — cross-repo dependencies must be pinned to a commit

## Current State

### Cross-Repo Replaces (→ use commit hash)

| Repo | Local Replace | Target Repo | Current Commit |
|------|---------------|-------------|----------------|
| **wasmd** | `cosmos-sdk => /Users/cozart/dev/cosmos-sdk` | cosmos-sdk | `482c72c957710ac5cfd8447169b411ec3235a034` |
| **ibc-go e2e** | `interchaintest/v11 => ../../interchaintest` | interchaintest | `50dd401fd4c841bde9d8b9cb07953189ad558643` |
| **interchaintest** | `wasmd => ../wasmd` | wasmd | `e0dd487bf04bdae8876cb7754ad526a8712c49bf` |
| **interchaintest** | `evm => ../evm` | evm | `5e62ed124cadc108e5647468c29fb838dc2bb2a1` |
| **evm** | `ibc-go/v11 => ../ibc-go` | ibc-go | `8906e4df35aa2e21f302851ab0337a77c0e0232e` |
| **evm evmd** | `ibc-go/v11 => ../../ibc-go` | ibc-go | `8906e4df35aa2e21f302851ab0337a77c0e0232e` |

### In-Repo Replaces (→ keep local)

| Repo | Local Replace | Reason |
|------|---------------|--------|
| **ibc-go e2e** | `ibc-go/v11 => ../` | Same repo (e2e is under ibc-go) |
| **ibc-go e2e** | `ibc-go/.../08-wasm/v11 => ../modules/light-clients/08-wasm` | Same repo |
| **evm evmd** | `evm => ../` | Same repo (evmd is under evm) |

### Repos with Lint Failures (blocking commit/push)

| Repo | Lint Issues |
|------|-------------|
| **ibc-go** | 33 revive (package-naming, use-slices-sort) + 1 e2e typecheck (missing go.sum) |
| **cosmos-sdk** | 6 gosec G118 (context cancel not called) |

---

## Part 1: Replace Local Paths with Commit Hashes

### Strategy

Use `go get module@commithash` to generate correct pseudo-versions, then remove local path replaces.

### Implementation Order

1. **cosmos-sdk** (no local replace in; it's a dependency)
2. **wasmd** (depends on cosmos-sdk)
3. **evm** (depends on ibc-go)
4. **interchaintest** (depends on wasmd, evm)
5. **ibc-go e2e** (depends on ibc-go, interchaintest)

### Commands

```bash
# wasmd - replace local cosmos-sdk with commit (cross-repo)
cd wasmd
go mod edit -dropreplace github.com/cosmos/cosmos-sdk
go get github.com/cosmos/cosmos-sdk@482c72c957710ac5cfd8447169b411ec3235a034
go mod tidy

# evm - replace local ibc-go with commit (cross-repo)
cd evm
go mod edit -dropreplace github.com/cosmos/ibc-go/v11
go get github.com/cosmos/ibc-go/v11@8906e4df35aa2e21f302851ab0337a77c0e0232e
go mod tidy

# evm evmd - replace local ibc-go only (keep evm => ../, same repo)
cd evm/evmd
go mod edit -dropreplace github.com/cosmos/ibc-go/v11
go get github.com/cosmos/ibc-go/v11@8906e4df35aa2e21f302851ab0337a77c0e0232e
go mod tidy

# interchaintest - replace local wasmd and evm with commits (cross-repo)
cd interchaintest
go mod edit -dropreplace github.com/CosmWasm/wasmd
go mod edit -dropreplace github.com/cosmos/evm
go get github.com/CosmWasm/wasmd@e0dd487bf04bdae8876cb7754ad526a8712c49bf
go get github.com/cosmos/evm@5e62ed124cadc108e5647468c29fb838dc2bb2a1
go mod tidy

# ibc-go e2e - replace interchaintest only (keep ibc-go and 08-wasm local, same repo)
cd ibc-go/e2e
go mod edit -dropreplace github.com/cosmos/interchaintest/v11
go get github.com/cosmos/interchaintest/v11@50dd401fd4c841bde9d8b9cb07953189ad558643
go mod tidy
```

### Important Notes

- **ibc-go e2e** keeps `../` for ibc-go and 08-wasm (in-repo testing).
- **evm evmd** keeps `../` for evm root (in-repo).
- **evm/evmd → ibc-go**: Keep local — ibc-go has unpushed commits.
- **interchaintest → wasmd**: Keep local — wasmd commit not found by proxy (chore/update branch).
- After each change: `make lint`, `make build`, `make test` to verify.

### Completed (2026-03-09)

- wasmd: cosmos-sdk → `e5b941276b` ✓
- ibc-go e2e: interchaintest → `50dd401fd4c8` ✓
- interchaintest: evm → `5e62ed124cad` ✓

---

## Part 2: Lint Fixes for ibc-go and cosmos-sdk

### cosmos-sdk (6 gosec G118 + 1 G120)

| File | Issue | Fix |
|------|-------|-----|
| `server/api/server.go:208` | G120: form parsing without MaxBytesReader | Wrap request body with `http.MaxBytesReader` |
| `testutil/network/util.go:95` | G118: cancel not called | `val.cancelFn` is stored - ensure it's called on cleanup |
| `testutil/simsx/msg_factory_test.go:89` | G118: done not called | Add `defer done()` |
| `testutil/simsx/registry.go:140` | G118: done not called | Add `defer done()` |
| `testutil/systemtests/system.go:1144` | G118: timeout cancel discarded | Use `ctx, cancel := context.WithTimeout` and `defer cancel()` |

### ibc-go (33 revive + 1 typecheck)

**e2e typecheck (quick fix):**
```bash
cd ibc-go/e2e
go mod tidy
# or: go get github.com/cosmos/ibc-go/e2e/testsuite
```

**revive (33 issues):**
- **package-naming** (types, utils, errors): Either add exclude in `.golangci.yml` or rename packages (large refactor).
- **use-slices-sort**: Replace `sort.Sort`, `sort.Strings`, `sort.SliceStable` with `slices.Sort`, `slices.SortFunc`, `slices.SortStableFunc`.

---

## Execution Order

1. **Fix cosmos-sdk lint** → commit & push → get new commit hash
2. **Fix ibc-go e2e typecheck** (go mod tidy) → verify ibc-go lint/build
3. **Replace cross-repo paths** with commit hashes (keep in-repo local replaces):
   - wasmd: cosmos-sdk
   - evm + evmd: ibc-go
   - interchaintest: wasmd, evm
   - ibc-go e2e: interchaintest only (keep ibc-go, 08-wasm local)
4. **ibc-go revive** – decide: exclude in config vs. fix (revive fixes are mechanical but numerous)

---

## Verification Checklist

After each change:
- [ ] `make lint` passes
- [ ] `make build` passes
- [ ] `make test` passes (or `make test-unit` for ibc-go)
- [ ] No local path replaces in go.mod (except cosmos-sdk internal tools)
- [ ] `go mod tidy` produces no changes
