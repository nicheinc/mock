package directive

import (
	"sort"
	"sync/atomic"
	"testing"
)

//go:mock sink_mock.go
type Source3 interface {
	f(sort.Interface, *testing.T, *atomic.Bool)
}
