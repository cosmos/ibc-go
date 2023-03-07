
# Go style guide

In order to keep our code looking good with lots of programmers working on it, it helps to have a "style guide", so all the code generally looks quite similar. This doesn't mean there is only one "right way" to write code, or even that this standard is better than your style.  But if we agree to a number of stylistic practices, it makes it much easier to read and modify new code. Please feel free to make suggestions if there's something you would like to add or modify.

We expect all contributors to be familiar with [Effective Go](https://golang.org/doc/effective_go.html) (and it's recommended reading for all Go programmers anyways). Additionally, we generally agree with the suggestions in [Uber's style guide](https://github.com/uber-go/guide/blob/master/style.md) and use that as a starting point.

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
- Comments should explain _why_ something is being done rather than _what_ the code is doing. For example:

	The comments in 

	```
	// assign a variable foo
	f := foo
	// assign f to b
	b := f
	```

	have little value,	but the following is more useful:

	```
	f := foo
	// we copy the variable f because we want to preserve the state at time of initialization
	b := f
	```

## Linting

- Run `make lint-fix` to fix any linting errors.

## Various

- Functions that return functions should have the suffix `Fn`.
- Names should not [stutter](https://blog.golang.org/package-names). For example, a struct generally shouldn’t have a field named after itself; e.g., this shouldn't occur:

	``` golang
	type middleware struct {
		middleware Middleware
	}
	```

- Acronyms are all capitalized, like "RPC", "gRPC", "API". "MyID", rather than "MyId".
- Whenever it is safe to use Go's built-in `error` instantiation functions (as opposed to Cosmos SDK's error instantiation functions), prefer `errors.New()` instead of `fmt.Errorf()` unless you're actually using the format feature with arguments.

## Importing libraries

- Use [goimports](https://godoc.org/golang.org/x/tools/cmd/goimports).
- Separate imports into blocks: one for the standard lib, one for external libs and one for application libs. For example:

```go
import (
  // standard library imports
  "fmt"
  "testing"
      
  // external library imports
  "github.com/stretchr/testify/require"
  abci "github.com/cometbft/cometbft/abci/types"
      
  // ibc-go library imports
  "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
)
```

## Dependencies

- Dependencies should be pinned by a release tag, or specific commit, to avoid breaking `go get` when external dependencies are updated.
- Refer to the [contributing](./development-setup.md#dependencies) document for more details.

## Testing

- Make use of table driven testing where possible and not-cumbersome. Read [this blog post](https://dave.cheney.net/2013/06/09/writing-table-driven-tests-in-go) for more information. See the [tests](https://github.com/cosmos/ibc-go/blob/f24f41ea8a61fe87f6becab94e84de08c8aa9381/modules/apps/transfer/keeper/msg_server_test.go#L11) for [`Transfer`](https://github.com/cosmos/ibc-go/blob/f24f41ea8a61fe87f6becab94e84de08c8aa9381/modules/apps/transfer/keeper/msg_server.go#L15) for an example.
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
