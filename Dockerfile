FROM golang:1.18 as builder

ENV GOPATH=""
ENV GOMODULE="on"

COPY go.mod go.mod
COPY go.sum go.sum

COPY testing/go.mod testing/go.mod
COPY testing/go.sum testing/go.sum

WORKDIR /go/testing
RUN go mod download

WORKDIR /go

ADD testing testing
ADD modules modules
ADD LICENSE LICENSE

COPY Makefile .

RUN make build

FROM ubuntu:20.04

COPY --from=builder /go/build/simd /bin/simd

ENTRYPOINT ["simd"]
