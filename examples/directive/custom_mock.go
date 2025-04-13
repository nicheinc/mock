package directive

import "testing"

// CustomOutputFileMock is a mock implementation of the CustomOutputFile
// interface.
type CustomOutputFileMock struct {
	T *testing.T
}

// Verify that *CustomOutputFileMock implements CustomOutputFile.
var _ CustomOutputFile = &CustomOutputFileMock{}
