FROM golang:1.20 as builder

ARG IBC_GO_VERSION

ENV GOPATH=""
ENV GOMODULE="on"

# ensure the ibc go version is being specified for this image.
RUN test -n "${IBC_GO_VERSION}"

COPY go.mod .
COPY go.sum .

RUN go mod download

ADD internal internal
ADD testing testing
ADD modules modules
ADD LICENSE LICENSE

COPY contrib/devtools/Makefile contrib/devtools/Makefile
COPY Makefile .


RUN make build

FROM ubuntu:20.04

ARG IBC_GO_VERSION

LABEL "org.cosmos.ibc-go" "${IBC_GO_VERSION}"

COPY --from=builder /go/build/simd /bin/simd

ENTRYPOINT ["simd"]
