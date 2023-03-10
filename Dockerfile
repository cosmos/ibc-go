FROM golang:1.19 as builder

ARG IBC_GO_VERSION

ENV GOPATH=""
ENV GOMODULE="on"

# ensure the ibc go version is being specified for this image.
RUN test -n "${IBC_GO_VERSION}"

# TODO: move this back down to below go mod tidy once we are not using a local pin.
ADD modules modules

COPY go.mod .
COPY go.sum .

RUN go mod download

ADD internal internal
ADD testing testing
ADD LICENSE LICENSE

COPY contrib/devtools/Makefile contrib/devtools/Makefile
COPY Makefile .


RUN make build

FROM ubuntu:20.04

ARG IBC_GO_VERSION

LABEL "org.cosmos.ibc-go" "${IBC_GO_VERSION}"

COPY --from=builder /go/build/simd /bin/simd

ENTRYPOINT ["simd"]
