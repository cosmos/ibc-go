# Dockerfile

FROM alpine:latest

RUN apk add --no-cache curl grep

ENV WASM_IMPORT="github.com/CosmWasm/wasmvm/v2"
ENV WASM_GO_MOD_PATH="modules/light-clients/08-wasm/go.mod"
ENV WASM_LIBRARY="libwasmvm_muslc.x86_64.a"

# Copy the go.mod file from the correct location
COPY ${WASM_GO_MOD_PATH} /tmp/go.mod

RUN WASM_VERSION=$(grep "$WASM_IMPORT" /tmp/go.mod | awk '{print $2}') && \
    echo "WASM_VERSION=${WASM_VERSION}" > /tmp/version.env

RUN source /tmp/version.env && \
    curl -sL "https://github.com/CosmWasm/wasmvm/releases/download/${WASM_VERSION}/checksums.txt" | \
    grep "${WASM_LIBRARY}" | awk '{print $1}' > /tmp/checksum.env

CMD echo "CosmWasm Version: $(cat /tmp/version.env)"; echo "Checksum: $(cat /tmp/checksum.env)"
