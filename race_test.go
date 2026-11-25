//go:build race

package golog

// The race detector changes how many allocations reflect makes,
// so tests that count allocations skip it.
func init() { raceEnabled = true }
