# Mock

`mock` is a code generation tool for creating simple mock implementations of
interfaces for use in testing.

Mocks are thread-safe.

## Installation

You can install `mock` locally using the following command:

```
go install github.com/nicheinc/mock
```

However, if your Go module depends on `mock`, you may wish to add it to your
`go.mod` as a tool dependency:

```
go get -tool github.com/nicheinc/mock
```

If you also vendor your dependencies, this ensures that contributors to your
module can create or update its mocks reproducibly using `go tool mock`,
regardless of which version of `mock` they may or may not have installed
locally.

## Usage

```
Usage: mock [options] [interface]

When the positional interface argument is omitted, all interfaces in the search
directory annotated with a "go:mock [output file]" directive will be mocked and
output to stdout or, with the -w option, written to files. If a go:mock
directive in a file called example.go doesn't specify an output file, the
default output file will be the -o flag (if provided) or else example_mock.go.

When an interface name is provided as a positional argument after all other
flags, only that interface will be mocked. The -w option is incompatible with an
interface argument.

Options:
  -d string
        Directory to search for interfaces in (default ".")
  -o string
        Output file (default stdout)
  -w    Write mocks to files rather than stdout
```

## Example

Given this interface (note the special `go:mock` directive) in a file called
`getter.go`:

```go
package main

//go:mock
type Getter interface {
	GetByID(id int) ([]string, error)
	GetByName(name string) ([]string, error)
}
```

`mock` will generate an implementation like this, and print it to stdout:

```go
package main

import (
	"sync/atomic"
	"testing"
)

// GetterMock is a mock implementation of the Getter
// interface.
type GetterMock struct {
	T               *testing.T
	GetByIDStub     func(id int) ([]string, error)
	GetByIDCalled   int32
	GetByNameStub   func(name string) ([]string, error)
	GetByNameCalled int32
}

// Verify that *GetterMock implements Getter.
var _ Getter = &GetterMock{}

// GetByID is a stub for the Getter.GetByID
// method that records the number of times it has been called.
func (m *GetterMock) GetByID(id int) ([]string, error) {
	atomic.AddInt32(&m.GetByIDCalled, 1)
	if m.GetByIDStub == nil {
		if m.T != nil {
			m.T.Error("GetByIDStub is nil")
		}
		panic("GetByID unimplemented")
	}
	return m.GetByIDStub(id)
}

// GetByName is a stub for the Getter.GetByName
// method that records the number of times it has been called.
func (m *GetterMock) GetByName(name string) ([]string, error) {
	atomic.AddInt32(&m.GetByNameCalled, 1)
	if m.GetByNameStub == nil {
		if m.T != nil {
			m.T.Error("GetByNameStub is nil")
		}
		panic("GetByName unimplemented")
	}
	return m.GetByNameStub(name)
}
```

To write the output to a file instead, pass the `-w` option: `mock -w`.

Voila! There should now be a `getter_mock.go` file containing your new mock, in
the same package as the interface definition. Subsequent runs of `mock -w` will
overwrite the file, so be careful not to edit it!

## Go Generate

> [!tip]
>
> In earlier releases of `mock`, `go generate` was the recommended way to run
> the tool. The disadvantage of the approach outlined below is that
> `go generate` has to run `mock` once per `go:generate` directive, which
> requires parsing the package's AST and computing type information each time.
> For modules with many mocked interfaces, that extra work adds up. However,
> `mock` continues to support this approach for backward compatibility.

To use with `go generate`, simply place a `go:generate` comment somewhere in
your package (e.g. above the interface definition), like so:

```go
//go:generate go tool mock -o getter_mock.go Getter
```

Note the use of the `-o` flag, which specifies the output file. If this flag is
not provided, the mocked implementation will be printed to stdout.

Then run the `go generate` command from the package directory.
