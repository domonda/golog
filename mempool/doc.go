// Package mempool provides memory pool implementations for common Go types
// to reduce garbage collection pressure and improve performance in high-throughput
// logging scenarios. It includes pools for maps, slices, and pointers with
// automatic clearing and capacity management.
//
// This package is extensively used by the golog package to pool frequently
// allocated objects such as:
//
// Core Objects:
//   - Message instances (via mempool.Pointer[Message])
//   - Writer slices (via mempool.Slice[Writer] with MinCap: 4)
//   - TextWriter, JSONWriter, CallbackWriter instances
//
// Attribute Objects:
//   - Attrib slices (via mempool.Slice[Attrib] with MinCap: 16)
//   - All attribute types: String, Strings, Int, Ints, Float, Floats, etc.
//   - Specialized types: UUID, UUIDs, JSON, Error, Errors
//
// The golog package maintains global pools for all these types and provides
// DrainAllMemPools() for testing and cleanup. This pooling strategy is
// crucial for golog's zero-allocation logging performance.
//
// Example usage in golog:
//
//	message := messagePool.GetOrNew()
//	// ... use message ...
//	messagePool.ClearAndPutBack(message)
//
//	writers := writersPool.GetOrMake(2, 4)
//	// ... use writers slice ...
//	writersPool.ClearAndPutBack(writers)
package mempool
