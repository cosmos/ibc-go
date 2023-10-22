#!/bin/sh

if [ -d $HOME/.simapp1/ ]; then
    rm -R $HOME/.simapp1/
fi

cd ./build

./simd init ibc --chain-id ibcgo1 --home "/Users/aleksejcistakov/.simapp1"
export VALIDATOR_MNEM=$(./simd keys add validator001 --keyring-backend=test --output json --home "/Users/aleksejcistakov/.simapp1" | jq -r '.mnemonic')
export VALIDATOR_ADDR=$(./simd keys show validator001 --keyring-backend=test --output json  --home "/Users/aleksejcistakov/.simapp1" | jq -r '.address')

export SIMD_RELAYER_MNEM=$(./simd keys add simd_relayer --keyring-backend=test --output json  --home "/Users/aleksejcistakov/.simapp1" | jq -r '.mnemonic')
export SIMD_RELAYER_ADDR=$(./simd keys show simd_relayer --keyring-backend=test --output json  --home "/Users/aleksejcistakov/.simapp1" | jq -r '.address')


./simd genesis add-genesis-account validator001 100000000stake --keyring-backend=test  --home "/Users/aleksejcistakov/.simapp1"
./simd genesis add-genesis-account simd_relayer 10stake --keyring-backend=test  --home "/Users/aleksejcistakov/.simapp1"

./simd genesis gentx validator001 "100000000stake" --amount="100000000stake" --keyring-backend=test --chain-id=ibcgo1 --from=validator001  --home "/Users/aleksejcistakov/.simapp1"
./simd genesis collect-gentxs  --home "/Users/aleksejcistakov/.simapp1"

MONIKER="node001"
P2P_URL=tcp://0.0.0.0:26756
RPC_URL=tcp://0.0.0.0:26757
CMD_URL=http://127.0.0.1:26757
PPROF_LADR=127.0.0.1:6070
GRPS_ADDR=127.0.0.1:9191
PERSISTENT_PEERS=

filename="settings1.sh"
# Проверить, существует ли файл,
if [ -f $filename ]; then
rm $filename
echo "$filename удален"
fi

# Создать пустой файл
touch $filename

echo "export VALIDATOR_ADDR=${VALIDATOR_ADDR}" >> $filename
echo "export VALIDATOR_MNEM=${VALIDATOR_MNEM}" >> $filename

echo "export SIMD_RELAYER_MNEM=${SIMD_RELAYER_MNEM}" >> $filename
echo "export SIMD_RELAYER_ADDR=${SIMD_RELAYER_ADDR}" >> $filename

echo "VALIDATOR_ADDR=${VALIDATOR_ADDR}"
echo "VALIDATOR_MNEM=${VALIDATOR_MNEM}"


echo "export SIMD_RELAYER_MNEM=${SIMD_RELAYER_MNEM}"
echo "export SIMD_RELAYER_ADDR=${SIMD_RELAYER_ADDR}"
 
./simd start --moniker ${MONIKER} --p2p.laddr=${P2P_URL} --rpc.laddr=${RPC_URL} --p2p.persistent_peers=${PERSISTENT_PEERS}  --rpc.pprof_laddr=${PPROF_LADR}  --home "/Users/aleksejcistakov/.simapp1" --grpc.address=${GRPS_ADDR}