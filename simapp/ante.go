package simapp

import (
	"errors"

	circuitante "cosmossdk.io/x/circuit/ante"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"

	ibcante "github.com/cosmos/ibc-go/v10/modules/core/ante"
	"github.com/cosmos/ibc-go/v10/modules/core/keeper"
)

/**
 * HandlerOptions: Extends the standard SDK options to include 
 * circuit breaking and IBC capabilities.
 */
type HandlerOptions struct {
	ante.HandlerOptions
	CircuitKeeper circuitante.CircuitBreaker
	IBCKeeper     *keeper.Keeper
}

/**
 * NewAnteHandler: Constructor for the transaction validation pipeline.
 * Order of decorators is critical for security (e.g., Check gas before signatures).
 */
func NewAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	// --- Integrity Checks ---
	if options.AccountKeeper == nil {
		return nil, errors.New("critical: account keeper is required for ante builder")
	}
	if options.BankKeeper == nil {
		return nil, errors.New("critical: bank keeper is required for ante builder")
	}
	if options.SignModeHandler == nil {
		return nil, errors.New("critical: sign mode handler is required for ante builder")
	}

	[Image of Cosmos SDK AnteHandler execution flow and transaction lifecycle]

	anteDecorators := []sdk.AnteDecorator{
		// 1. Context Setup: Must be the outermost layer
		ante.NewSetUpContextDecorator(),
		
		// 2. Circuit Breaker: Emergency stop for specific transaction types
		circuitante.NewCircuitBreakerDecorator(options.CircuitKeeper),
		
		// 3. Basic Validation: Stateless checks (memo length, gas prices, timeouts)
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		
		// 4. Gas & Fee Management: Deduct fees before heavy cryptographic checks
		// This prevents Spam/DoS attacks by charging the user early.
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, options.TxFeeChecker),
		
		// 5. Authentication: Public key and signature verification
		ante.NewSetPubKeyDecorator(options.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		
		// 6. State Updates: Increment sequence to prevent replay attacks
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		
		// 7. IBC Support: Specific checks for inter-blockchain communication
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
	}

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}
