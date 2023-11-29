package keeper

import (
	"context"
	"encoding/hex"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

var _ types.QueryServer = (*Keeper)(nil)

// Code implements the Query/Code gRPC method
func (k Keeper) Code(goCtx context.Context, req *types.QueryCodeRequest) (*types.QueryCodeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	checksum, err := hex.DecodeString(req.Checksum)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid checksum")
	}

	// Only return checksums we previously stored, not arbitrary checksums that might be stored via e.g Wasmd.
	if !types.HasChecksum(sdk.UnwrapSDKContext(goCtx), k.cdc, checksum) {
		return nil, status.Error(codes.NotFound, errorsmod.Wrap(types.ErrWasmChecksumNotFound, req.Checksum).Error())
	}

	code, err := ibcwasm.GetVM().GetCode(checksum)
	if err != nil {
		return nil, status.Error(codes.NotFound, errorsmod.Wrap(types.ErrWasmChecksumNotFound, req.Checksum).Error())
	}

	return &types.QueryCodeResponse{
		Data: code,
	}, nil
}

// Checksums implements the Query/Checksums gRPC method. It returns a list of hex encoded checksums stored.
func (k Keeper) Checksums(goCtx context.Context, req *types.QueryChecksumsRequest) (*types.QueryChecksumsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var checksums []string
	storedChecksums, err := types.GetAllChecksums(ctx, k.cdc)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	for _, hash := range storedChecksums {
		checksums = append(checksums, hex.EncodeToString(hash))
	}

	return &types.QueryChecksumsResponse{
		Checksums: checksums,
	}, nil
}
