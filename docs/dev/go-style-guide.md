
# Go style guide

In order to keep our code looking good with lots of programmers working on it, it helps to have a "style guide", so all the code generally looks quite similar. This doesn't mean there is only one "right way" to write code, or even that this standard is better than your style.  But if we agree to a number of stylistic practices, it makes it much easier to read and modify new code. Please feel free to make suggestions if there's something you would like to add or modify.

We expect all contributors to be familiar with [Effective Go](https://golang.org/doc/effective_go.html) (and it's recommended reading for all Go programmers anyways). Additionally, we generally agree with the suggestions in [Google's style guide](https://google.github.io/styleguide/go/index) and use that as a starting point.

## Code Structure

Perhaps more key for code readability than good commenting is having the right structure. As a rule of thumb, try to write in a logical order of importance, taking a little time to think how to order and divide the code such that someone could scroll down and understand the functionality of it just as well as you do. A loose example of such order would be:

- Constants, global and package-level variables.
- Main struct definition.
- Options (only if they are seen as critical to the struct else they should be placed in another file).
- Initialization/start and stop of the service functions.
- Public functions (in order of most important).
- Private/helper functions.
- Auxiliary structs and function (can also be above private functions or in a separate file).

## General

- Use `gofumpt` to format all code upon saving it (or run `make format`).
- Think about documentation, and try to leave godoc comments, when it will help new developers.
- Every package should have a high level doc.go file to describe the purpose of that package, its main functions, and any other relevant information.
- Applications (e.g. clis/servers) should panic on unexpected unrecoverable errors and print a stack trace.

## Comments

- Use a space after the comment deliminter (ex. `// your comment`).
- Many comments are not sentences. These should begin with a lower case letter and end without a period.
- Conversely, sentences in comments should be sentenced-cased and end with a period.
- Comments should explain *why* something is being done rather than *what* the code is doing. For example:

 The comments in

```go
// assign a variable foo
f := foo
// assign f to b
b := f
```

 have little value, but the following is more useful:

```go
f := foo
// we copy the variable f because we want to preserve the state at time of initialization
b := f
```

## Linting

- Run `make lint` to see linting errors and `make lint-fix` to fix many issues (some linters do not support auto-fix).

## Various

- Functions that return functions should have the suffix `Fn`.
- Names should not [stutter](https://blog.golang.org/package-names). For example, a struct generally shouldn’t have a field named after itself; e.g., this shouldn't occur:

```go
type middleware struct {
  middleware Middleware
}
```

- Acronyms are all capitalized, like "RPC", "gRPC", "API". "MyID", rather than "MyId".
- Whenever it is safe to use Go's built-in `error` instantiation functions (as opposed to Cosmos SDK's error instantiation functions), prefer `errors.New()` instead of `fmt.Errorf()` unless you're actually using the format feature with arguments.
- As a general guideline, prefer to make the methods for a type [either all pointer methods or all value methods.](https://google.github.io/styleguide/go/decisions#receiver-type)

## Importing libraries

- Use [goimports](https://godoc.org/golang.org/x/tools/cmd/goimports).
- Separate imports into blocks. For example:

```go
import (
  // standard library imports
  "fmt"
  "testing"
      
  // external library imports
  "github.com/stretchr/testify/require"

  // Cosmos-SDK imports
  abci "github.com/cometbft/cometbft/abci/types"
      
  // ibc-go library imports
  "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
)
```

Run `make lint-fix` to get the imports ordered and grouped automatically. 

## Dependencies

- Dependencies should be pinned by a release tag, or specific commit, to avoid breaking `go get` when external dependencies are updated.
- Refer to the [contributing](./development-setup.md#dependencies) document for more details.

## Testing

- Make use of table driven testing where possible and not-cumbersome. Read [this blog post](https://dave.cheney.net/2013/06/09/writing-table-driven-tests-in-go) for more information. See the [tests](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/apps/transfer/keeper/msg_server_test.go#L11) for [`Transfer`](https://github.com/cosmos/ibc-go/blob/v7.0.0/modules/apps/transfer/keeper/msg_server.go#L15) for an example.
- Make use of Testify [assert](https://godoc.org/github.com/stretchr/testify/assert) and [require](https://godoc.org/github.com/stretchr/testify/require).
- When using mocks, it is recommended to use Testify [mock](https://pkg.go.dev/github.com/stretchr/testify/mock) along with [Mockery](https://github.com/vektra/mockery) for autogeneration.

## Errors

- Ensure that errors are concise, clear and traceable.
- Depending on the context, use either `cosmossdk.io/errors` or `stdlib` error packages.
- For wrapping errors, use `fmt.Errorf()` with `%w`.
- Panic is appropriate when an internal invariant of a system is broken, while all other cases (in particular, incorrect or invalid usage) should return errors.
- Error messages should be formatted as following:

```go
sdkerrors.Wrapf(
  <most specific error type possible>,
  "<optional text description ended by colon and space>expected %s, got %s",
  <value 1>,
  <value 2>
)
```

## Common mistakes

This is a compilation of some of the common mistakes we see in the repo that should be avoided.

---
Keep receiver names short [Details here](https://google.github.io/styleguide/go/decisions#receiver-names)

```go
// bad
func (chain *TestChain) NextBlock() {
    res, err := chain.App.FinalizeBlock(&abci.RequestFinalizeBlock{
        Height:             chain.ProposedHeader.Height,
        Time:               chain.ProposedHeader.GetTime(),
        NextValidatorsHash: chain.NextVals.Hash(),
    })
    require.NoError(chain.TB, err)
    chain.commitBlock(res)
}
```

```go
// good
func (c *TestChain) NextBlock() {
    // Ommitted
```

---
**Naked returns**

We should always try to avoid naked returns. [Reference](https://google.github.io/styleguide/go/decisions#named-result-parameters)

---
**Function and method calls should not be separated based solely on line length**

The signature of a function or method declaration [should remain on a single line](https://google.github.io/styleguide/go/decisions#function-formatting) to avoid indentation confusion.

```go
// bad
func (im IBCMiddleware) OnRecvPacket(
    ctx sdk.Context,
    channelVersion string,
    packet channeltypes.Packet,
    relayer sdk.AccAddress,
) ibcexported.Acknowledgement {

// good
func (im IBCMiddleware) OnRecvPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
```

---
**Don't Use Get in function/Method names**
[Reference](https://google.github.io/styleguide/go/decisions#getters)

```go
// bad

// GetChainBID returns the chain-id for chain B.
func (tc TestConfig) GetChainBID() string {
    if tc.ChainConfigs[1].ChainID != "" {
        return tc.ChainConfigs[1].ChainID
    }
    return "chainB-1"
}

// good
func (tc TestConfig) ChainID(i int) string {
    if tc.ChainConfigs[i].ChainID != "" {
        return tc.ChainConfigs[i].ChainID
    }
    return "chainB-1"
}
```

---
**Do not make confusing indentation for saving vertical spaces**

```go
// Bad
cases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{"verification success", func() {}, nil},
		{"verification success: delay period passed", func() {
			delayTimePeriod = uint64(1 * time.Second.Nanoseconds())
		}, nil},
		{"delay time period has not passed", func() {
			delayTimePeriod = uint64(1 * time.Hour.Nanoseconds())
		}, errorsmod.Wrap(ibctm.ErrDelayPeriodNotPassed, "failed packet commitment verification for client (07-tendermint-0): cannot verify packet until time: 1577926940000000000, current time: 1577923345000000000")},
		{"client status is not active - client is expired", func() {
			clientState, ok := path.EndpointB.GetClientState().(*ibctm.ClientState)
			suite.Require().True(ok)
			clientState.FrozenHeight = clienttypes.NewHeight(0, 1)
			path.EndpointB.SetClientState(clientState)
		}, errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (07-tendermint-0) status is Frozen")},
	}
```

```go
// Bad
{
	"nil underlying app", func() {
		isNilApp = true
	}, nil,
},
```

```go
// Good
{
    "nil underlying app", 
    func() {
		    isNilApp = true
    }, 
    nil,
},
```

## Good Practices

**Testing context**

Go 1.24 added a (testing.TB).Context() method. In tests, prefer using (testing.TB).Context() over context.Background() to provide the initial context.Context used by the test. Helper functions, environment or test double setup, and other functions called from the test function body that require a context should have one explicitly passed. [Reference](https://google.github.io/styleguide/go/decisions#contexts)

---
**Error Logging**

If you return an error, it’s usually better not to log it yourself but rather let the caller handle it.
[Reference](https://google.github.io/styleguide/go/best-practices.html#error-logging)

---
**Struct defined outside of the package**

Must have fields specified. [Reference](https://google.github.io/styleguide/go/decisions#field-names)

```go
// Good:
r := csv.Reader{
  Comma: ',',
  Comment: '#',
  FieldsPerRecord: 4,
}
```

```go
// Bad:
r := csv.Reader{',', '#', 4, false, false, false, false}
```

---
**Naming struct fields in tabular tests**

If tabular test struct has more than two fields, consider explicitly naming them. If the test struct has one name and one error field, then we can allow upto three fields. If test struct has more fields, consider naming them when writing test cases.

```go
// Good

tests := []struct {
		name                string
		memo                string
		expectedPass        bool
		message             string
		registerInterfaceFn func(registry codectypes.InterfaceRegistry)
		assertionFn         func(t *testing.T, msgs []sdk.Msg)
	}{
		{
			name:         "packet data generation succeeds (MsgDelegate & MsgSend)",
			memo:         "",
			expectedPass: true,
			message:      multiMsg,
			registerInterfaceFn: func(registry codectypes.InterfaceRegistry) {
				stakingtypes.RegisterInterfaces(registry)
				banktypes.RegisterInterfaces(registry)
			},
			assertionFn: func(t *testing.T, msgs []sdk.Msg) {
				t.Helper()
				assertMsgDelegate(t, msgs[0])
				assertMsgBankSend(t, msgs[1])
			},
		},
  }
```

```go
// Bad
testCases := []struct {
		name       string
		malleate   func()
		callbackFn func(
			ctx sdk.Context,
			packetDataUnmarshaler porttypes.PacketDataUnmarshaler,
			packet channeltypes.Packet,
			maxGas uint64,
		) (types.CallbackData, bool, error)
		getSrc bool
	}{
		{
			"success: src_callback v1",
			func() {
				packetData = transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, sender),
				}

				expCallbackData = expSrcCallBack

				s.path.EndpointA.ChannelConfig.Version = transfertypes.V1
				s.path.EndpointA.ChannelConfig.PortID = transfertypes.ModuleName
				s.path.EndpointB.ChannelConfig.Version = transfertypes.V1
				s.path.EndpointB.ChannelConfig.PortID = transfertypes.ModuleName
			},
			types.GetSourceCallbackData,
			true,
		},
  }
```

## Known Anti Patterns

It's strongly recommended [not to create a custom context](https://google.github.io/styleguide/go/decisions#custom-contexts). The Cosmos SDK has it's own context that is passed around, and we should not try to work against that pattern to avoid confusion.

---
Test outputs should include the actual value that the function returned before printing the value that was expected. A standard format for printing test outputs is YourFunc(%v) = %v, want %v. Where you would write “actual” and “expected”, prefer using the words “got” and “want”, respectively. [Reference](https://google.github.io/styleguide/go/decisions#got-before-want)

But testify has it other way around.

`Require.Equal(Expected, Actual)`

This is a known anti pattern that we allow as the testify package is used heavily in tests.
