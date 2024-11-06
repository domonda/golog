package golog

import "testing"

func TestCallingFunction(t *testing.T) {
	t.Fatal(CallingFunction())
}
