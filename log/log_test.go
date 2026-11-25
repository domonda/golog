package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFatalAndPanic_Cyclic tests that FatalAndPanic formats a value that
// contains itself with a cycle error and panics, instead of formatting
// it until the stack overflows, a fatal error that recover can't catch.
func TestFatalAndPanic_Cyclic(t *testing.T) {
	s := []any{nil}
	s[0] = s
	assert.Panics(t, func() { FatalAndPanic(s) })
}
