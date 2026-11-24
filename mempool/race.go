//go:build race

package mempool

// RaceEnabled reports whether the binary was built with the race detector.
//
// The race detector makes sync.Pool drop returned items at random instead of
// reusing them, so with RaceEnabled a test can't expect an item that was put
// back into a pool to be handed out again.
const RaceEnabled = true
