FROM golang:1.20 as builder

ARG IBC_GO_VERSION

ENV GOPATH=""
ENV GOMODULE="on"

# ensure the ibc go version is being specified for this image.
RUN test -n "${IBC_GO_VERSION}"

COPY go.mod .
COPY go.sum .

ADD modules modules
ADD internal internal
ADD testing testing
ADD LICENSE LICENSE

RUN go mod download

COPY contrib/devtools/Makefile contrib/devtools/Makefile
COPY Makefile .

RUN make build
# Grab the wasmvm library in order to copy it into the final image.
RUN cp $(ldd build/simd | grep "libwasmvm" | awk '{ print $3 }') libwasmvm.so

FROM ubuntu:22.04

ARG IBC_GO_VERSION

LABEL "org.cosmos.ibc-go" "${IBC_GO_VERSION}"

COPY --from=builder go/libwasmvm.so /usr/lib/libwasmvm.x86_64.so
COPY --from=builder /go/build/simd /bin/simd

ENTRYPOINT ["simd"]
