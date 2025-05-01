package directive

import "github.com/nathanjcochran/mock/examples/directive/internal"

// Generic is a sample generic interface with a complex type parameter list.
//
//go:mock
type Generic[T interface{ byte | internal.Internal }, U any] interface {
	GetT() T
	GetU() U
}
