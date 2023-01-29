FROM golang:1.19 as builder

ENV GOPATH=""
ENV GOMODULE="on"

COPY go.mod .
COPY go.sum .

RUN go mod download

# TODO: add specific Dockerfile to each branch adding only the required directories.
#ADD internal internal
#ADD testing testing
#ADD modules modules
#ADD LICENSE LICENSE
#COPY contrib/devtools/Makefile contrib/devtools/Makefile
#COPY Makefile .

ADD . .

RUN make build

FROM ubuntu:20.04

COPY --from=builder /go/build/simd /bin/simd

ENTRYPOINT ["simd"]
