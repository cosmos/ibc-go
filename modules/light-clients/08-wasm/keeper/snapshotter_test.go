package keeper_test

import (
	"encoding/hex"
	"time"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/testing/simapp"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
)

func (s *KeeperTestSuite) TestSnapshotter() {
	gzippedContract, err := types.GzipIt(wasmtesting.CreateMockContract([]byte("gzipped-contract")))
	s.Require().NoError(err)

	testCases := []struct {
		name      string
		contracts [][]byte
	}{
		{
			name:      "single contract",
			contracts: [][]byte{wasmtesting.Code},
		},
		{
			name:      "multiple contracts",
			contracts: [][]byte{wasmtesting.Code, gzippedContract},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			t := s.T()
			wasmClientApp := s.SetupSnapshotterWithMockVM()

			ctx := wasmClientApp.NewUncachedContext(false, cmtproto.Header{
				ChainID: "foo",
				Height:  wasmClientApp.LastBlockHeight() + 1,
				Time:    time.Now(),
			})

			var srcChecksumCodes []byte
			var checksums [][]byte
			// store contract on chain
			for _, contract := range tc.contracts {
				signer := authtypes.NewModuleAddress(govtypes.ModuleName).String()
				msg := types.NewMsgStoreCode(signer, contract)

				res, err := wasmClientApp.WasmClientKeeper.StoreCode(ctx, msg)
				s.Require().NoError(err)

				checksums = append(checksums, res.Checksum)
				srcChecksumCodes = append(srcChecksumCodes, res.Checksum...)

				s.Require().NoError(err)
			}

			// create snapshot
			res, err := wasmClientApp.Commit()
			s.Require().NoError(err)
			s.Require().NotNil(res)

			snapshotHeight := uint64(wasmClientApp.LastBlockHeight())
			snapshot, err := wasmClientApp.SnapshotManager().Create(snapshotHeight)
			s.Require().NoError(err)
			s.Require().NotNil(snapshot)

			// setup dest app with snapshot imported
			destWasmClientApp := simapp.SetupWithEmptyStore(t, s.mockVM)
			destCtx := destWasmClientApp.NewUncachedContext(false, cmtproto.Header{
				ChainID: "bar",
				Height:  destWasmClientApp.LastBlockHeight() + 1,
				Time:    time.Now(),
			})

			resp, err := destWasmClientApp.WasmClientKeeper.Checksums(destCtx, &types.QueryChecksumsRequest{})
			s.Require().NoError(err)
			s.Require().Empty(resp.Checksums)

			s.Require().NoError(destWasmClientApp.SnapshotManager().Restore(*snapshot))

			for i := range snapshot.Chunks {
				chunkBz, err := wasmClientApp.SnapshotManager().LoadChunk(snapshot.Height, snapshot.Format, i)
				s.Require().NoError(err)

				end, err := destWasmClientApp.SnapshotManager().RestoreChunk(chunkBz)
				s.Require().NoError(err)

				if end {
					break
				}
			}

			var allDestAppChecksumsInWasmVMStore []byte
			// check wasm contracts are imported
			ctx = destWasmClientApp.NewUncachedContext(false, cmtproto.Header{
				ChainID: "foo",
				Height:  destWasmClientApp.LastBlockHeight() + 1,
				Time:    time.Now(),
			})

			for _, checksum := range checksums {
				resp, err := destWasmClientApp.WasmClientKeeper.Code(ctx, &types.QueryCodeRequest{Checksum: hex.EncodeToString(checksum)})
				s.Require().NoError(err)

				checksum, err := types.CreateChecksum(resp.Data)
				s.Require().NoError(err)

				allDestAppChecksumsInWasmVMStore = append(allDestAppChecksumsInWasmVMStore, checksum...)
			}

			s.Require().Equal(srcChecksumCodes, allDestAppChecksumsInWasmVMStore)
		})
	}
}
