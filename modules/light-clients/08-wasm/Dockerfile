FROM golang:1.21-alpine as builder

RUN apk add --no-cache ca-certificates build-base git libusb-dev linux-headers curl

ENV GOPATH=""


# Copy relevant files before go mod download. Replace directives to local paths break if local
# files are not copied before go mod download.
ADD internal internal
ADD testing testing
ADD modules modules
ADD LICENSE LICENSE

COPY go.mod .
COPY go.sum .

RUN go mod download

WORKDIR /go/modules/light-clients/08-wasm

RUN set -eux; \
  WASM_VERSION=$(go list -m all | grep github.com/CosmWasm/wasmvm/v2 | awk '{print $2}'); \
  if [ ! -z "${WASM_VERSION}" ]; then \
    curl -L -o /lib/libwasmvm_muslc.x86_64.a https://github.com/CosmWasm/wasmvm/releases/download/${WASM_VERSION}/libwasmvm_muslc.x86_64.a; \
    curl -L -o /lib/libwasmvm_muslc.aarch64.a https://github.com/CosmWasm/wasmvm/releases/download/${WASM_VERSION}/libwasmvm_muslc.aarch64.a; \
  fi;

RUN go build -mod=readonly -tags "netgo ledger muslc" -ldflags '-X github.com/cosmos/cosmos-sdk/version.Name=sim -X github.com/cosmos/cosmos-sdk/version.AppName=simd -X github.com/cosmos/cosmos-sdk/version.Version= -X github.com/cosmos/cosmos-sdk/version.Commit= -X "github.com/cosmos/cosmos-sdk/version.BuildTags=netgo ledger muslc," -w -s -linkmode=external -extldflags "-Wl,-z,muldefs -static"' -trimpath -o /go/build/ ./...

FROM alpine:3.18

COPY --from=builder /go/build/simd /bin/simd

ENTRYPOINT ["simd"]
