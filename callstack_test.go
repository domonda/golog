package golog

import (
	"testing"
)

func testFuncForCallingFunction() string {
	defer func() {}() // Prevent inlining
	return CallingFunction()
}

func genericTestFuncForCallingFunction[T any]() string {
	defer func() {}() // Prevent inlining
	return CallingFunction()
}

func testFuncForCallingFunctionName() string {
	defer func() {}() // Prevent inlining
	return CallingFunctionName()
}

func genericTestFuncForCallingFunctionName[T any]() string {
	defer func() {}() // Prevent inlining
	return CallingFunctionName()
}

func TestCallingFunction(t *testing.T) {
	f := CallingFunction()
	if f != "github.com/domonda/golog.TestCallingFunction" {
		t.Errorf("CallingFunction() should return the name of the calling function, but got %q", f)
	}
	if testFuncForCallingFunction() != "github.com/domonda/golog.testFuncForCallingFunction" {
		t.Errorf("CallingFunction() should return the name of the calling function, but got %q", testFuncForCallingFunction())
	}
	if genericTestFuncForCallingFunction[int]() != "github.com/domonda/golog.genericTestFuncForCallingFunction" {
		t.Errorf("CallingFunction() should return the name of the calling function, but got %q", genericTestFuncForCallingFunction[int]())
	}
}

func TestCallingFunctionName(t *testing.T) {
	name := CallingFunctionName()
	if name != "TestCallingFunctionName" {
		t.Errorf("CallingFunctionName() should return the name of the calling function, but got %q", name)
	}
	if testFuncForCallingFunctionName() != "testFuncForCallingFunctionName" {
		t.Errorf("CallingFunctionName() should return the name of the calling function, but got %q", testFuncForCallingFunctionName())
	}
	if genericTestFuncForCallingFunctionName[int]() != "genericTestFuncForCallingFunctionName" {
		t.Errorf("CallingFunctionName() should return the name of the calling function, but got %q", genericTestFuncForCallingFunctionName[int]())
	}
}

func TestCallingFunctionPackageName(t *testing.T) {
	pkg := CallingFunctionPackageName()
	if pkg != "golog" {
		t.Errorf("CallingFunctionPackageName() should return the package name of the calling function, but got %q", pkg)
	}
}
