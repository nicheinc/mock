package directive

import "github.com/nicheinc/mock/examples/directive/internal"

// Generic is a sample generic interface with a complex type parameter list.
//
//go:mock
type Generic[T interface{ byte | internal.Internal }, U any] interface {
	GetT() T
	GetU() U
}

// GenericAlias is an alias of [Generic].
//
//go:mock
type GenericAlias[T interface{ byte | internal.Internal }, U any] = Generic[T, U]
