package directive

import (
	"github.com/nicheinc/mock/examples/directive/internal/one/atomic"
	"github.com/nicheinc/mock/examples/directive/internal/one/sort"
	"github.com/nicheinc/mock/examples/directive/internal/one/testing"
)

// Source1, [Source2], and [Source3] demonstrate features related to output
// filename and package import naming. First, go:mock directives can override
// the default mock filename, and multiple mocks (in this case Source1Mock,
// Source2Mock, and Source3Mock) can be generated into the same file.
//
// Second, mock automatically resolves name collisions in package imports.
// source1.go, source2.go, and source3.go import all different sets of "sort",
// "testing", and "atomic" packages, which conflict with each other.
// Furthermore, the two sets of internal "atomic" and "testing" packages
// conflict with packages that are imported automatically into every mock file
// because they're used in the implementation of the mock. For each collision,
// one of the imports should be renamed within sink_mock.go.
//
//go:mock sink_mock.go
type Source1 interface {
	f(sort.Interface, *testing.T, *atomic.Bool)
}
