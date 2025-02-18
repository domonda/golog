package golog

import "testing"

func TestCallingFunction(t *testing.T) {
	f := CallingFunction()
	if f != "github.com/domonda/golog.TestCallingFunction" {
		t.Errorf("CallingFunction() should return the name of the calling function, but got %q", f)
	}
}

func TestCallingFunctionName(t *testing.T) {
	name := CallingFunctionName()
	if name != "TestCallingFunctionName" {
		t.Errorf("CallingFunctionName() should return the name of the calling function, but got %q", name)
	}
}
