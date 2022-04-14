package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	SubModuleName = "beefy-client"
)

// IBC beefy client errors
var (
	ErrInvalidChainID             = sdkerrors.Register(SubModuleName, 0, "invalid chain-id")
	ErrInvalidRootHash            = sdkerrors.Register(SubModuleName, 1, "invalid root hash")
	ErrInvalidHeaderHeight        = sdkerrors.Register(SubModuleName, 2, "invalid header height")
	ErrProcessedTimeNotFound      = sdkerrors.Register(SubModuleName, 3, "processed time not found")
	ErrProcessedHeightNotFound    = sdkerrors.Register(SubModuleName, 4, "processed height not found")
	ErrDelayPeriodNotPassed       = sdkerrors.Register(SubModuleName, 5, "packet-specified delay period has not been reached")
	ErrCommitmentNotFinal         = sdkerrors.Register(SubModuleName, 6, "commitment isn't final")
	ErrAuthoritySetUnknown        = sdkerrors.Register(SubModuleName, 7, "authority set is unknown")
	ErrInvalidCommitment          = sdkerrors.Register(SubModuleName, 8, "invalid commitment")
	ErrInvalidCommitmentSignature = sdkerrors.Register(SubModuleName, 9, "invalid commitment signature")
	ErrInvalidMMRLeaf             = sdkerrors.Register(SubModuleName, 10, "invalid MMR leaf")
	ErrFailedEncodeMMRLeaf        = sdkerrors.Register(SubModuleName, 11, "failed to encode MMR leaf")
	ErrFailedVerifyMMRLeaf        = sdkerrors.Register(SubModuleName, 12, "failed to verify MMR leaf")
	ErrInvalivParachainHeadsProof = sdkerrors.Register(SubModuleName, 13, "invalid parachain heads proof")
)
