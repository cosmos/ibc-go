package e2e

import (
	"context"
	"fmt"
	dockerClient "github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	TestChain1ID       = "test-1"
	TestChain2ID       = "test-2"
	TestContainer1Name = "e2e-chain-1-1"
	TestContainer2Name = "e2e-chain-2-1"
	Chain1ValMnemonic  = "clock post desk civil pottery foster expand merit dash seminar song memory figure uniform spice circle try happy obvious trash crime hybrid hood cushion"
	Chain2ValMnemonic  = "angry twist harsh drastic left brass behave host shove marriage fall update business leg direct reward object ugly security warm tuna model broccoli choice"
)

// chainContainer holds information about test chains. It has their container name, their chain id and a
// mnemonic used to restore a validator account.
type chainContainer struct {
	containerName     string
	chainID           string
	validatorMnemonic string
}

func setupChain(t *testing.T, cli *dockerClient.Client, chain chainContainer) {
	ctx := context.TODO()

	home := fmt.Sprintf("data/%s", chain.chainID)

	initCmd := []string{"simd", "init", "test", "--home", home, "--chain-id", chain.chainID}
	DockerExec(t, ctx, cli, chain.containerName, initCmd)

	val1Cmd := []string{"bash", "-c", fmt.Sprintf(`echo "%s" | simd keys add val1 --home  %s --recover --keyring-backend=test`, chain.validatorMnemonic, home)}
	DockerExec(t, ctx, cli, chain.containerName, val1Cmd)

	genesisAccountCmd := []string{"bash", "-c", fmt.Sprintf("simd add-genesis-account $(simd --home %s keys show val1 --keyring-backend test -a) 100000000000stake  --home %s", home, home)}
	DockerExec(t, ctx, cli, chain.containerName, genesisAccountCmd)

	genTxCmd := []string{"simd", "gentx", "val1", "7000000000stake", "--home", home, "--chain-id", chain.chainID, "--keyring-backend", "test"}
	DockerExec(t, ctx, cli, chain.containerName, genTxCmd)

	collectTxCmd := []string{"simd", "collect-gentxs", "--home", home}
	DockerExec(t, ctx, cli, chain.containerName, collectTxCmd)

	DockerExec(t, ctx, cli, chain.containerName, []string{"bash", "-c", `sed -i -e 's#"tcp://0.0.0.0:26656"#"tcp://0.0.0.0:'"$P2PPORT"'"#g' $CHAIN_DIR/$CHAIN_ID/config/config.toml`})
	DockerExec(t, ctx, cli, chain.containerName, []string{"bash", "-c", `sed -i -e 's#"tcp://127.0.0.1:26657"#"tcp://0.0.0.0:'"$RPCPORT"'"#g' $CHAIN_DIR/$CHAIN_ID/config/config.toml`})
	DockerExec(t, ctx, cli, chain.containerName, []string{"bash", "-c", `sed -i -e 's/timeout_commit = "5s"/timeout_commit = "1s"/g' $CHAIN_DIR/$CHAIN_ID/config/config.toml`})
	DockerExec(t, ctx, cli, chain.containerName, []string{"bash", "-c", `sed -i -e 's/timeout_propose = "3s"/timeout_propose = "1s"/g' $CHAIN_DIR/$CHAIN_ID/config/config.toml`})
	DockerExec(t, ctx, cli, chain.containerName, []string{"bash", "-c", `sed -i -e 's/index_all_keys = false/index_all_keys = true/g' $CHAIN_DIR/$CHAIN_ID/config/config.toml`})
	DockerExec(t, ctx, cli, chain.containerName, []string{"bash", "-c", `sed -i -e 's/enable = false/enable = true/g' $CHAIN_DIR/$CHAIN_ID/config/app.toml`})
	DockerExec(t, ctx, cli, chain.containerName, []string{"bash", "-c", `sed -i -e 's/index_all_keys = false/index_all_keys = true/g' $CHAIN_DIR/$CHAIN_ID/config/config.toml`})
	DockerExec(t, ctx, cli, chain.containerName, []string{"bash", "-c", `sed -i -e 's/swagger = false/swagger = true/g' $CHAIN_DIR/$CHAIN_ID/config/app.toml`})
	DockerExec(t, ctx, cli, chain.containerName, []string{"bash", "-c", `sed -i -e 's#"tcp://0.0.0.0:1317"#"tcp://0.0.0.0:'"$RESTPORT"'"#g' $CHAIN_DIR/$CHAIN_ID/config/app.toml`})
	DockerExec(t, ctx, cli, chain.containerName, []string{"bash", "-c", `sed -i -e 's#":8080"#":'"$ROSETTA"'"#g' $CHAIN_DIR/$CHAIN_ID/config/app.toml`})

	// start chain
	DockerExecUnattached(t, ctx, cli, chain.containerName, []string{"bash", "-c", fmt.Sprintf(`simd start --log_level trace --log_format json --home %s --pruning=nothing --grpc.address="0.0.0.0:$GRPCPORT" --grpc-web.address="0.0.0.0:$GRPCWEB"`, home)})
}

func TestTokenTransfer(t *testing.T) {
	// This test assumes a testing environment is available already.
	cli, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv)
	assert.NoError(t, err)

	chainA := chainContainer{chainID: TestChain1ID, validatorMnemonic: Chain1ValMnemonic, containerName: TestContainer1Name}
	chainB := chainContainer{chainID: TestChain2ID, validatorMnemonic: Chain2ValMnemonic, containerName: TestContainer2Name}

	setupChain(t, cli, chainA)
	setupChain(t, cli, chainB)
}
