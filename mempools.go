package golog

import "github.com/domonda/golog/mempool"

var (
	messagePool        mempool.Pointer[Message]
	textWriterPool     mempool.Pointer[TextWriter]
	jsonWriterPool     mempool.Pointer[JSONWriter]
	callbackWriterPool mempool.Pointer[CallbackWriter]
)

var (
	attribsPool = mempool.Slice[Attrib]{
		MinCap: 16,
	}
	stringPool  mempool.Pointer[String]
	stringsPool mempool.Pointer[Strings]
	nilPool     mempool.Pointer[Nil]
	anyPool     mempool.Pointer[Any]
	boolPool    mempool.Pointer[Bool]
	boolsPool   mempool.Pointer[Bools]
	intPool     mempool.Pointer[Int]
	intsPool    mempool.Pointer[Ints]
	uintPool    mempool.Pointer[Uint]
	uintsPool   mempool.Pointer[Uints]
	floatPool   mempool.Pointer[Float]
	floatsPool  mempool.Pointer[Floats]
	errorPool   mempool.Pointer[Error]
	errorsPool  mempool.Pointer[Errors]
	timePool    mempool.Pointer[Time]
	timesPool   mempool.Pointer[Times]
	uuidPool    mempool.Pointer[UUID]
	uuidsPool   mempool.Pointer[UUIDs]
	jsonPool    mempool.Pointer[JSON]
)

func DrainAllMemPools() {
	messagePool.Drain()
	textWriterPool.Drain()
	jsonWriterPool.Drain()
	callbackWriterPool.Drain()
	attribsPool.Drain()
	stringPool.Drain()
	stringsPool.Drain()
	nilPool.Drain()
	anyPool.Drain()
	boolPool.Drain()
	boolsPool.Drain()
	intPool.Drain()
	intsPool.Drain()
	uintPool.Drain()
	uintsPool.Drain()
	floatPool.Drain()
	floatsPool.Drain()
	errorPool.Drain()
	errorsPool.Drain()
	timePool.Drain()
	timesPool.Drain()
	uuidPool.Drain()
	uuidsPool.Drain()
	jsonPool.Drain()
}
