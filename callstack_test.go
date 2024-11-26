package golog

import "testing"

func TestCallingFunction(t *testing.T) {
	f := CallingFunction()
	if f != "github.com/domonda/golog.TestCallingFunction" {
		t.Errorf("CallingFunction() should return the name of the calling function, but got %q", f)
	}
}
