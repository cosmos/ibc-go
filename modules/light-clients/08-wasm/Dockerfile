FROM golang:1.24-alpine AS builder-base

ARG LIBWASM_VERSION
ARG TARGETARCH

RUN test -n "${LIBWASM_VERSION}"
ENV GOPATH=""

RUN set -eux; apk add --no-cache ca-certificates build-base git libusb-dev linux-headers curl;

# Copy relevant files before go mod download. Replace directives to local paths break if local
# files are not copied before go mod download.
ADD internal internal
ADD testing testing
ADD modules modules
ADD LICENSE LICENSE

COPY go.mod .
COPY go.sum .

RUN go mod download


# Since it is not easy to fully cache a RUN script download of libwasmvm, we use two different stages
# and copy the correct file in the final stage. The multistage setup also helps speed up the build process
FROM alpine:3.21 AS amd64-stage
ARG LIBWASM_VERSION
ADD https://github.com/CosmWasm/wasmvm/releases/download/${LIBWASM_VERSION}/libwasmvm_muslc.x86_64.a /lib/libwasmvm_muslc.x86_64.a


FROM alpine:3.21 AS arm64-stage
ARG LIBWASM_VERSION
ADD https://github.com/CosmWasm/wasmvm/releases/download/${LIBWASM_VERSION}/libwasmvm_muslc.aarch64.a /lib/libwasmvm_muslc.aarch64.a

# We have this one with nothing else in it, because COPY --from can't use variables (but FROM can)
FROM ${TARGETARCH}-stage AS libwasm-stage

# Having this is a separate stage allows the previous stages to run in parallel
FROM builder-base AS builder
WORKDIR /go/modules/light-clients/08-wasm

COPY --from=libwasm-stage /lib/libwasmvm_muslc.* /lib/

RUN go build -mod=readonly -tags "netgo ledger muslc" -ldflags '-X github.com/cosmos/cosmos-sdk/version.Name=sim -X github.com/cosmos/cosmos-sdk/version.AppName=simd -X github.com/cosmos/cosmos-sdk/version.Version= -X github.com/cosmos/cosmos-sdk/version.Commit= -X "github.com/cosmos/cosmos-sdk/version.BuildTags=netgo ledger muslc," -w -s -linkmode=external -extldflags "-Wl,-z,muldefs -static"' -trimpath -o /go/build/ ./...


FROM alpine:3.21
COPY --from=builder /go/build/simd /bin/simd
ENTRYPOINT ["simd"]
