FROM golang:1.21-alpine3.18 as builder
ARG IBC_GO_VERSION

RUN set -eux; apk add --no-cache git libusb-dev linux-headers gcc musl-dev make ca-certificates build-base;

ENV GOPATH=""
ENV GOMODULE="on"

# ensure the ibc go version is being specified for this image.
RUN test -n "${IBC_GO_VERSION}"

# Grab the static library and copy it to location that will be found by the linker flag `-lwasmvm_muslc`.
# TODO: nice to have: a script to download version matching the wasmvm version in go.mod.
ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.4.0/libwasmvm_muslc.aarch64.a /lib/libwasmvm_muslc.aarch64.a
ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.4.0/libwasmvm_muslc.x86_64.a /lib/libwasmvm_muslc.x86_64.a
RUN sha256sum /lib/libwasmvm_muslc.aarch64.a | grep 2a72c7062e3c791792b3dab781c815c9a76083a7997ce6f9f2799aaf577f3c25
RUN sha256sum /lib/libwasmvm_muslc.x86_64.a | grep 8ea2e3b5fae83e671da2bb51115adc88591045953f509955ec38dc02ea5a7b94

# Copy the library you want to the final location that will be found by the linker flag `-lwasmvm_muslc`
RUN cp /lib/libwasmvm_muslc.${arch}.a /lib/libwasmvm_muslc.a

# Copy relevant files before go mod download. Replace directives to local paths break if local
# files are not copied before go mod download.


ADD internal internal
ADD testing testing
ADD modules modules
ADD LICENSE LICENSE

COPY contrib/devtools/Makefile contrib/devtools/Makefile
COPY Makefile .

COPY go.mod .
COPY go.sum .

RUN go mod download

# force it to use static lib (from above) not standard libgo_cosmwasm.so file
RUN LEDGER_ENABLED=false BUILD_TAGS=muslc LINK_STATICALLY=true make build



FROM alpine:3.18
ARG IBC_GO_VERSION

LABEL "org.cosmos.ibc-go" "${IBC_GO_VERSION}"

COPY --from=builder /go/build/simd /bin/simd

ENTRYPOINT ["simd"]

