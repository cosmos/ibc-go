package keeper

import (
	"context"
	"strconv"
	"strings"

	"github.com/cosmos/sandbox-ledger/x/ift/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	gmptypes "github.com/cosmos/ibc-go/v11/modules/apps/27-gmp/types"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	k Keeper
}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{k: keeper}
}

// RegisterIFTBridge registers a new IBC bridge to a counterparty IFT contract
func (m msgServer) RegisterIFTBridge(goCtx context.Context, msg *types.MsgRegisterIFTBridge) (*types.MsgRegisterIFTBridgeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	params, err := m.k.ParamsStore.Get(ctx)
	if err != nil {
		return nil, err
	}

	if msg.Signer != params.Authority {
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "expected %s, got %s", params.Authority, msg.Signer)
	}

	if msg.Denom == "" {
		return nil, errorsmod.Wrap(types.ErrInvalidDenom, "denom cannot be empty")
	}
	if msg.ClientId == "" {
		return nil, errorsmod.Wrap(types.ErrInvalidClientID, "client_id cannot be empty")
	}
	// Validate constructor and counterparty address
	if err := types.ValidateConstructorString(msg.IftSendCallConstructor); err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidConstructorType, "invalid constructor: %s", err)
	}
	if err := types.ValidateCounterpartyAddress(msg.IftSendCallConstructor, msg.CounterpartyIftAddress); err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidReceiver, "invalid counterparty address: %s", err)
	}

	// Validate IBC client exists
	if _, found := m.k.ibcClientKeeper.GetClientState(ctx, msg.ClientId); !found {
		return nil, errorsmod.Wrapf(types.ErrInvalidClientID, "IBC client %s not found", msg.ClientId)
	}

	// Check if denom exists in token factory
	if !m.k.tokenFactoryKeeper.HasDenom(ctx, msg.Denom) {
		return nil, errorsmod.Wrapf(types.ErrDenomNotFound, "denom %s not found in token factory", msg.Denom)
	}

	// Check if bridge already exists (allow updates)
	isUpdate, err := m.k.IFTBridgeStore.Has(ctx, collections.Join(msg.Denom, msg.ClientId))
	if err != nil {
		return nil, err
	}

	bridge := types.IFTBridge{
		ClientId:               msg.ClientId,
		CounterpartyIftAddress: msg.CounterpartyIftAddress,
		IftSendCallConstructor: msg.IftSendCallConstructor,
	}

	if err := m.k.IFTBridgeStore.Set(ctx, collections.Join(msg.Denom, msg.ClientId), bridge); err != nil {
		return nil, err
	}

	eventType := types.EventTypeIFTBridgeRegistered
	logAction := "registered"
	if isUpdate {
		eventType = types.EventTypeIFTBridgeUpdated
		logAction = "updated"
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			eventType,
			sdk.NewAttribute(types.AttributeKeyDenom, msg.Denom),
			sdk.NewAttribute(types.AttributeKeyClientID, msg.ClientId),
			sdk.NewAttribute(types.AttributeKeyCounterpartyIFTAddress, msg.CounterpartyIftAddress),
			sdk.NewAttribute(types.AttributeKeyIFTSendCallConstructor, msg.IftSendCallConstructor),
		),
	)

	m.k.Logger(ctx).Info("IFT bridge "+logAction,
		"denom", msg.Denom,
		"client_id", msg.ClientId,
		"counterparty_address", msg.CounterpartyIftAddress)

	return &types.MsgRegisterIFTBridgeResponse{}, nil
}

// RemoveIFTBridge removes an existing IBC bridge
func (m msgServer) RemoveIFTBridge(goCtx context.Context, msg *types.MsgRemoveIFTBridge) (*types.MsgRemoveIFTBridgeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	params, err := m.k.ParamsStore.Get(ctx)
	if err != nil {
		return nil, err
	}

	if msg.Signer != params.Authority {
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "expected %s, got %s", params.Authority, msg.Signer)
	}

	// Check if bridge exists
	exists, err := m.k.IFTBridgeStore.Has(ctx, collections.Join(msg.Denom, msg.ClientId))
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errorsmod.Wrapf(types.ErrBridgeNotFound, "bridge for denom %s and client %s not found", msg.Denom, msg.ClientId)
	}

	// Check for pending transfers - cannot remove bridge with in-flight transfers
	hasPending, err := m.k.HasPendingTransfersForBridge(ctx, msg.Denom, msg.ClientId)
	if err != nil {
		return nil, err
	}
	if hasPending {
		return nil, errorsmod.Wrapf(types.ErrBridgeHasPendingTransfers,
			"cannot remove bridge for denom %s and client %s with pending transfers", msg.Denom, msg.ClientId)
	}

	if err := m.k.IFTBridgeStore.Remove(ctx, collections.Join(msg.Denom, msg.ClientId)); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeIFTBridgeRemoved,
			sdk.NewAttribute(types.AttributeKeyDenom, msg.Denom),
			sdk.NewAttribute(types.AttributeKeyClientID, msg.ClientId),
		),
	)

	m.k.Logger(ctx).Info("IFT bridge removed",
		"denom", msg.Denom,
		"client_id", msg.ClientId)

	return &types.MsgRemoveIFTBridgeResponse{}, nil
}

// IFTTransfer initiates a cross-chain token transfer
func (m msgServer) IFTTransfer(goCtx context.Context, msg *types.MsgIFTTransfer) (*types.MsgIFTTransferResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Denom == "" {
		return nil, errorsmod.Wrap(types.ErrInvalidDenom, "denom cannot be empty")
	}
	if msg.ClientId == "" {
		return nil, errorsmod.Wrap(types.ErrInvalidClientID, "client_id cannot be empty")
	}
	if msg.Receiver == "" {
		return nil, errorsmod.Wrap(types.ErrInvalidReceiver, "receiver cannot be empty")
	}
	if msg.Amount.IsNil() {
		return nil, errorsmod.Wrap(types.ErrInvalidAmount, "amount cannot be nil")
	}
	if !msg.Amount.IsPositive() {
		return nil, errorsmod.Wrapf(types.ErrInvalidAmount, "amount must be positive, got %s", msg.Amount)
	}

	// Validate timeout is in the future (consistent with Solidity IFT)
	blockTime := uint64(ctx.BlockTime().Unix())
	if msg.TimeoutTimestamp <= blockTime {
		return nil, errorsmod.Wrapf(types.ErrInvalidTimeout, "timeout %d must be greater than block time %d", msg.TimeoutTimestamp, blockTime)
	}

	sender, err := m.k.addressCodec.StringToBytes(msg.Signer)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid sender address: %s", err)
	}

	// Get bridge info
	bridge, err := m.k.IFTBridgeStore.Get(ctx, collections.Join(msg.Denom, msg.ClientId))
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrBridgeNotFound, "bridge for denom %s and client %s not found", msg.Denom, msg.ClientId)
	}

	// Burn tokens from sender
	if err := m.k.tokenFactoryKeeper.BurnFrom(ctx, msg.Denom, msg.Amount, sender); err != nil {
		return nil, errorsmod.Wrapf(types.ErrBurnFailed, "failed to burn tokens: %s", err)
	}

	counterpartyInfo, found := m.k.ibcClientV2Keeper.GetClientCounterparty(ctx, msg.ClientId)
	if !found {
		return nil, errorsmod.Wrapf(types.ErrInvalidClientID, "counterparty for client %s not found", msg.ClientId)
	}

	// Construct mint call payload based on constructor type
	var payload []byte
	var encoding string
	constructorType := types.ParseConstructorType(bridge.IftSendCallConstructor)

	switch constructorType {
	case types.ConstructorSolana:
		encoding = gmptypes.EncodingProtobuf
		constructor, err := types.NewSolanaConstructor(bridge.IftSendCallConstructor, bridge.CounterpartyIftAddress, m.k.GetModuleAddress().String(), counterpartyInfo.ClientId)
		if err != nil {
			return nil, errorsmod.Wrapf(types.ErrConstructMintCallFailed, "failed to build solana constructor: %s", err)
		}
		receiver := strings.TrimSpace(msg.Receiver)
		payload, err = constructor.ConstructMintCall(m.k.cdc, receiver, msg.Amount, "", "")
		if err != nil {
			return nil, errorsmod.Wrapf(types.ErrConstructMintCallFailed, "failed to construct mint call: %s", err)
		}

	case types.ConstructorEVM:
		var err error
		payload, err = types.ConstructMintCall(m.k.cdc, msg.Receiver, msg.Amount, bridge.IftSendCallConstructor, "", "")
		if err != nil {
			return nil, errorsmod.Wrapf(types.ErrConstructMintCallFailed, "failed to construct mint call: %s", err)
		}

	case types.ConstructorCosmos:
		// For CosmosTx, we need to know the ICA address that will execute the message
		accountID := gmptypes.NewAccountIdentifier(msg.ClientId, m.k.GetModuleAddress().String(), nil)
		icaAddr, err := gmptypes.BuildAddressPredictable(&accountID)
		if err != nil {
			return nil, errorsmod.Wrapf(types.ErrConstructMintCallFailed, "failed to compute ICA address: %s", err)
		}
		icaAddress := icaAddr.String()
		payload, err = types.ConstructMintCall(m.k.cdc, msg.Receiver, msg.Amount, bridge.IftSendCallConstructor, msg.Denom, icaAddress)
		if err != nil {
			return nil, errorsmod.Wrapf(types.ErrConstructMintCallFailed, "failed to construct mint call: %s", err)
		}

	default:
		return nil, errorsmod.Wrapf(types.ErrInvalidConstructorType, "unknown constructor type: %s", constructorType)
	}

	// Send via ICS27-GMP
	sendMsg := &gmptypes.MsgSendCall{
		Sender:           m.k.GetModuleAddress().String(),
		SourceClient:     bridge.ClientId,
		Receiver:         bridge.CounterpartyIftAddress,
		Salt:             nil,
		Payload:          payload,
		TimeoutTimestamp: msg.TimeoutTimestamp,
		Memo:             "",
		Encoding:         encoding,
	}

	handler := m.k.msgRouter.Handler(sendMsg)
	if handler == nil {
		return nil, errorsmod.Wrap(types.ErrSendCallFailed, "no handler for MsgSendCall")
	}

	res, err := handler(ctx, sendMsg)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrSendCallFailed, "failed to send GMP call: %s", err)
	}

	// Propagate events from the GMP handler (critical for relayer to detect send_packet)
	ctx.EventManager().EmitEvents(res.GetEvents())

	// Extract sequence from response - fail if we cannot get the sequence
	// since callbacks need it to match pending transfers
	if len(res.MsgResponses) == 0 {
		return nil, errorsmod.Wrap(types.ErrSendCallFailed, "no response from GMP send call")
	}
	var sendResp gmptypes.MsgSendCallResponse
	if err := sendResp.Unmarshal(res.MsgResponses[0].Value); err != nil {
		return nil, errorsmod.Wrapf(types.ErrSendCallFailed, "failed to unmarshal GMP response: %s", err)
	}
	sequence := sendResp.Sequence

	// Store pending transfer
	pending := types.PendingTransfer{
		Denom:    msg.Denom,
		ClientId: msg.ClientId,
		Sequence: sequence,
		Sender:   msg.Signer,
		Amount:   msg.Amount,
	}

	if err := m.k.SetPendingTransfer(ctx, msg.ClientId, sequence, pending); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeIFTTransferInitiated,
			sdk.NewAttribute(types.AttributeKeyDenom, msg.Denom),
			sdk.NewAttribute(types.AttributeKeyClientID, msg.ClientId),
			sdk.NewAttribute(types.AttributeKeySequence, strconv.FormatUint(sequence, 10)),
			sdk.NewAttribute(types.AttributeKeySender, msg.Signer),
			sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
			sdk.NewAttribute(types.AttributeKeyAmount, msg.Amount.String()),
		),
	)

	m.k.Logger(ctx).Info("IFT transfer initiated",
		"denom", msg.Denom,
		"client_id", msg.ClientId,
		"sequence", sequence,
		"sender", msg.Signer,
		"receiver", msg.Receiver,
		"amount", msg.Amount.String())

	return &types.MsgIFTTransferResponse{Sequence: sequence}, nil
}

// IFTMint mints tokens in response to a cross-chain transfer
func (m msgServer) IFTMint(goCtx context.Context, msg *types.MsgIFTMint) (*types.MsgIFTMintResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Denom == "" {
		return nil, errorsmod.Wrap(types.ErrInvalidDenom, "denom cannot be empty")
	}
	if msg.Amount.IsNil() {
		return nil, errorsmod.Wrap(types.ErrInvalidAmount, "amount cannot be nil")
	}
	if !msg.Amount.IsPositive() {
		return nil, errorsmod.Wrapf(types.ErrInvalidAmount, "amount must be positive, got %s", msg.Amount)
	}

	// Validate denom exists in token factory
	if !m.k.tokenFactoryKeeper.HasDenom(ctx, msg.Denom) {
		return nil, errorsmod.Wrapf(types.ErrDenomNotFound, "denom %s not found in token factory", msg.Denom)
	}

	signer, err := m.k.addressCodec.StringToBytes(msg.Signer)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid signer address: %s", err)
	}

	receiver, err := m.k.addressCodec.StringToBytes(msg.Receiver)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidReceiver, "invalid receiver address: %s", err)
	}

	// Get the ICS27-GMP account for the signer (reverse lookup)
	ics27Account, err := m.k.gmpKeeper.GetAccount(ctx, signer)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrUnauthorizedSender, "failed to get ICS27 account: %s", err)
	}

	accountID := ics27Account.AccountId

	// Get the bridge for this denom and client
	bridge, err := m.k.IFTBridgeStore.Get(ctx, collections.Join(msg.Denom, accountID.ClientId))
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrBridgeNotFound, "bridge for denom %s and client %s not found", msg.Denom, accountID.ClientId)
	}

	// Validate sender is the counterparty IFT address
	if bridge.CounterpartyIftAddress != accountID.Sender {
		return nil, errorsmod.Wrapf(types.ErrUnauthorizedSender, "expected sender %s, got %s", bridge.CounterpartyIftAddress, accountID.Sender)
	}

	// Validate no salt was used (prevents address spoofing)
	if len(accountID.Salt) > 0 {
		return nil, errorsmod.Wrap(types.ErrUnexpectedSalt, "IFT does not allow salted ICS27 accounts")
	}

	// Mint tokens to receiver
	if err := m.k.tokenFactoryKeeper.MintTo(ctx, msg.Denom, msg.Amount, receiver); err != nil {
		return nil, errorsmod.Wrapf(types.ErrMintFailed, "failed to mint tokens: %s", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeIFTMintReceived,
			sdk.NewAttribute(types.AttributeKeyDenom, msg.Denom),
			sdk.NewAttribute(types.AttributeKeyClientID, accountID.ClientId),
			sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
			sdk.NewAttribute(types.AttributeKeyAmount, msg.Amount.String()),
		),
	)

	m.k.Logger(ctx).Info("IFT mint received",
		"denom", msg.Denom,
		"client_id", accountID.ClientId,
		"receiver", msg.Receiver,
		"amount", msg.Amount.String())

	return &types.MsgIFTMintResponse{}, nil
}

// UpdateParams updates the module parameters
func (m msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if m.k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "expected %s, got %s", m.k.GetAuthority(), msg.Authority)
	}

	// Validate new authority address format
	if _, err := m.k.addressCodec.StringToBytes(msg.Params.Authority); err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid new authority address: %s", err)
	}

	if err := m.k.ParamsStore.Set(ctx, msg.Params); err != nil {
		return nil, err
	}

	m.k.Logger(ctx).Info("IFT params updated", "authority", msg.Params.Authority)

	return &types.MsgUpdateParamsResponse{}, nil
}
