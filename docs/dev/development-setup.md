# Development setup

## Dependencies

We use [Go 1.16 Modules](https://go.dev/wiki/Modules) to manage dependency versions.

The main branch of every Cosmos repository should just build with `go get`, which means they should be kept up-to-date with their dependencies, so we can get away with telling  people they can just `go get` our software.

Since some dependencies are not under our control, a third party may break our build, in which case we can fall back on `go mod tidy -v`.

Other helpful commands:

- `go get` to add a new go module (including if the existing go module is being semantic version bumped, i.e. my/module/v1 -> my/module/v2).
- `go get -u` to update an existing dependency.
- `go mod tidy` to update dependencies in `go.sum`.

## Vulnerability scanning

The [OpenSSF Scorecard](https://scorecard.dev/viewer/?uri=github.com/cosmos/ibc-go) Vulnerabilities check uses [OSV-Scanner](https://google.github.io/osv-scanner/). To reproduce it locally, scan every manifest in the repository (all `go.mod` files, `requirements.txt` and `docs/package-lock.json`) from the repository root:

```shell
go run github.com/google/osv-scanner/v2/cmd/osv-scanner@latest scan source -r .
```

To check whether ibc-go code actually calls a reported Go vulnerability, enable Go call analysis, which runs [govulncheck](https://go.dev/doc/tutorial/govulncheck) while still applying `osv-scanner.toml`. Add `--all-vulns` to also list vulnerabilities that are not called:

```shell
go run github.com/google/osv-scanner/v2/cmd/osv-scanner@latest scan source -r --call-analysis=go .
```

Fix vulnerabilities by bumping the dependency, then run `make tidy-all`. For the docs site, use `npm audit fix --package-lock-only` in `docs/`, or add an entry to `overrides` in `docs/package.json` when a transitive dependency pins a vulnerable version.

A vulnerability with no fix, or one that does not affect ibc-go, can be ignored by adding an `[[IgnoredVulns]]` entry with an `id` and a `reason` to the root `osv-scanner.toml`. OSV-Scanner only applies that file to manifests in the same directory, so every other manifest directory has an `osv-scanner.toml` symlink to it. Edit only the root file.

On every pull request, the Dependency Review workflow (`.github/workflows/dependencies-review.yml`) runs the call-analysis scan above and also fails on newly introduced high-severity vulnerabilities. Both accept the advisories listed in the root `osv-scanner.toml`.

## Protobuf

We use [Protocol Buffers](https://developers.google.com/protocol-buffers) along with [buf](https://docs.buf.build/introduction) and [gogoproto](https://github.com/gogo/protobuf) to generate code for use in ibc-go.

For deterministic behavior around protobuf tooling, everything is containerized using Docker. Make sure to have Docker installed on your machine, or head to [Docker's website](https://docs.docker.com/get-docker/) to install it.

For formatting code in `.proto` files, you can run the `make proto-format` command.

For linting and checking breaking changes, we also use [buf](https://buf.build/). You can use the commands `make proto-lint` and `make proto-check-breaking` to respectively lint your proto files and check for breaking changes.

To generate the protobuf stubs, you can run `make proto-gen`.

We also added the `make proto-all` command to run the above commands (`proto-format`, `proto-lint` and `proto-gen`) sequentially.

To update third-party protobuf dependencies, you can run `make proto-update-deps`. This requires `buf` to be installed in the local development environment (see [`buf`s installation documentation](https://docs.buf.build/installation) for more details).

For generating or updating the swagger file that documents the URLs of the RESTful API that exposes the gRPC endpoints over HTTP, you can run the `proto-swagger-gen` command.

It reads protobuf service definitions and generates a reverse-proxy server which translates a RESTful HTTP API into gRPC.

## Developing and testing

- The latest state of development is on `main`.
- Build the `simd` test chain binary with `make build`.
- `main` must never fail `make test`.
- No `--force` onto `main` (except when reverting a broken commit, which should seldom happen).
- Create a development branch either on `github.com/cosmos/ibc-go`, or your fork (using `git remote add fork`).
- Before submitting a pull request, begin `git rebase` on top of `main`.
- Ensure you are using the pre-commit hooks by running `make setup-pre-commit`.

All Go tests in ibc-go can be ran by running `make test`.

Please make sure to run `make format` before every commit - the easiest way to do this is have your editor run it for you upon saving a file. Additionally please ensure that your code is lint compliant by running `make lint-fix` (requires `golangci-lint`).

When testing a function under a variety of different inputs, we prefer to use [table driven tests](https://github.com/golang/go/wiki/TableDrivenTests).

All unit tests should use the testing package. Please see the testing package [README](../../testing/README.md) for more information.

## Documentation

- If you open a PR on ibc-go, it is mandatory to update the relevant documentation in `/docs`.
- We lint the markdown files for documentation with [markdownlint-cli](https://github.com/igorshubovych/markdownlint-cli). Please run `make docs-lint` before pushing changes in the markdown files (you will need to have `markdownlint-cli` installed, so please follow the [installation instructions](https://github.com/igorshubovych/markdownlint-cli#installation)).
- Generate the folder `docs/.vuepress/dist` with all the static files for the documentation site with `make build-docs`.
- Run the documentation site locally with `make view-docs`.
