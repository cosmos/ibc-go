package e2e

import (
	"context"
	"fmt"
	dockerClient "github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
)

const (
	TestChain1ID         = "test-1"
	TestChain2ID         = "test-2"
	TestContainer1Name   = "e2e-chain-1-1"
	TestContainer2Name   = "e2e-chain-2-1"
	RelayerContainerName = "e2e-hermes-1"
	Chain1ValMnemonic    = "clock post desk civil pottery foster expand merit dash seminar song memory figure uniform spice circle try happy obvious trash crime hybrid hood cushion"
	Chain2ValMnemonic    = "angry twist harsh drastic left brass behave host shove marriage fall update business leg direct reward object ugly security warm tuna model broccoli choice"
)

type Context struct {
	client           *dockerClient.Client
	chains           []chainContainer
	relayerContainer relayerContainer
}

type relayerContainer struct {
	containerName  string
	configFilePath string
}

// chainContainer holds information about test chains. It has their container name, their chain id and a
// mnemonic used to restore a validator account.
type chainContainer struct {
	containerName     string
	chainID           string
	validatorMnemonic string
}

func (c chainContainer) home() string {
	return fmt.Sprintf("data/%s", c.chainID)
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

func generateKeys(t *testing.T, ctx Context) {
	for _, chain := range ctx.chains {
		generateKeyCmd := []string{"bash", "-c", fmt.Sprintf(`hermes -c %s keys restore %s -m "%s"`, ctx.relayerContainer.configFilePath, chain.chainID, chain.validatorMnemonic)}
		DockerExec(t, context.TODO(), ctx.client, ctx.relayerContainer.containerName, generateKeyCmd)
	}
}

func createConnection(t *testing.T, ctx Context) {
	connectionCreationCmd := []string{"hermes", "-c", ctx.relayerContainer.configFilePath, "create", "connection", ctx.chains[0].chainID, ctx.chains[1].chainID}
	DockerExec(t, context.TODO(), ctx.client, ctx.relayerContainer.containerName, connectionCreationCmd)
}

func createChannel(t *testing.T, ctx Context) {
	channelCreationCmd := []string{"hermes", "-c", ctx.relayerContainer.configFilePath, "create", "channel", "--port-a", "transfer", "--port-b", "transfer", ctx.chains[0].chainID, "connection-0"}
	DockerExec(t, context.TODO(), ctx.client, ctx.relayerContainer.containerName, channelCreationCmd)
}

func startHermes(t *testing.T, ctx Context) {
	startCmd := []string{"hermes", "-c", ctx.relayerContainer.configFilePath, "start"}
	DockerExecUnattached(t, context.TODO(), ctx.client, ctx.relayerContainer.containerName, startCmd)
}

func getWalletAddress(t *testing.T, ctx Context, chain chainContainer) string {
	// simd --home %s keys show val1 --keyring-backend test -a
	getAddressCmd := []string{"simd", "--home", chain.home(), "keys", "show", "val1", "--keyring-backend", "test", "-a"}
	cmd := DockerExec(t, context.TODO(), ctx.client, chain.containerName, getAddressCmd)
	return strings.TrimSuffix(cmd.Stdout(), "\n")
}

func transferTokens(t *testing.T, ctx Context, amount string) {
	fromAddress := getWalletAddress(t, ctx, ctx.chains[0])
	toAddress := getWalletAddress(t, ctx, ctx.chains[1])

	//assert.Equal(t, fromAddress, "cosmos1qnk2n4nlkpw9xfqntladh74w6ujtulwn7j8za9")

	sender := ctx.chains[0]

	sendTokensCmd := []string{"bash", "-c", fmt.Sprintf(`simd tx ibc-transfer transfer transfer channel-0 "%s" %s --from "%s" --home %s --keyring-backend test --chain-id  %s --node tcp://localhost:16657 --yes`, toAddress, amount, fromAddress, sender.home(), sender.chainID)}
	//sendTokensCmd := []string{"bash", "-c", fmt.Sprintf(`simd tx ibc-transfer transfer transfer channel-0 cosmos1qnk2n4nlkpw9xfqntladh74w6ujtulwn7j8za9 %s --from cosmos18hl5c9xn5dze2g50uaw0l2mr02ew57zk2fgr8q --home %s --keyring-backend test --chain-id  %s --node tcp://localhost:16657 --yes`, amount, sender.home(), sender.chainID)}
	DockerExec(t, context.TODO(), ctx.client, sender.containerName, sendTokensCmd)
}

//simd tx ibc-transfer transfer transfer channel-0 $WALLET_3 1000stake --from $WALLET_1 --home ./data/test-1 --keyring-backend test --chain-id test-1 --node tcp://localhost:16657

func TestTokenTransfer(t *testing.T) {
	// This test assumes a testing environment is available already.
	cli, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv)
	assert.NoError(t, err)

	ctx := Context{
		client: cli,
		relayerContainer: relayerContainer{
			containerName:  RelayerContainerName,
			configFilePath: "/config/config.toml",
		},
		chains: []chainContainer{
			{chainID: TestChain1ID, validatorMnemonic: Chain1ValMnemonic, containerName: TestContainer1Name},
			{chainID: TestChain2ID, validatorMnemonic: Chain2ValMnemonic, containerName: TestContainer2Name},
		},
	}

	t.Run("chains are configured and started", func(t *testing.T) {
		for _, chain := range ctx.chains {
			setupChain(t, cli, chain)
		}
	})

	t.Run("keys are generated", func(t *testing.T) {
		time.Sleep(time.Second * 5)
		generateKeys(t, ctx)
	})

	t.Run("connection is successfully created", func(t *testing.T) {
		time.Sleep(time.Second * 5)
		createConnection(t, ctx)
	})

	t.Run("channel is successfully created", func(t *testing.T) {
		time.Sleep(time.Second * 5)
		createChannel(t, ctx)
	})

	t.Run("hermes is successfully started", func(t *testing.T) {
		time.Sleep(time.Second * 5)
		startHermes(t, ctx)
	})

	t.Run("transfer happens successfully", func(t *testing.T) {
		transferTokens(t, ctx, "1000stake")
	})
}
