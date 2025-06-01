FROM golang:1.24-alpine AS builder
ARG IBC_GO_VERSION

RUN set -eux; apk add --no-cache gcc git libusb-dev linux-headers make musl-dev;

ENV GOPATH=""

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

RUN make build

FROM alpine:3.21
ARG IBC_GO_VERSION

LABEL "org.cosmos.ibc-go"="${IBC_GO_VERSION}"

COPY --from=builder /go/build/simd /bin/simd

ENTRYPOINT ["simd"]
