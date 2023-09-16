FROM golang:1.21-alpine3.18 as builder

RUN set -eux; apk add --no-cache git libusb-dev linux-headers gcc musl-dev make;


ENV GOPATH=""
ENV GOMODULE="on"

# Copy relevant files before go mod download. Replace directives to local paths break if local
# files are not copied before go mod download.
ADD internal internal
ADD testing testing
ADD modules modules
ADD LICENSE LICENSE

COPY go.mod .
COPY go.sum .

WORKDIR modules/apps/callbacks

# Grab the static library and copy it to location that will be found by the linker flag `-lwasmvm_muslc`.
# TODO: nice to have: a script to download version matching the wasmvm version in go.mod.
ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.4.0/libwasmvm_muslc.aarch64.a /lib/libwasmvm_muslc.aarch64.a
ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.4.0/libwasmvm_muslc.x86_64.a /lib/libwasmvm_muslc.x86_64.a
RUN sha256sum /lib/libwasmvm_muslc.aarch64.a | grep 2a72c7062e3c791792b3dab781c815c9a76083a7997ce6f9f2799aaf577f3c25
RUN sha256sum /lib/libwasmvm_muslc.x86_64.a | grep 8ea2e3b5fae83e671da2bb51115adc88591045953f509955ec38dc02ea5a7b94

# Copy the library you want to the final location that will be found by the linker flag `-lwasmvm_muslc`
RUN cp /lib/libwasmvm_muslc.${arch}.a /lib/libwasmvm_muslc.a

RUN go mod download

RUN BUILD_TAGS=muslc GOOS=linux GOARCH=amd64 LINK_STATICALLY=true LEDGER_ENABLED=false go build -mod=readonly -tags "netgo ledger" -ldflags '-X github.com/cosmos/cosmos-sdk/version.Name=sim -X github.com/cosmos/cosmos-sdk/version.AppName=simd -X github.com/cosmos/cosmos-sdk/version.Version= -X github.com/cosmos/cosmos-sdk/version.Commit= -X "github.com/cosmos/cosmos-sdk/version.BuildTags=netgo ledger," -w -s' -trimpath -o /go/build/ ./...

RUN echo "Ensuring binary is statically linked ..." \
  && (file /code/build/wasmd | grep "statically linked")


FROM alpine:3.18

COPY --from=builder /go/build/simd /bin/simd

ENTRYPOINT ["simd"]
