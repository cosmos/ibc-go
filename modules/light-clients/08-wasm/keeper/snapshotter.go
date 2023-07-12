package keeper

import (
	"io"

	errorsmod "cosmossdk.io/errors"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	snapshot "github.com/cosmos/cosmos-sdk/snapshots/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
)

var _ snapshot.ExtensionSnapshotter = &WasmSnapshotter{}

// SnapshotFormat format 1 is just gzipped wasm byte code for each item payload. No protobuf envelope, no metadata.
const SnapshotFormat = 1

type WasmSnapshotter struct {
	wasm *Keeper
	cms  sdk.MultiStore
}

func NewWasmSnapshotter(cms sdk.MultiStore, wasm *Keeper) *WasmSnapshotter {
	return &WasmSnapshotter{
		wasm: wasm,
		cms:  cms,
	}
}

func (ws *WasmSnapshotter) SnapshotName() string {
	return types.ModuleName
}

func (ws *WasmSnapshotter) SnapshotFormat() uint32 {
	return SnapshotFormat
}

func (ws *WasmSnapshotter) SupportedFormats() []uint32 {
	// If we support older formats, add them here and handle them in Restore
	return []uint32{SnapshotFormat}
}

func (ws *WasmSnapshotter) SnapshotExtension(height uint64, payloadWriter snapshot.ExtensionPayloadWriter) error {
	cacheMS, err := ws.cms.CacheMultiStoreWithVersion(int64(height))
	if err != nil {
		return err
	}

	ctx := sdk.NewContext(cacheMS, tmproto.Header{}, false, nil)
	seenBefore := make(map[string]bool)
	var rerr error

	ws.wasm.IterateCodeInfos(ctx, func(id uint64, codeID string) bool {
		if seenBefore[codeID] {
			return false
		}
		seenBefore[codeID] = true

		// load code and abort on error
		wasmBytes, err := ws.wasm.GetWasmByte(ctx, codeID)
		if err != nil {
			rerr = err
			return true
		}

		compressedWasm, err := types.GzipIt(wasmBytes)
		if err != nil {
			rerr = err
			return true
		}

		err = payloadWriter(compressedWasm)
		if err != nil {
			rerr = err
			return true
		}

		return false
	})

	return rerr
}

func (ws *WasmSnapshotter) RestoreExtension(height uint64, format uint32, payloadReader snapshot.ExtensionPayloadReader) error {
	if format == SnapshotFormat {
		return ws.processAllItems(height, payloadReader, restoreV1, finalizeV1)
	}
	return snapshot.ErrUnknownFormat
}

func restoreV1(_ sdk.Context, k *Keeper, compressedCode []byte) error {
	if !types.IsGzip(compressedCode) {
		return types.ErrInvalid.Wrap("not a gzip")
	}
	wasmCode, err := types.Uncompress(compressedCode, uint64(types.MaxWasmSize))
	if err != nil {
		return errorsmod.Wrap(errorsmod.Wrap(err, "failed to store contract"), err.Error())
	}

	// FIXME: check which codeIDs the checksum matches??
	_, err = k.wasmVM.StoreCode(wasmCode)
	if err != nil {
		return errorsmod.Wrap(errorsmod.Wrap(err, "failed to store contract"), err.Error())
	}
	return nil
}

func finalizeV1(ctx sdk.Context, k *Keeper) error {
	return nil
}

func (ws *WasmSnapshotter) processAllItems(
	height uint64,
	payloadReader snapshot.ExtensionPayloadReader,
	cb func(sdk.Context, *Keeper, []byte) error,
	finalize func(sdk.Context, *Keeper) error,
) error {
	ctx := sdk.NewContext(ws.cms, tmproto.Header{Height: int64(height)}, false, nil)
	for {
		payload, err := payloadReader()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if err := cb(ctx, ws.wasm, payload); err != nil {
			return errorsmod.Wrap(err, "processing snapshot item")
		}
	}

	return finalize(ctx, ws.wasm)
}
