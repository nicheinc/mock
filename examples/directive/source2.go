package directive

import (
	"github.com/nicheinc/mock/examples/directive/internal/two/atomic"
	"github.com/nicheinc/mock/examples/directive/internal/two/sort"
	"github.com/nicheinc/mock/examples/directive/internal/two/testing"
)

//go:mock sink_mock.go
type Source2 interface {
	f(sort.Interface, *testing.T, *atomic.Bool)
}
