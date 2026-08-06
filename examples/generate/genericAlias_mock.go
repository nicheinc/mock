package generate

import (
	"sync/atomic"
	"testing"

	"github.com/nicheinc/mock/examples/generate/internal"
)

// GenericAliasMock is a mock implementation of the GenericAlias
// interface.
type GenericAliasMock[T interface{ byte | internal.Internal }, U any] struct {
	T          *testing.T
	GetTStub   func() T
	GetTCalled int32
	GetUStub   func() U
	GetUCalled int32
}

// Verify that *GenericAliasMock implements GenericAlias.
func _[T interface{ byte | internal.Internal }, U any]() {
	var _ GenericAlias[T, U] = &GenericAliasMock[T, U]{}
}

// GetT is a stub for the GenericAlias.GetT
// method that records the number of times it has been called.
func (m *GenericAliasMock[T, U]) GetT() T {
	atomic.AddInt32(&m.GetTCalled, 1)
	if m.GetTStub == nil {
		if m.T != nil {
			m.T.Error("GetTStub is nil")
		}
		panic("GetT unimplemented")
	}
	return m.GetTStub()
}

// GetU is a stub for the GenericAlias.GetU
// method that records the number of times it has been called.
func (m *GenericAliasMock[T, U]) GetU() U {
	atomic.AddInt32(&m.GetUCalled, 1)
	if m.GetUStub == nil {
		if m.T != nil {
			m.T.Error("GetUStub is nil")
		}
		panic("GetU unimplemented")
	}
	return m.GetUStub()
}
