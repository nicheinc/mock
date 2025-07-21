package directive

import (
	sort3 "sort"
	"sync/atomic"
	"testing"

	atomic2 "github.com/nicheinc/mock/examples/directive/internal/one/atomic"
	"github.com/nicheinc/mock/examples/directive/internal/one/sort"
	testing2 "github.com/nicheinc/mock/examples/directive/internal/one/testing"
	atomic3 "github.com/nicheinc/mock/examples/directive/internal/two/atomic"
	sort2 "github.com/nicheinc/mock/examples/directive/internal/two/sort"
	testing3 "github.com/nicheinc/mock/examples/directive/internal/two/testing"
)

// Source1Mock is a mock implementation of the Source1
// interface.
type Source1Mock struct {
	T       *testing.T
	fStub   func(sort.Interface, *testing2.T, *atomic2.Bool)
	fCalled int32
}

// Verify that *Source1Mock implements Source1.
var _ Source1 = &Source1Mock{}

// f is a stub for the Source1.f
// method that records the number of times it has been called.
func (m *Source1Mock) f(param1 sort.Interface, param2 *testing2.T, param3 *atomic2.Bool) {
	atomic.AddInt32(&m.fCalled, 1)
	if m.fStub == nil {
		if m.T != nil {
			m.T.Error("fStub is nil")
		}
		panic("f unimplemented")
	}
	m.fStub(param1, param2, param3)
}

// Source2Mock is a mock implementation of the Source2
// interface.
type Source2Mock struct {
	T       *testing.T
	fStub   func(sort2.Interface, *testing3.T, *atomic3.Bool)
	fCalled int32
}

// Verify that *Source2Mock implements Source2.
var _ Source2 = &Source2Mock{}

// f is a stub for the Source2.f
// method that records the number of times it has been called.
func (m *Source2Mock) f(param1 sort2.Interface, param2 *testing3.T, param3 *atomic3.Bool) {
	atomic.AddInt32(&m.fCalled, 1)
	if m.fStub == nil {
		if m.T != nil {
			m.T.Error("fStub is nil")
		}
		panic("f unimplemented")
	}
	m.fStub(param1, param2, param3)
}

// Source3Mock is a mock implementation of the Source3
// interface.
type Source3Mock struct {
	T       *testing.T
	fStub   func(sort3.Interface, *testing.T, *atomic.Bool)
	fCalled int32
}

// Verify that *Source3Mock implements Source3.
var _ Source3 = &Source3Mock{}

// f is a stub for the Source3.f
// method that records the number of times it has been called.
func (m *Source3Mock) f(param1 sort3.Interface, param2 *testing.T, param3 *atomic.Bool) {
	atomic.AddInt32(&m.fCalled, 1)
	if m.fStub == nil {
		if m.T != nil {
			m.T.Error("fStub is nil")
		}
		panic("f unimplemented")
	}
	m.fStub(param1, param2, param3)
}
