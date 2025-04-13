package directive

import "github.com/nicheinc/mock/example/directive/internal"

// MyGenericInterface is a sample generic interface with a complex type
// parameter list.
//
//go:mock
type MyGenericInterface[T interface{ byte | internal.Internal }, U any] interface {
	GetT() T
	GetU() U
}
