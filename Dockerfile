FROM golang:1.22-alpine3.20 as builder
ARG IBC_GO_VERSION

ARG LIBWASM_VERSION
ARG LIBWASM_CHECKSUM

RUN test -n "${LIBWASM_VERSION}"
RUN test -n "${LIBWASM_CHECKSUM}"

RUN set -eux; apk add --no-cache git libusb-dev linux-headers gcc musl-dev make;

ENV GOPATH=""

# Grab the static library and copy it to location that will be found by the linker flag `-lwasmvm_muslc`.
ADD https://github.com/CosmWasm/wasmvm/releases/download/${LIBWASM_VERSION}/libwasmvm_muslc.x86_64.a /lib/libwasmvm_muslc.x86_64.a
RUN sha256sum /lib/libwasmvm_muslc.x86_64.a | grep ${LIBWASM_CHECKSUM}
RUN cp /lib/libwasmvm_muslc.x86_64.a /lib/libwasmvm_muslc.a

# ensure the ibc go version is being specified for this image.
RUN test -n "${IBC_GO_VERSION}"

# Copy relevant files before go mod download. Replace directives to local paths break if local
# files are not copied before go mod download.
ADD internal internal
ADD simapp simapp
ADD testing testing
ADD modules modules
ADD LICENSE LICENSE

COPY contrib/devtools/Makefile contrib/devtools/Makefile
COPY Makefile .

COPY go.mod .
COPY go.sum .

RUN go mod download

RUN BUILD_TAGS=muslc LINK_STATICALLY=true  make build

FROM alpine:3.18
ARG IBC_GO_VERSION

LABEL "org.cosmos.ibc-go" "${IBC_GO_VERSION}"

COPY --from=builder /go/build/simd /bin/simd

ENTRYPOINT ["simd"]
