# Development setup

## Dependencies

We use [Go 1.14 Modules](https://github.com/golang/go/wiki/Modules) to manage dependency versions.

The main branch of every Cosmos repository should just build with `go get`, which means they should be kept up-to-date with their dependencies, so we can get away with telling  people they can just `go get` our software.

Since some dependencies are not under our control, a third party may break our build, in which case we can fall back on `go mod tidy -v`.

Other helpful commands:

- `go get` to add a new go module (including if the existing go module is being semantic version bumped, i.e. my/module/v1 -> my/module/v2).
- `go get -u` to update an existing dependency.
- `go mod tidy` to update dependencies in `go.sum`.

## Protobuf

We use [Protocol Buffers](https://developers.google.com/protocol-buffers) along with [buf](https://docs.buf.build/introduction) and [gogoproto](https://github.com/gogo/protobuf) to generate code for use in ibc-go.

For determinstic behavior around protobuf tooling, everything is containerized using Docker. Make sure to have Docker installed on your machine, or head to [Docker's website](https://docs.docker.com/get-docker/) to install it.

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

All Go tests in ibc-go can be ran by running `make test`.

Please make sure to run `make format` before every commit - the easiest way to do this is have your editor run it for you upon saving a file. Additionally please ensure that your code is lint compliant by running `make lint-fix` (requires `golangci-lint`).

When testing a function under a variety of different inputs, we prefer to use [table driven tests](https://github.com/golang/go/wiki/TableDrivenTests).

All unit tests should use the testing package. Please see the testing package [README](../../testing/README.md) for more information.

Test coverage is continuously deployed at https://app.codecov.io/github/cosmos/ibc-go. PRs that improve test coverage are welcome, but in general the test coverage should be used as a guidance for finding API use cases that are not covered by tests. We don't recommend adding tests that only improve coverage but not actually test a meaning use case.

## Documentation

- If you open a PR on ibc-go, it is mandatory to update the relevant documentation in `/docs`.
- Generate the folder `docs/.vuepress/dist` with all the static files for the documentation site with `make build-docs`.
- Run the documentation site locally with `make view-docs`.
