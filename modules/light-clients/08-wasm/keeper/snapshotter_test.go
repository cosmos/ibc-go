package keeper_test

import (
	"os"
	"testing"
	"time"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	"github.com/cosmos/ibc-go/v7/testing/simapp"
	"github.com/stretchr/testify/require"
)

func TestSnapshotter(t *testing.T) {
	testCases := []struct {
		name      string
		wasmFiles []string
	}{
		{
			name:      "single contract",
			wasmFiles: []string{"../test_data/ics10_grandpa_cw.wasm.gz"},
		},

		{
			name:      "multiple contract",
			wasmFiles: []string{"../test_data/ics07_tendermint_cw.wasm.gz", "../test_data/ics10_grandpa_cw.wasm.gz"},
		},
	}

	for _, spec := range testCases {
		t.Run(spec.name, func(t *testing.T) {

			wasmClientApp := simapp.SetupWithSnapShotter(t)
			ctx := wasmClientApp.NewUncachedContext(false, tmproto.Header{
				ChainID: "foo",
				Height:  wasmClientApp.LastBlockHeight() + 1,
				Time:    time.Now(),
			})

			var srcChecksumCodes []byte
			var codeHashes [][]byte
			// store contract on chain
			for _, contractDir := range spec.wasmFiles {
				signer := authtypes.NewModuleAddress(govtypes.ModuleName).String()
				code, _ := os.ReadFile(contractDir)
				msg := types.NewMsgStoreCode(signer, code)

				res, err := wasmClientApp.WasmClientKeeper.StoreCode(ctx, msg)
				codeHashes = append(codeHashes, res.Checksum)
				srcChecksumCodes = append(srcChecksumCodes, res.Checksum...)

				require.NoError(t, err)
			}

			// create snapshot
			wasmClientApp.Commit()
			snapshotHeight := uint64(wasmClientApp.LastBlockHeight())
			snapshot, err := wasmClientApp.SnapshotManager().Create(snapshotHeight)
			require.NoError(t, err)
			require.NotNil(t, snapshot)

			// setup dest app with snapshot imported
			destWasmClientApp := simapp.SetupWithEmptyStore(t)

			require.NoError(t, destWasmClientApp.SnapshotManager().Restore(*snapshot))
			for i := uint32(0); i < snapshot.Chunks; i++ {
				chunkBz, err := wasmClientApp.SnapshotManager().LoadChunk(snapshot.Height, snapshot.Format, i)
				require.NoError(t, err)
				end, err := destWasmClientApp.SnapshotManager().RestoreChunk(chunkBz)
				require.NoError(t, err)
				if end {
					break
				}
			}

			var allDestAppCodeHashInWasmVMStore []byte
			// check wasm contracts are imported
			ctx = destWasmClientApp.NewUncachedContext(false, tmproto.Header{
				ChainID: "foo",
				Height:  destWasmClientApp.LastBlockHeight() + 1,
				Time:    time.Now(),
			})
			for _, codeHash := range codeHashes {
				code, err := destWasmClientApp.WasmClientKeeper.GetCodeByCodeHash(ctx, codeHash)
				require.NoError(t, err)
				allDestAppCodeHashInWasmVMStore = append(allDestAppCodeHashInWasmVMStore, generateWasmCodeHash(code)...)

			}

			require.Equal(t, srcChecksumCodes, allDestAppCodeHashInWasmVMStore)
		})
	}

}
