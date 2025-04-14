package directive

// These types demonstrate the use of go:mock directives overriding the default
// mock filenames, as well as generation of two mocks into the same file.

//go:mock custom_mock.go
type CustomOutputFile1 any

//go:mock custom_mock.go
type CustomOutputFile2 any
