#!/bin/sh

if [ -d $HOME/.simapp/ ]; then
    rm -R $HOME/.simapp/
fi

cd ./build

./simd init ibc --chain-id ibcgo
export VALIDATOR_MNEM=$(./simd keys add validator001 --keyring-backend=test --output json | jq -r '.mnemonic')
export VALIDATOR_ADDR=$(./simd keys show validator001 --keyring-backend=test --output json | jq -r '.address')

export SIMD_RELAYER_MNEM=$(./simd keys add simd_relayer --keyring-backend=test --output json | jq -r '.mnemonic')
export SIMD_RELAYER_ADDR=$(./simd keys show simd_relayer --keyring-backend=test --output json | jq -r '.address')


./simd genesis add-genesis-account validator001 100000000stake --keyring-backend=test
./simd genesis add-genesis-account simd_relayer 10stake --keyring-backend=test

./simd genesis gentx validator001 "100000000stake" --amount="100000000stake" --keyring-backend=test --chain-id=ibcgo --from=validator001
./simd genesis collect-gentxs


MONIKER="node001"
P2P_URL=tcp://0.0.0.0:26656
RPC_URL=tcp://0.0.0.0:26657
CMD_URL=http://127.0.0.1:26657
PERSISTENT_PEERS=


filename="settings.sh"
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

./simd start --moniker ${MONIKER} p2p.laddr=${P2P_URL} --rpc.laddr=${RPC_URL} --p2p.persistent_peers=${PERSISTENT_PEERS}