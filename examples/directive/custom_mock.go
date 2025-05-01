package directive

import "testing"

// CustomOutputFile1Mock is a mock implementation of the CustomOutputFile1
// interface.
type CustomOutputFile1Mock struct {
	T *testing.T
}

// Verify that *CustomOutputFile1Mock implements CustomOutputFile1.
var _ CustomOutputFile1 = &CustomOutputFile1Mock{}

// CustomOutputFile2Mock is a mock implementation of the CustomOutputFile2
// interface.
type CustomOutputFile2Mock struct {
	T *testing.T
}

// Verify that *CustomOutputFile2Mock implements CustomOutputFile2.
var _ CustomOutputFile2 = &CustomOutputFile2Mock{}
