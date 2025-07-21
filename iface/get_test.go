package iface

import (
	"testing"

	"github.com/nicheinc/expect"
)

func TestIncrementName(t *testing.T) {
	type testCase struct {
		input      string
		expected   string
		errorCheck expect.ErrorCheck
	}
	run := func(name string, testCase testCase) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			t.Helper()
			actual, err := incrementName(testCase.input)
			testCase.errorCheck(t, err)
			expect.Equal(t, actual, testCase.expected)
		})
	}

	run("Error/OutOfRange", testCase{
		input:      "package9999999999999999999999999999999",
		expected:   "",
		errorCheck: expect.ErrorNonNil,
	})
	run("Success/Empty", testCase{
		input:      "",
		expected:   "2",
		errorCheck: expect.ErrorNil,
	})
	run("Success/NumberOnly", testCase{
		input:      "123",
		expected:   "124",
		errorCheck: expect.ErrorNil,
	})
	run("Success/NumberedPackage", testCase{
		input:      "package7",
		expected:   "package8",
		errorCheck: expect.ErrorNil,
	})
	run("Success/NumberedPackage/LeadingZeros", testCase{
		input:      "package007",
		expected:   "package8",
		errorCheck: expect.ErrorNil,
	})
	run("Success/InternalNumber", testCase{
		input:      "package123package",
		expected:   "package123package2",
		errorCheck: expect.ErrorNil,
	})
	run("Success/InternalNumber/TrailingNumber", testCase{
		input:      "package123package456",
		expected:   "package123package457",
		errorCheck: expect.ErrorNil,
	})
}
