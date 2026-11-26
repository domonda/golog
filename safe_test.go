package golog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requireReturns runs f and fails the test if f panics
// or doesn't return within 5 seconds, or 30 seconds with the
// race detector, which makes checking the largest values that
// the limits allow about 20 times slower.
func requireReturns(t *testing.T, f func()) {
	t.Helper()
	timeout := 5 * time.Second
	if raceEnabled {
		timeout = 30 * time.Second
	}
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		f()
	}()
	select {
	case p := <-done:
		require.Nil(t, p, "panicked")
	case <-time.After(timeout):
		t.Fatalf("did not return within %s", timeout)
	}
}

// testPanicError panics in its Error method.
type testPanicError struct{}

func (testPanicError) Error() string { panic("Error boom") }

// testPanicLoggable panics in its Log method after logging an attrib.
type testPanicLoggable struct{}

func (testPanicLoggable) Log(m *Message) {
	m.Str("before", "panic")
	panic("Log boom")
}

// testPanicStringer panics in its String method.
type testPanicStringer struct{}

func (testPanicStringer) String() string { panic("String boom") }

// testPanicJSON panics in its MarshalJSON method.
type testPanicJSON string

func (testPanicJSON) MarshalJSON() ([]byte, error) { panic("MarshalJSON boom") }

// testPanicText panics in its MarshalText method.
type testPanicText int

func (testPanicText) MarshalText() ([]byte, error) { panic("MarshalText boom") }

// testPanicNull panics in its IsNull and IsZero methods.
type testPanicNull struct{ V int }

func (testPanicNull) IsNull() bool { panic("IsNull boom") }

func (testPanicNull) IsZero() bool { panic("IsZero boom") }

// testPanicBytes is a byte slice type that panics in its String method.
type testPanicBytes []byte

func (testPanicBytes) String() string { panic("String boom") }

// TestNeverPanics tests that a panic of a method that golog calls
// on a logged value, like Error, Log, String, MarshalJSON, MarshalText,
// IsNull and IsZero, is logged as error instead of panicking.
func TestNeverPanics(t *testing.T) {
	log, textOut, jsonOut := newTestLogger()
	// golog tags, because omitnull is a golog modifier unknown to encoding/json
	type withNull struct {
		N testPanicNull `golog:"n,omitnull"`
		Z testPanicNull `golog:"z,omitzero"`
	}

	for _, tc := range []struct {
		name string
		log  func(*Message) *Message
		json string
	}{
		{name: "Err", log: func(m *Message) *Message { return m.Err(testPanicError{}) }, json: `"error":"golog: recovered panic in Error method of golog.testPanicError: Error boom"`},
		{name: "Errors", log: func(m *Message) *Message { return m.Errors("e", []error{testPanicError{}, nil}) }, json: `"e":["golog: recovered panic in Error method of golog.testPanicError: Error boom",null]`},
		{name: "Any error", log: func(m *Message) *Message { return m.Any("e", testPanicError{}) }, json: `"e":"golog: recovered panic in Error method of golog.testPanicError: Error boom"`},
		{name: "Any Loggable", log: func(m *Message) *Message { return m.Any("l", testPanicLoggable{}) }, json: `"l":"golog: recovered panic in Log method of golog.testPanicLoggable: Log boom"`},
		{name: "Any MarshalJSON", log: func(m *Message) *Message { return m.Any("j", testPanicJSON("x")) }, json: `"j":"golog: recovered panic in MarshalJSON method of golog.testPanicJSON: MarshalJSON boom"`},
		{name: "Any MarshalText", log: func(m *Message) *Message { return m.Any("t", testPanicText(1)) }, json: `"t":"golog: recovered panic in MarshalText method of golog.testPanicText: MarshalText boom"`},
		{name: "Any String of byte slice", log: func(m *Message) *Message { return m.Any("b", testPanicBytes("x")) }, json: `"b":"golog: recovered panic in String method of golog.testPanicBytes: String boom"`},
		{name: "Any IsNull", log: func(m *Message) *Message { return m.Any("n", testPanicNull{V: 1}) }, json: `"n":{"V":1}`},
		{name: "Any struct with MarshalJSON field", log: func(m *Message) *Message { return m.Any("s", struct{ J testPanicJSON }{"x"}) }, json: `"s":"error while marshalling struct { J golog.testPanicJSON } as JSON for logging: golog: recovered panic while marshalling struct { J golog.testPanicJSON } as JSON: MarshalJSON boom"`},
		{name: "Stringer", log: func(m *Message) *Message { return m.Stringer("s", testPanicStringer{}) }, json: `"s":"golog: recovered panic in String method of golog.testPanicStringer: String boom"`},
		{name: "AsJSON", log: func(m *Message) *Message { return m.AsJSON("j", testPanicJSON("x")) }, json: `"j":"can't log golog.testPanicJSON AsJSON because of: golog: recovered panic while marshalling golog.testPanicJSON as JSON: MarshalJSON boom"`},
		{name: "Loggable", log: func(m *Message) *Message { return m.Loggable(testPanicLoggable{}) }, json: `"before":"panic","error":"golog: recovered panic in Log method of golog.testPanicLoggable: Log boom"`},
		{name: "StructFields IsNull and IsZero", log: func(m *Message) *Message {
			return m.StructFields(withNull{N: testPanicNull{V: 1}, Z: testPanicNull{V: 2}})
		}, json: `"n":{"V":1},"z":{"V":2}`},
		{name: "attribs", log: func(m *Message) *Message {
			return m.Any("a", Attribs{NewError("e", testPanicError{}), NewAny("l", testPanicLoggable{}), NewErrors("es", []error{testPanicError{}})})
		}, json: `"a":{"e":"golog: recovered panic in Error method of golog.testPanicError: Error boom","l":"golog: recovered panic in Log method of golog.testPanicLoggable: Log boom","es":["golog: recovered panic in Error method of golog.testPanicError: Error boom"]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requireReturns(t, func() { tc.log(log.Info("Msg")).Log() })
			assert.Contains(t, jsonOut.String(), `"message":"Msg",`+tc.json+"}\n")
			assert.NotEmpty(t, textOut.String())
			textOut.Reset()
			jsonOut.Reset()
		})
	}

	// The attribs of a sub-logger write the same errors
	requireReturns(t, func() {
		sub := log.With().Err(testPanicError{}).Any("l", testPanicLoggable{}).SubLogger()
		sub.Info("Msg").Log()
		assert.Contains(t, NewError("e", testPanicError{}).ValueString(), "Error boom")
		assert.Contains(t, NewErrors("e", []error{testPanicError{}}).ValueString(), "Error boom")
	})
	assert.Contains(t, jsonOut.String(), `"error":"golog: recovered panic in Error method of golog.testPanicError: Error boom","l":"golog: recovered panic in Log method of golog.testPanicLoggable: Log boom"`)
}

// TestNeverPanics_NilPointerError tests that a nil pointer error,
// whose Error method with value receiver would panic, is logged as null.
func TestNeverPanics_NilPointerError(t *testing.T) {
	log, _, jsonOut := newTestLogger()
	var nilErr *testValueError
	requireReturns(t, func() {
		log.Info("Msg").Err(nilErr).Errors("es", []error{nilErr}).Log()
	})
	assert.Contains(t, jsonOut.String(), `"error":null,"es":[null]}`)
	assert.Equal(t, `"e":null`, string(NewError("e", nilErr).AppendJSON(nil)))
}

// TestNeverHangs tests that values that reference themselves are logged
// with a cycle error instead of being formatted until the stack overflows,
// a fatal error that recover can't catch, also when they reference
// themselves many times.
func TestNeverHangs(t *testing.T) {
	selfSlice := []any{1, nil}
	selfSlice[1] = selfSlice
	selfMap := map[string]any{}
	selfMap["self"] = selfMap
	fanOut := make([]any, 10)
	for i := range fanOut {
		fanOut[i] = fanOut
	}
	cycle := "golog: encountered a cycle via []interface {}"

	log, textOut, jsonOut := newTestLogger()
	requireReturns(t, func() {
		log.Info("Msg").Any("s", selfSlice).Print("p", selfSlice).Print("m", selfMap).Log()
	})
	assert.Contains(t, jsonOut.String(), `"s":[1,[1,"`+cycle+`"]],"p":"`+cycle+`","m":"golog: encountered a cycle via map[string]interface {}"}`)
	assert.Contains(t, textOut.String(), `s=[1,[1,"`+cycle+`"]]`)
	jsonOut.Reset()

	requireReturns(t, func() {
		log.Infof("formatted %v", selfSlice).Log()
		log.Errorf("error %v", selfSlice).Log()
		log.NewLevelWriter(log.Config().InfoLevel()).Printf("printf %v", selfSlice)
	})
	assert.Equal(t, 3, strings.Count(jsonOut.String(), cycle), jsonOut.String())
	jsonOut.Reset()

	// fmt formats a pointer argument to a struct like &{...}
	// with the value that it points to, which can contain itself
	type holder struct{ X []any }
	ptr := &holder{X: []any{nil}}
	ptr.X[0] = ptr.X
	requireReturns(t, func() {
		log.Errorf("error %v", ptr).Log()
		log.NewLevelWriter(log.Config().InfoLevel()).Printf("printf %+v", ptr)
	})
	assert.Equal(t, 2, strings.Count(jsonOut.String(), "golog: encountered a cycle via *golog.holder"), jsonOut.String())
	jsonOut.Reset()

	// The Any attrib ends the cycle at the first repetition,
	// so a slice that contains itself many times doesn't take
	// time and space exponential in the number of times
	requireReturns(t, func() {
		j := NewAny("s", fanOut).AppendJSON(nil)
		assert.Equal(t, 10, strings.Count(string(j), cycle))
		assert.Equal(t, `[1,"`+cycle+`"]`, NewAny("s", selfSlice).ValueString())
		sub := log.With().Any("s", selfSlice).SubLogger()
		sub.Info("Msg").Log()
	})
	assert.Contains(t, jsonOut.String(), `"s":[1,[1,"`+cycle+`"]]`)

	// A self-containing value without a cycle error, like errors.New,
	// is formatted as usual
	assert.Equal(t, "a", safeSprint(errors.New("a")))
}

// TestCheckFormattable tests which values the fmt package would format forever.
func TestCheckFormattable(t *testing.T) {
	selfSlice := []any{nil}
	selfSlice[0] = selfSlice
	pointerToSelf := []any{nil}
	pointerToSelf[0] = &pointerToSelf
	type node struct{ Children []node }
	selfNodes := []node{{}}
	selfNodes[0].Children = selfNodes

	assert.Error(t, checkFormattable(reflect.ValueOf(selfSlice), printVerb, new(valueCheckBudget)), "slice that contains itself")
	assert.Error(t, checkFormattable(reflect.ValueOf(&selfSlice), printVerb, new(valueCheckBudget)), "pointer to slice that contains itself")
	assert.Error(t, checkFormattable(reflect.ValueOf(selfNodes), printVerb, new(valueCheckBudget)), "structs that contain their slice")
	assert.NoError(t, checkFormattable(reflect.ValueOf(pointerToSelf), printVerb, new(valueCheckBudget)), "fmt formats a nested pointer as address")
	assert.NoError(t, checkFormattable(reflect.ValueOf([]any{1, "a", []int{1}}), printVerb, new(valueCheckBudget)))
	assert.NoError(t, checkFormattable(reflect.ValueOf([][]int{{1}}), printVerb, new(valueCheckBudget)))
	assert.NoError(t, checkFormattable(reflect.ValueOf(nil), printVerb, new(valueCheckBudget)))
}

// testCyclicStringer is a slice with a String method that can contain itself.
type testCyclicStringer []any

func (testCyclicStringer) String() string { return "stringer" }

// testSelfPanicError panics with itself in its Error method,
// so formatting the panic value panics again.
type testSelfPanicError struct{}

func (e testSelfPanicError) Error() string { panic(e) }

// testCyclicPanicStringer panics with a slice that contains itself.
type testCyclicPanicStringer struct{}

func (testCyclicPanicStringer) String() string {
	s := []any{nil}
	s[0] = s
	panic(s)
}

// testSelfLoggable is a struct Loggable that logs itself.
type testSelfLoggable struct{ N int }

func (l testSelfLoggable) Log(m *Message) { m.Int("n", l.N).Loggable(l) }

// testSelfAnyLoggable is a struct Loggable that logs itself with Any.
type testSelfAnyLoggable struct{ N int }

func (l testSelfAnyLoggable) Log(m *Message) { m.Any("self", l) }

// testAttribsHolder references itself through Attribs,
// which encoding/json marshals with Attribs.MarshalJSON.
type testAttribsHolder struct{ A Attribs }

// testFormatter formats itself, also when it contains itself.
type testFormatter []any

func (testFormatter) Format(s fmt.State, _ rune) { _, _ = s.Write([]byte("F")) }

// TestNeverPanicsOrHangs_Formatting tests values that used to make
// golog panic or overflow the stack while formatting them.
func TestNeverPanicsOrHangs_Formatting(t *testing.T) {
	cyclicStringer := testCyclicStringer{nil}
	cyclicStringer[0] = cyclicStringer
	selfSlice := []any{nil}
	selfSlice[0] = selfSlice
	var dag any = []any{1}
	for range 40 {
		dag = []any{dag, dag}
	}
	var selfFunc LoggableFunc
	selfFunc = func(m *Message) { m.Any("self", selfFunc) }
	holder := &testAttribsHolder{}
	holder.A = Attribs{NewAny("h", holder)}
	formatter := testFormatter{nil}
	formatter[0] = formatter

	log, _, jsonOut := newTestLogger()
	for name, f := range map[string]func(){
		// fmt doesn't call String for %d and %#v
		"Infof %d of cyclic Stringer":  func() { log.Infof("%d", cyclicStringer).Log() },
		"Infof %#v of cyclic Stringer": func() { log.Infof("%#v", cyclicStringer).Log() },
		// %T and %p don't format the value, which isn't checked
		"Infof %T of DAG":                  func() { log.Infof("%T %p", dag, dag).Log() },
		"Err that panics with itself":      func() { log.Info("Msg").Err(testSelfPanicError{}).Log() },
		"Print of error that panics":       func() { log.Info("Msg").Print("p", testSelfPanicError{}).Log() },
		"Infof of error that panics":       func() { log.Infof("%v", testSelfPanicError{}).Log() },
		"Print of cyclic panic value":      func() { log.Info("Msg").Print("p", testCyclicPanicStringer{}).Log() },
		"Stringer that panics with itself": func() { log.Info("Msg").Stringer("s", testCyclicPanicStringer{}).Log() },
		"Print of reflect.Value":           func() { log.Info("Msg").Print("p", reflect.ValueOf(selfSlice)).Log() },
		"Infof of reflect.Value":           func() { log.Infof("%v", reflect.ValueOf(selfSlice)).Log() },
		"struct Loggable logging itself":   func() { log.Info("Msg").Loggable(testSelfLoggable{N: 1}).Log() },
		"struct Loggable with Any":         func() { log.Info("Msg").Any("l", testSelfAnyLoggable{N: 1}).Log() },
		"LoggableFunc logging itself":      func() { log.Info("Msg").Any("f", selfFunc).Log() },
		"cycle through Attribs.MarshalJSON": func() {
			log.Info("Msg").Any("h", holder).AsJSON("j", holder).Log()
		},
		"nil Request": func() { log.Info("Msg").Request(nil).Log() },
	} {
		t.Run(name, func(t *testing.T) {
			requireReturns(t, f)
			assert.True(t, json.Valid(bytes.TrimSpace(jsonOut.Bytes())), "valid JSON: %s", jsonOut)
			jsonOut.Reset()
		})
	}

	// Any.ValueString writes a value that is no plain number, string or
	// bool like it is logged as JSON, ending a cycle with an error
	assert.Equal(t, `["golog: encountered a cycle via golog.testFormatter"]`, NewAny("f", formatter).ValueString())
}

// testSecret redacts itself with a MarshalJSON method.
type testSecret string

func (testSecret) MarshalJSON() ([]byte, error) { return []byte(`"***"`), nil }

// TestMessage_Any_NestedRedaction tests that Message.Any writes a slice
// nested in a slice as JSON array like the Any attrib, if its elements
// aren't plain numbers, strings or bools, so the MarshalJSON methods of
// its elements redact them, instead of formatting it with the fmt
// package. A slice of plain values is still formatted by fmt.
func TestMessage_Any_NestedRedaction(t *testing.T) {
	log, textOut, jsonOut := newTestLogger()
	log.Info("Msg").
		Any("a", [][]testSecret{{"pw"}}).
		Any("b", []any{[]testSecret{"pw"}}).
		Any("c", [][]int{{1, 2}}).
		Log()
	assert.Contains(t, jsonOut.String(), `"a":[["***"]],"b":[["***"]],"c":["[1 2]"]}`)
	assert.NotContains(t, textOut.String(), "pw")
	assert.Equal(t, `"***"`, NewAny("s", testSecret("pw")).ValueString())
	assert.Equal(t, `["***"]`, NewAny("s", []testSecret{"pw"}).ValueString())
}

// TestTextWriter_WriteError_Panic tests that a panic of an Error method
// in a slice of errors leaves no separator behind in the text output.
func TestTextWriter_WriteError_Panic(t *testing.T) {
	log, textOut, _ := newTestLogger()
	log.Info("Msg").Errors("e", []error{errors.New("a"), testPanicError{}}).Log()
	assert.Contains(t, textOut.String(), "e=[`a`,`golog: recovered panic in Error method of golog.testPanicError: Error boom`]")
}

// TestMessage_Any_UUIDAllocs tests that logging a UUID with Any
// doesn't allocate more than logging it with UUID through reflection.
func TestMessage_Any_UUIDAllocs(t *testing.T) {
	log := NewLogger(NewConfig(&DefaultLevels, AllLevelsActive, NewJSONWriterConfig(io.Discard, nil)))
	uuid := MustParseUUID("a547276f-b02b-4e7d-b67e-c6deb07567da")
	allocs := testing.AllocsPerRun(100, func() { log.Info("Msg").Any("u", uuid).Log() })
	assert.LessOrEqual(t, allocs, 3.0)
}

// raceEnabled is set by race_test.go when the tests run with -race.
var raceEnabled bool

// testIntLoggable logs one int.
type testIntLoggable struct{ N int }

func (l *testIntLoggable) Log(m *Message) { m.Int("n", l.N) }

// testRow has an interface field, so its values are walked
// by checkJSONSize and checkFormattable.
type testRow struct {
	ID  int
	Val any
}

// TestNeverPanicsOrHangs_Allocs tests that the checks for panics
// and cycles don't allocate for common values: a Loggable logged with
// Message.Loggable, and structs with interface fields, which used to be
// copied to the heap for every element of a slice.
func TestNeverPanicsOrHangs_Allocs(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector changes the allocations")
	}
	log := NewLogger(NewConfig(&DefaultLevels, AllLevelsActive, NewJSONWriterConfig(io.Discard, nil)))
	var loggable Loggable = &testIntLoggable{N: 1}
	baseline := testing.AllocsPerRun(100, func() { log.Info("Msg").Int("n", 1).Log() })
	allocs := testing.AllocsPerRun(100, func() { log.Info("Msg").Loggable(loggable).Log() })
	assert.Equal(t, baseline, allocs, "Message.Loggable")

	rows := reflect.ValueOf(make([]testRow, 20))
	var budget valueCheckBudget
	assert.Zero(t, testing.AllocsPerRun(100, func() { _ = checkJSONSize(rows, &budget) }), "checkJSONSize")
	assert.Zero(t, testing.AllocsPerRun(100, func() { _ = checkFormattable(rows, printVerb, &budget) }), "checkFormattable")
}

// testFanOutLoggable is a struct Loggable that logs itself twice.
type testFanOutLoggable struct{ N int }

func (l testFanOutLoggable) Log(m *Message) { m.Any("a", l).Any("b", l) }

// testFanOutPtrLoggable logs pointers to copies of itself twice,
// a new address on every level.
type testFanOutPtrLoggable struct{ N int }

func (l testFanOutPtrLoggable) Log(m *Message) { m.Any("a", &l).Any("b", &l) }

// testTree is a struct Loggable that logs its children,
// which are values of its own type.
type testTree struct {
	N    int
	Kids []testTree
}

func (t testTree) Log(m *Message) { m.Int("n", t.N).Any("kids", t.Kids) }

// testDAG is a struct that references values twice through interfaces.
type testDAG struct{ L, R any }

// TestNeverHangs_FanOut tests that values that reference themselves
// several times end in linear time instead of exponential time,
// and that legitimate deep values are logged completely.
func TestNeverHangs_FanOut(t *testing.T) {
	var fanOutFunc LoggableFunc
	fanOutFunc = func(m *Message) { m.Any("a", fanOutFunc).Any("b", fanOutFunc) }
	holder := &testAttribsHolder{}
	holder.A = Attribs{NewAny("a", holder), NewAny("b", holder)}
	var dag any = 0
	for range 40 {
		dag = testDAG{L: dag, R: dag}
	}

	log, textOut, jsonOut := newTestLogger()
	for name, f := range map[string]func(){
		"struct Loggable":             func() { log.Info("Msg").Loggable(testFanOutLoggable{}).Any("l", testFanOutLoggable{}).Log() },
		"Loggable pointers":           func() { log.Info("Msg").Any("l", testFanOutPtrLoggable{}).Log() },
		"LoggableFunc":                func() { log.Info("Msg").Any("f", fanOutFunc).Log() },
		"Attribs referenced twice":    func() { log.Info("Msg").Any("h", holder).AsJSON("j", holder).Log() },
		"struct DAG in interfaces %T": func() { log.Infof("%T", dag).Log() },
	} {
		t.Run(name, func(t *testing.T) {
			requireReturns(t, f)
			assert.True(t, json.Valid(bytes.TrimSpace(jsonOut.Bytes())), "valid JSON")
			jsonOut.Reset()
		})
	}

	// A tree of struct Loggables 100 deep is logged completely
	tree := testTree{N: 100}
	for n := 99; n >= 0; n-- {
		tree = testTree{N: n, Kids: []testTree{tree}}
	}
	log.Info("Msg").Any("tree", tree).Log()
	assert.Contains(t, jsonOut.String(), `{"n":100,"kids":null}`)
	assert.NotContains(t, jsonOut.String(), "golog:")
	jsonOut.Reset()

	textOut.Reset()
	// fmt.Sprint keeps the spaces between values that are no strings
	log.NewLevelWriter(log.Config().InfoLevel()).Print(1, time.Second, errors.New("a"), errors.New("b"), 2)
	assert.Contains(t, textOut.String(), "1 1s a b 2")
}

// testUser logs only its name, hiding its password.
type testUser struct {
	Name     string
	Password string
}

func (u testUser) Log(m *Message) { m.Str("name", u.Name) }

// testSecretPtr redacts itself with a MarshalJSON method with pointer receiver.
type testSecretPtr struct{ V string }

func (*testSecretPtr) MarshalJSON() ([]byte, error) { return []byte(`"***"`), nil }

// TestNeverLeaks tests that secrets that values hide with their Log,
// Error or MarshalJSON methods are hidden by Any.ValueString and by the
// arrays that the Any attrib writes, which used to show them.
func TestNeverLeaks(t *testing.T) {
	for name, val := range map[string]any{
		"Loggable":             testUser{Name: "a", Password: "pw"},
		"struct field":         struct{ S testSecret }{"pw"},
		"map value":            map[string]testSecret{"k": "pw"},
		"array behind pointer": &[1]testSecretPtr{{V: "pw"}},
		"nested array":         [][1]testSecretPtr{{{V: "pw"}}},
	} {
		t.Run(name, func(t *testing.T) {
			assert.NotContains(t, NewAny("k", val).ValueString(), "pw")
			assert.NotContains(t, string(NewAny("k", val).AppendJSON(nil)), "pw")
		})
	}
}

// TestAttribs_MarshalJSON_DeepCallStack tests that Attribs marshalled by
// encoding/json deep in a call stack are no cycle.
func TestAttribs_MarshalJSON_DeepCallStack(t *testing.T) {
	var recurse func(depth int) ([]byte, error)
	recurse = func(depth int) ([]byte, error) {
		if depth == 0 {
			return json.Marshal(struct{ A Attribs }{Attribs{NewString("k", "v")}})
		}
		return recurse(depth - 1)
	}
	j, err := recurse(2000)
	require.NoError(t, err)
	assert.Equal(t, `{"A":{"k":"v"}}`, string(j))
}

// testItem and testOrder are Loggables, an order logs its many items.
type testItem struct{ ID int }

func (i testItem) Log(m *Message) { m.Int("id", i.ID) }

type testOrder struct{ Items []testItem }

func (o testOrder) Log(m *Message) { m.Int("n", len(o.Items)).Any("items", o.Items) }

// testStringerMap is a map with a String method that can contain itself.
type testStringerMap map[string]any

func (testStringerMap) String() string { return "stringer map" }

// TestNoFalseLimits tests that legitimate values are logged completely:
// a Loggable that logs many Loggables, Attribs that many goroutines
// marshal nested in each other, and the arguments of verbs like %T, %p
// and %v of a Stringer that fmt formats without looking inside them.
func TestNoFalseLimits(t *testing.T) {
	log, _, jsonOut := newTestLogger()

	order := testOrder{Items: make([]testItem, 2000)}
	for i := range order.Items {
		order.Items[i] = testItem{ID: i}
	}
	log.Info("Msg").Any("order", order).Log()
	assert.Contains(t, jsonOut.String(), `{"id":1999}]}`)
	assert.NotContains(t, jsonOut.String(), "golog:")
	jsonOut.Reset()

	shared := Attribs{NewString("k", "v")}
	outer := Attribs{NewAny("w", testAttribsHolder{A: shared})}
	var wg sync.WaitGroup
	var failed atomic.Int64
	for range 8 {
		wg.Go(func() {
			for range 200 {
				_, _ = json.Marshal(testAttribsHolder{A: shared})
			}
		})
		wg.Go(func() {
			for range 200 {
				j, _ := json.Marshal(testAttribsHolder{A: outer})
				if string(j) != `{"A":{"w":{"A":{"k":"v"}}}}` {
					failed.Add(1)
				}
			}
		})
	}
	wg.Wait()
	assert.Zero(t, failed.Load(), "false cycles of Attribs marshalled concurrently")

	var dag any = 0
	for range 40 {
		dag = testDAG{L: dag, R: dag}
	}
	big := make([]any, maxValueCheckNodes+1)
	stringerMap := testStringerMap{}
	stringerMap["self"] = stringerMap
	assert.Equal(t, "golog.testDAG", safeSprintf("%T", dag))
	assert.Equal(t, "[]interface {} true", safeSprintf("%T %t", big, safeSprintf("%p", big) != ""))
	assert.Equal(t, "stringer map", safeSprintf("%v", stringerMap))
	assert.Contains(t, safeSprintf(badFormat("%d"), testStringerMap{"a": testSliceDAG(40)}),
		"golog: golog.testStringerMap has too many values to format")

	// encoding/json would write the DAG in exponential time
	requireReturns(t, func() { log.Info("Msg").Any("dag", dag).Log() })
	assert.Contains(t, jsonOut.String(), "values to marshal as JSON")
	jsonOut.Reset()

	// Values that JSON can't encode are written in Go syntax
	assert.Equal(t, "(1+2i)", NewAny("c", complex(1, 2)).ValueString())
}

// TestFormatArgVerbs tests how fmt formats the arguments
// with the verbs of a format.
func TestFormatArgVerbs(t *testing.T) {
	v := fmtVerb{methods: fmtV}.bit()
	vs := fmtVerb{methods: fmtV, followPointers: true}.bit()
	goSyntax := fmtVerb{methods: fmtGoSyntax}.bit()
	formatter := fmtVerb{methods: fmtFormatter}.bit()
	formatterFollow := fmtVerb{methods: fmtFormatter, followPointers: true}.bit()
	noMethods := fmtVerb{methods: fmtNoMethods}.bit()
	err := errors.New("e")
	for _, tc := range []struct {
		format   string
		args     []any
		wrapErrs bool
		want     []fmtVerbs
	}{
		{format: "%T %p", args: []any{1, &err}, want: []fmtVerbs{0, 0}},
		{format: "%p", args: []any{struct{}{}}, want: []fmtVerbs{noMethods}},
		{format: "%d %v", args: []any{1, 2}, want: []fmtVerbs{formatter, v}},
		{format: "%#v %s", args: []any{1, 2}, want: []fmtVerbs{goSyntax, vs}},
		{format: "%t %x", args: []any{1, 2}, want: []fmtVerbs{formatterFollow, v}},
		{format: "%[2]v %[1]T", args: []any{1, 2}, want: []fmtVerbs{0, v}},
		{format: "%*d %v", args: []any{1, 2}, want: []fmtVerbs{0, formatter}},
		{format: "100%% %v", args: []any{1, 2}, want: []fmtVerbs{v, v}},
		{format: "%v %d %[1]d", args: []any{1, 2}, want: []fmtVerbs{v | formatter, formatter}},
		{format: "%w %w", args: []any{err, 1}, want: []fmtVerbs{noMethods, noMethods}},
		{format: "%w %w", args: []any{err, 1}, wrapErrs: true, want: []fmtVerbs{v, noMethods}},
		// Bad indexes use no argument, and no argument is an extra one
		{format: "%[0]v %d", args: []any{1, 2}, want: []fmtVerbs{formatter, 0}},
		{format: "%[9]v%d", args: []any{1, 2}, want: []fmtVerbs{formatter, 0}},
		{format: "%[+1]v %[ 1]v %[1]2d %d", args: []any{1, 2}, want: []fmtVerbs{formatter, 0}},
		// A width above 1e6 ends the format like in fmt
		{format: "%12345678d %d", args: []any{1, 2}, want: []fmtVerbs{v, v}},
		{format: "%d", args: []any{1, 2, 3}, want: []fmtVerbs{formatter, v, v}},
		// Precisions read from arguments, and an index before a precision
		{format: "%.*d", args: []any{1, 2}, want: []fmtVerbs{0, formatter}},
		{format: "%.[2]*d", args: []any{1, 2, 3}, want: []fmtVerbs{0, 0, formatter}},
		{format: "%[1].2d", args: []any{1}, want: []fmtVerbs{0}},
		// %#w uses only the Format and GoString methods of an error
		{format: "%#w", args: []any{err}, wrapErrs: true, want: []fmtVerbs{fmtVerb{methods: fmtNoMethods, followPointers: true}.bit()}},
	} {
		assert.Equal(t, tc.want, formatArgVerbs(tc.format, tc.args, tc.wrapErrs, nil), tc.format)
	}
}

// testTreeNode is a tree whose children point back to their parent,
// which encoding/json reports as cycle.
type testTreeNode struct {
	Name     string
	Parent   *testTreeNode
	Children []*testTreeNode
}

// testJSONTreeNode is a tree whose back-references to the parent
// are left out by encoding/json, so it marshals without cycle.
type testJSONTreeNode struct {
	Name     string
	Parent   *testJSONTreeNode `json:"-"`
	Children []*testJSONTreeNode
}

// testRedactedData redacts its data with a MarshalJSON method
// with pointer receiver, so encoding/json never looks inside it.
type testRedactedData struct{ Data []any }

func (*testRedactedData) MarshalJSON() ([]byte, error) { return []byte(`"redacted"`), nil }

// TestNeverHangs_JSONPointerCycle tests that a value that references
// itself through pointers, which encoding/json reports as cycle, ends
// early instead of growing the stack by a call for every value until
// the size limit, and that the values that encoding/json leaves out,
// like a json:"-" field, don't count.
func TestNeverHangs_JSONPointerCycle(t *testing.T) {
	// Without cycle detection the check recursed a million calls deep,
	// which overflows this stack limit instead of using 512 MB
	defer debug.SetMaxStack(debug.SetMaxStack(64 << 20))

	root := &testTreeNode{Name: "root"}
	root.Children = []*testTreeNode{{Name: "kid", Parent: root}}
	jsonRoot := &testJSONTreeNode{Name: "root"}
	jsonRoot.Children = []*testJSONTreeNode{{Name: "kid", Parent: jsonRoot}}
	redacted := &struct{ S testRedactedData }{S: testRedactedData{Data: []any{testSliceDAG(40)}}}

	log, _, jsonOut := newTestLogger()
	requireReturns(t, func() {
		log.Info("Msg").Any("tree", root).Any("jsonTree", jsonRoot).Any("redacted", redacted).AsJSON("asJSON", jsonRoot).Log()
	})
	assert.Contains(t, jsonOut.String(), `"tree":"error while marshalling golog.testTreeNode as JSON for logging: golog: encountered a cycle via `)
	assert.Contains(t, jsonOut.String(), `"jsonTree":{"Name":"root","Children":[{"Name":"kid","Children":null}]}`)
	assert.Contains(t, jsonOut.String(), `"redacted":{"S":"redacted"}`)
	assert.Contains(t, jsonOut.String(), `"asJSON":{"Name":"root","Children":[{"Name":"kid","Children":null}]}`)
	assert.Equal(t, `{"Name":"root","Children":[{"Name":"kid","Children":null}]}`, NewAny("t", jsonRoot).ValueString())
}

// testCyclicError is an error whose fields reference itself.
type testCyclicError struct{ ctx map[string]any }

func (testCyclicError) Error() string { return "cyclic error" }

// testCyclicHolder holds a pointer to a value that contains itself.
type testCyclicHolder struct{ P *testCyclicSliceHolder }

type testCyclicSliceHolder struct{ S testCyclicStringer }

// TestNeverHangs_FormatVerbs tests the verbs and argument indexes
// for which fmt formats a value without the methods that end a cycle,
// or follows a pointer nested in it, like %w outside of Errorf,
// invalid argument indexes and %s of a pointer in a struct.
func TestNeverHangs_FormatVerbs(t *testing.T) {
	cyclicErr := testCyclicError{ctx: map[string]any{}}
	cyclicErr.ctx["err"] = cyclicErr
	cyclicStringer := testCyclicStringer{nil}
	cyclicStringer[0] = cyclicStringer
	holder := testCyclicHolder{P: &testCyclicSliceHolder{S: cyclicStringer}}

	log, _, jsonOut := newTestLogger()
	for name, f := range map[string]func(){
		"Infof %w":                 func() { log.Infof(badFormat("failed: %w"), cyclicErr).Log() },
		"Errorf %w of no error":    func() { log.Errorf(badFormat("failed: %w"), cyclicStringer).Log() },
		"Printf %w":                func() { log.NewLevelWriter(log.Config().InfoLevel()).Printf(badFormat("failed: %w"), cyclicErr) },
		"index zero":               func() { log.Infof(badFormat("%[0]v %d"), cyclicStringer).Log() },
		"index out of range":       func() { log.Infof(badFormat("%[9]v%d"), cyclicStringer).Log() },
		"index with sign":          func() { log.Infof(badFormat("%[+1]v %d"), cyclicStringer).Log() },
		"%s of pointer in struct":  func() { log.Infof(badFormat("%s"), holder).Log() },
		"%t of pointer in struct":  func() { log.Infof("%t", []any{holder}).Log() },
		"%v of exponential DAG":    func() { log.Infof("%v", testSliceDAG(40)).Log() },
		"Print of exponential DAG": func() { log.Info("Msg").Print("p", testSliceDAG(40)).Log() },
		"Errorf %#w":               func() { log.Errorf(badFormat("failed: %#w"), cyclicErr).Log() },
		"%.*d precision":           func() { log.Infof(badFormat("%.*d"), 100, cyclicStringer).Log() },
	} {
		t.Run(name, func(t *testing.T) {
			requireReturns(t, f)
			assert.True(t, json.Valid(bytes.TrimSpace(jsonOut.Bytes())), "valid JSON: %s", jsonOut)
			assert.Contains(t, jsonOut.String(), "golog:")
			jsonOut.Reset()
		})
	}

	// Errorf wraps an error with %w, formatted by its Error method
	assert.Equal(t, "failed: cyclic error", safeErrorf("failed: %w", cyclicErr).Error())
	// %s of a pointer in a struct that contains no cycle
	assert.Equal(t, "{%!s(*golog.testItem=&{1})}", safeSprintf(badFormat("%s"), struct{ P *testItem }{&testItem{ID: 1}}))
	// fmt follows a nested pointer once and formats the pointers in it
	// as addresses, so a pointer to itself is no cycle
	next := &testNextNode{}
	next.Next = next
	requireReturns(t, func() {
		assert.NotContains(t, safeSprintf(badFormat("%s"), struct{ P *testNextNode }{next}), "golog:")
	})
	self := []any{nil}
	self[0] = &self
	assert.Equal(t, fmt.Sprintf(badFormat("%s"), &self), safeSprintf(badFormat("%s"), &self))
}

// badFormat returns format as non-constant string,
// so go vet doesn't report the bad formats that tests use.
func badFormat(format string) string { return format }

// testSliceDAG returns slices nested depth deep, each of which contains
// the next one twice, which fmt and encoding/json would write in time
// exponential in depth.
func testSliceDAG(depth int) any {
	var dag any = []any{1}
	for range depth {
		dag = []any{dag, dag}
	}
	return dag
}

// TestNeverHangs_AnyDAG tests that Message.Any and the Any attrib
// end values that reference the same slices many times with an error
// instead of writing them in exponential time.
func TestNeverHangs_AnyDAG(t *testing.T) {
	dag := testSliceDAG(40)
	arrayDAG := any([2]any{1, 2})
	for range 40 {
		arrayDAG = [2]any{arrayDAG, arrayDAG}
	}
	// The values are written until the limit of slices, arrays and maps,
	// each in its own requireReturns, which the race detector slows down
	log, _, jsonOut := newTestLogger()
	requireReturns(t, func() { log.Info("Msg").Any("dag", dag).Log() })
	assert.Contains(t, jsonOut.String(), "slices, arrays and maps")
	assert.True(t, json.Valid(bytes.TrimSpace(jsonOut.Bytes())), "valid JSON")
	jsonOut.Reset()
	requireReturns(t, func() {
		assert.Contains(t, string(NewAny("arrays", arrayDAG).AppendJSON(nil)), "slices, arrays and maps")
	})
	requireReturns(t, func() {
		assert.Contains(t, string(log.With().Any("dag", dag).SubLogger().Attribs().AppendJSON(nil)), "slices, arrays and maps")
	})
}

// TestNoFalseLimits_LargeValues tests that values that are large
// but take linear time to format or marshal are written completely.
func TestNoFalseLimits_LargeValues(t *testing.T) {
	big := make([]any, maxValueCheckNodes+1)
	for i := range big {
		big[i] = i
	}
	assert.NotContains(t, safeSprintf("%v", big), "golog:")
	assert.NotContains(t, safeSprint(map[string]any{"rows": big}), "golog:")
	j, err := safeJSONMarshal(map[string]any{"rows": big}, new(valueCheckBudget))
	require.NoError(t, err)
	assert.True(t, bytes.HasSuffix(j, []byte(`1000000]}`)))

	// A shared slice referenced many times is written for every reference
	shared := []any{1, 2, 3}
	outer := make([]any, 1000)
	for i := range outer {
		outer[i] = shared
	}
	assert.NotContains(t, safeSprintf("%v", outer), "golog:")
}

// TestMessage_LimitsPerMessage tests that the limits of the Log calls
// apply to every log message, so messages can log up to maxNestedLogs
// Loggables each, but not more.
func TestMessage_LimitsPerMessage(t *testing.T) {
	var buf bytes.Buffer
	log := NewLogger(NewConfig(&DefaultLevels, AllLevelsActive, NewJSONWriterConfig(&buf, nil)))
	for range 2 {
		m := log.Info("Msg")
		for range maxNestedLogs / 2 {
			m.Any("k", testItem{ID: 1}).Loggable(testItem{ID: 2})
		}
		m.Log()
	}
	assert.NotContains(t, buf.String(), "golog:")
	buf.Reset()
	m := log.Info("Msg")
	for range maxNestedLogs + 1 {
		m.Any("k", testItem{ID: 1})
	}
	m.Log()
	assert.Contains(t, buf.String(), "Loggables logged inside one logged value or message")
}

// testPanicErrLoggable panics with an error in its Log method.
type testPanicErrLoggable struct{}

func (testPanicErrLoggable) Log(*Message) {
	panic(fmt.Errorf("db closed: %w", errors.New("conn reset")))
}

// TestDescribePanic tests that a panic value is described by its
// error or String method, and a value that is too large to format
// is not described as a cycle.
func TestDescribePanic(t *testing.T) {
	log, _, jsonOut := newTestLogger()
	log.Info("Msg").Loggable(testPanicErrLoggable{}).Log()
	assert.Contains(t, jsonOut.String(), "recovered panic in Log method of golog.testPanicErrLoggable: db closed: conn reset")
	assert.Equal(t, "panic value of type golog.testSelfPanicError", describePanic(testSelfPanicError{}))
	assert.Contains(t, describePanic(testSliceDAG(40)), "values to format")
}

// testHash is a named 16 byte array that is no UUID.
type testHash [16]byte

// TestMessage_Any_NamedArrayNotUUID tests that a named [16]byte type
// that holds no valid UUID is logged as array of numbers.
func TestMessage_Any_NamedArrayNotUUID(t *testing.T) {
	log, _, jsonOut := newTestLogger()
	log.Info("Msg").Any("h", testHash{1, 2, 3}).Log()
	assert.Contains(t, jsonOut.String(), `"h":[1,2,3,0,0,0,0,0,0,0,0,0,0,0,0,0]`)
	assert.Equal(t, `[1,2,3,0,0,0,0,0,0,0,0,0,0,0,0,0]`, string(appendAnyJSON(nil, testHash{1, 2, 3})))
}

// TestLevelWriter_Println_Cycle tests that LevelWriter.Println
// writes a cycle error for a slice that contains itself.
func TestLevelWriter_Println_Cycle(t *testing.T) {
	selfSlice := []any{1, nil}
	selfSlice[1] = selfSlice
	log, _, jsonOut := newTestLogger()
	requireReturns(t, func() {
		log.NewLevelWriter(log.Config().InfoLevel()).Println("a", selfSlice)
	})
	assert.Equal(t, 1, strings.Count(jsonOut.String(), "golog: encountered a cycle via []interface {}"))
}

// testNextNode points to the next node.
type testNextNode struct{ Next *testNextNode }

// TestNeverHangs_Deep tests that values nested deeper than
// maxValueCheckDepth are reported as error instead of exhausting
// the stack, and that values nested less deep are written.
func TestNeverHangs_Deep(t *testing.T) {
	var deep any = 1
	for range maxValueCheckDepth + 10 {
		deep = []any{deep}
	}
	assert.Contains(t, safeSprintf("%v", deep), "nested more than 10000 deep to format")
	_, err := safeJSONMarshal(deep, new(valueCheckBudget))
	assert.ErrorContains(t, err, "nested more than 10000 deep to marshal as JSON")

	var shallow any = 1
	for range 5000 {
		shallow = []any{shallow}
	}
	assert.NotContains(t, safeSprintf("%v", shallow), "golog:")
	_, err = safeJSONMarshal(shallow, new(valueCheckBudget))
	assert.NoError(t, err)
}

// testSharedSlices returns n slices nested in each other,
// whose elements reference the same slice width times.
func testSharedSlices(width, n int) any {
	var shared any = 1
	for range n {
		s := make([]any, width)
		for i := range s {
			s[i] = shared
		}
		shared = s
	}
	return shared
}

// testEmbNode embeds pointers to the next node twice,
// through testEmbLeft and testEmbRight.
type testEmbNode struct {
	testEmbLeft
	testEmbRight
	N int `json:"n"`
}

type testEmbLeft struct{ *testEmbNode }

type testEmbRight struct{ *testEmbNode }

// testEmbSelf embeds a pointer to itself.
type testEmbSelf struct {
	*testEmbSelf
	X int
}

// testPrivParentNode references its parent with an unexported field,
// which encoding/json leaves out.
type testPrivParentNode struct {
	Name   string
	parent *testPrivParentNode
	Kids   []*testPrivParentNode
}

// testEmbInner is embedded unexported by testEmbOuter.
type testEmbInner struct{ D any }

type testEmbOuter struct{ testEmbInner }

// TestNeverHangs_SharedValues tests values that reference the same
// values many times, which would take exponential time or time that is
// the product of their lengths, also through nested slices of numbers,
// several top-level elements, Attribs marshalled by encoding/json and
// embedded pointers of StructFields.
func TestNeverHangs_SharedValues(t *testing.T) {
	// Nested slices of ints whose levels share one slice 8 times
	nested := reflect.ValueOf([]int{1})
	for range 10 {
		level := reflect.MakeSlice(reflect.SliceOf(nested.Type()), 8, 8)
		for i := range 8 {
			level.Index(i).Set(nested)
		}
		nested = level
	}
	ints := nested.Interface()
	// Attribs referencing the same holder twice on every level
	holder := testAttribsHolder{}
	for range 40 {
		holder = testAttribsHolder{A: Attribs{NewAny("a", holder), NewAny("b", holder)}}
	}
	// Embedded pointers to the same node twice on every level
	node := &testEmbNode{}
	for n := 1; n <= 60; n++ {
		node = &testEmbNode{testEmbLeft{node}, testEmbRight{node}, n}
	}
	// 20 references to the same DAG of 2^18 slices
	dags := make([]any, 20)
	for i := range dags {
		dags[i] = testSliceDAG(18)
	}

	log, _, jsonOut := newTestLogger()
	for name, f := range map[string]func(){
		"Any nested ints":        func() { log.Info("Msg").Any("v", ints).Log() },
		"Any struct nested ints": func() { log.Info("Msg").Any("v", struct{ V any }{ints}).Log() },
		"Infof nested ints":      func() { log.Infof("%v", ints).Log() },
		"Any attrib nested ints": func() { log.Info("Msg").Any("v", Attribs{NewAny("v", ints)}).Log() },
		"Attribs as JSON":        func() { log.Info("Msg").Any("h", holder).Log() },
		"Loggable Attribs":       func() { log.Info("Msg").Loggable(holder.A).Log() },
		"StructFields":           func() { log.Info("Msg").StructFields(node).Log() },
		"Any DAG elements":       func() { log.Info("Msg").Any("dags", dags).Log() },
		"Infof shared slices":    func() { log.Infof("%v", testSharedSlices(700, 3)).Log() },
		"Any shared slices":      func() { log.Info("Msg").Any("s", testSharedSlices(700, 3)).Log() },
	} {
		t.Run(name, func(t *testing.T) {
			requireReturns(t, f)
			assert.True(t, json.Valid(bytes.TrimSpace(jsonOut.Bytes())), "valid JSON")
			jsonOut.Reset()
		})
	}
	log.Info("Msg").Any("dags", dags).Log()
	assert.Contains(t, jsonOut.String(), "slices, arrays and maps")
	jsonOut.Reset()
	log.Info("Msg").StructFields(node).Log()
	assert.Equal(t, 1, strings.Count(jsonOut.String(), `"n":0`), "a struct that several embedded pointers point to is logged once")
	jsonOut.Reset()
	assert.Contains(t, safeSprintf("%v", testSharedSlices(700, 3)), "too many values to format")
}

// TestNoFalseLimits_JSON tests that values that encoding/json
// marshals are not reported as cycle or as too large.
func TestNoFalseLimits_JSON(t *testing.T) {
	embSelf := &testEmbSelf{X: 1}
	embSelf.testEmbSelf = embSelf
	parent := &testPrivParentNode{Name: "p"}
	parent.Kids = []*testPrivParentNode{{Name: "k", parent: parent}}
	// A leaf referenced twice deeper than startDetectingCyclesAfter
	leaf := &testTreeNode{Name: "leaf"}
	deep := &testTreeNode{Children: []*testTreeNode{leaf, leaf}}
	for range startDetectingCyclesAfter {
		deep = &testTreeNode{Children: []*testTreeNode{deep}}
	}

	for name, val := range map[string]any{
		"embedded self pointer":  embSelf,
		"unexported parent":      parent,
		"shared deep leaf":       deep,
		"embedded node pointers": &testEmbNode{testEmbLeft{&testEmbNode{N: 1}}, testEmbRight{}, 2},
		"mutual embedding": func() *testMutualA {
			a := &testMutualA{X: 1}
			a.testMutualB = &testMutualB{testMutualA: a, Y: 2}
			return a
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			want, err := json.Marshal(val)
			require.NoError(t, err)
			got, err := safeJSONMarshal(val, new(valueCheckBudget))
			require.NoError(t, err)
			assert.Equal(t, string(want), string(got))
		})
	}

	// An embedded unexported struct is marshalled, also its DAG
	_, err := safeJSONMarshal(testEmbOuter{testEmbInner{D: testSliceDAG(40)}}, new(valueCheckBudget))
	assert.ErrorContains(t, err, "too many values to marshal as JSON")
}

// testGoStringerSlice is a slice with a GoString method that can contain itself.
type testGoStringerSlice []any

func (testGoStringerSlice) GoString() string { return "go stringer" }

// testGoStringerError is an error slice with a GoString method
// that can contain itself.
type testGoStringerError []any

func (testGoStringerError) Error() string    { return "error" }
func (testGoStringerError) GoString() string { return "go stringer error" }

// TestNoFalseLimits_Methods tests that a value that contains itself
// is formatted with the method that fmt uses for the verb.
func TestNoFalseLimits_Methods(t *testing.T) {
	formatter := testFormatter{nil}
	formatter[0] = formatter
	goStringer := testGoStringerSlice{nil}
	goStringer[0] = goStringer
	stringerMap := testStringerMap{}
	stringerMap["self"] = stringerMap
	selfMap := map[string]any{}
	selfMap["self"] = selfMap

	assert.Equal(t, "F", safeSprintf(badFormat("%d"), formatter))
	assert.Equal(t, "go stringer", safeSprintf("%#v", goStringer))
	assert.Equal(t, `"stringer map"`, safeSprintf("%q", stringerMap))
	assert.Equal(t, fmt.Sprintf("%p", selfMap), safeSprintf("%p", selfMap))

	// A reflect.Value is formatted like fmt formats it,
	// also of an unexported field and an invalid one
	cyclicSlice := []any{nil}
	cyclicSlice[0] = cyclicSlice
	cyclicUnexported := reflect.ValueOf(struct{ x []any }{cyclicSlice}).Field(0)
	assert.Contains(t, safeSprint(cyclicUnexported), "golog: encountered a cycle")
	// fmt formats a pointer to an interface as address
	var cyclicAny any = cyclicSlice
	assert.NotContains(t, safeSprint(&cyclicAny), "golog:")
	// %#w uses the GoString method of an error
	goStringerErr := testGoStringerError{nil}
	goStringerErr[0] = goStringerErr
	assert.Equal(t, "go stringer error", safeErrorf(badFormat("%#w"), goStringerErr).Error())

	unexported := reflect.ValueOf(struct{ x int }{42}).Field(0)
	assert.Equal(t, fmt.Sprint(unexported), safeSprint(unexported))
	assert.Equal(t, fmt.Sprint(reflect.Value{}), safeSprint(reflect.Value{}))
}

// TestMessage_LimitsPerWriter tests that the limits apply to every
// log message, and that every writer writes the same value the same way.
func TestMessage_LimitsPerWriter(t *testing.T) {
	rows := make([][]testSecret, maxValueCheckNodes*3/5)
	for i := range rows {
		rows[i] = []testSecret{"x"}
	}
	var buf bytes.Buffer
	log := NewLogger(NewConfig(&DefaultLevels, AllLevelsActive, NewJSONWriterConfig(&buf, nil)))
	log.Info("Msg").Any("a", rows).Log()
	log.Info("Msg").Any("b", rows).Log()
	assert.NotContains(t, buf.String(), "golog:")

	// The text and the JSON writer write the same value
	log, textOut, jsonOut := newTestLogger()
	log.Info("Msg").Any("v", struct{ D any }{testSliceDAG(18)}).Any("r", rows).Log()
	assert.NotContains(t, textOut.String(), "golog:")
	assert.NotContains(t, jsonOut.String(), "golog:")
}

// testNilErrMarshaler returns a typed nil error from its methods.
type testNilErrMarshaler string

type testNilErr struct{ msg string }

func (e *testNilErr) Error() string { return e.msg }

func (testNilErrMarshaler) MarshalJSON() ([]byte, error) {
	var err *testNilErr
	return nil, err
}

type testNilErrTextMarshaler int

func (testNilErrTextMarshaler) MarshalText() ([]byte, error) {
	var err *testNilErr
	return nil, err
}

// TestNeverPanics_MarshalerNilError tests that a typed nil error
// returned by a MarshalJSON or MarshalText method doesn't panic.
func TestNeverPanics_MarshalerNilError(t *testing.T) {
	log, _, jsonOut := newTestLogger()
	requireReturns(t, func() {
		log.Info("Msg").
			Any("j", testNilErrMarshaler("x")).
			Any("t", testNilErrTextMarshaler(1)).
			Any("s", struct{ J testNilErrMarshaler }{"x"}).
			Any("a", Attribs{NewAny("j", testNilErrMarshaler("x")), NewAny("s", struct{ J testNilErrMarshaler }{"x"})}).
			Log()
		_ = NewAny("s", struct{ J testNilErrMarshaler }{"x"}).ValueString()
	})
	assert.True(t, json.Valid(bytes.TrimSpace(jsonOut.Bytes())), "valid JSON: %s", jsonOut)
	assert.Contains(t, jsonOut.String(), `"j":null,"t":null`)
}

// testMutualA and testMutualB embed pointers to each other,
// which encoding/json embeds only once.
type testMutualA struct {
	*testMutualB
	X int
}

type testMutualB struct {
	*testMutualA
	Y int
}

// TestNeverHangs_SharedBudget tests that the checks of the values
// of one log message, and of the arguments of one format, share one
// budget, also for plain slices that reference the same slice.
func TestNeverHangs_SharedBudget(t *testing.T) {
	dags := make([]any, 20)
	for i := range dags {
		dags[i] = struct{ D any }{testSliceDAG(18)}
	}
	ints := make([]int, 100_000)
	sharedInts := make([][]int, 200)
	for i := range sharedInts {
		sharedInts[i] = ints
	}

	log, _, jsonOut := newTestLogger()
	requireReturns(t, func() { log.Info("Msg").Any("dags", dags).Log() })
	assert.Contains(t, jsonOut.String(), "too many values to marshal as JSON")
	jsonOut.Reset()
	requireReturns(t, func() { log.Infof(strings.Repeat("%v", len(dags)), dags...).Log() })
	assert.Contains(t, jsonOut.String(), "too many values to format")
	jsonOut.Reset()
	requireReturns(t, func() { log.Info("Msg").Print("p", dags...).Log() })
	assert.Contains(t, jsonOut.String(), "too many values to format")
	jsonOut.Reset()
	requireReturns(t, func() {
		log.Info("Msg").Any("ints", []any{sharedInts}).Log()
		log.Infof("%v", sharedInts).Log()
	})
	assert.Equal(t, 2, strings.Count(jsonOut.String(), "too many values to format"), jsonOut.String())
	jsonOut.Reset()
}

// testTagNode embeds a pointer to itself with a json tag name, which
// encoding/json marshals as regular field, and references the same
// node with another field.
type testTagNode struct {
	*testTagNode `json:"a"`
	B            *testTagNode `json:"b"`
}

// testSliceHolder is a struct that holds a slice.
type testSliceHolder struct{ B []int }

// TestNeverHangs_FinalReview tests values found by the final review:
// embedded fields that encoding/json marshals as regular fields,
// structs that hold slices referenced many times, a pointer to an
// array that contains itself, arrays nested deeply, and Loggables
// that log to a message they captured with several writers.
func TestNeverHangs_FinalReview(t *testing.T) {
	tagNode := &testTagNode{}
	for range 40 {
		tagNode = &testTagNode{testTagNode: tagNode, B: tagNode}
	}
	ints := make([]int, 1<<20)
	holders := make([]testSliceHolder, 32)
	holderPtrs := make([]any, 32)
	for i := range holders {
		holders[i] = testSliceHolder{B: ints}
		holderPtrs[i] = &holders[i]
	}
	selfArray := &[1]any{}
	selfArray[0] = selfArray
	var deepArray any = 1
	for range 100_000 {
		deepArray = [1]any{deepArray}
	}

	log, _, jsonOut := newTestLogger()
	for name, f := range map[string]func(){
		"tag named embedded field": func() { log.Info("Msg").Any("n", tagNode).Log() },
		"struct holding slice fmt": func() { log.Info("Msg").Print("p", holders).Log() },
		"struct holding slice Any": func() { log.Info("Msg").Any("h", holderPtrs).Log() },
		"pointer to self array":    func() { log.Info("Msg").Any("a", selfArray).Any("b", []any{selfArray}).Log() },
		"deep arrays":              func() { log.Info("Msg").Any("a", deepArray).Log() },
	} {
		t.Run(name, func(t *testing.T) {
			requireReturns(t, f)
			assert.True(t, json.Valid(bytes.TrimSpace(jsonOut.Bytes())), "valid JSON")
			assert.Contains(t, jsonOut.String(), "golog:")
			jsonOut.Reset()
		})
	}
	// A Loggable that logs to a message that it captured, instead of
	// the one that it is passed, writes into the value being written,
	// which is no valid JSON, but ends
	requireReturns(t, func() {
		m := log.Info("Msg")
		var lf LoggableFunc
		lf = func(*Message) { m.Any("k", []any{lf}) }
		m.Loggable(lf).Log()
	})
	jsonOut.Reset()
	log.Info("Msg").Any("a", []any{selfArray}).Log()
	assert.Contains(t, jsonOut.String(), "encountered a cycle via [1]interface {}")
	jsonOut.Reset()
	requireReturns(t, func() {
		assert.Contains(t, string(NewAny("a", selfArray).AppendJSON(nil)), "encountered a cycle")
		assert.Contains(t, string(NewAny("a", holderPtrs).AppendJSON(nil)), "golog:")
	})
}

// TestNoFalseLimits_FinalReview tests values found by the final review
// that are logged completely: a large byte slice in a struct that is
// checked, and []any nested as deep as encoding/json decodes.
func TestNoFalseLimits_FinalReview(t *testing.T) {
	body := struct {
		Body    []byte
		Headers map[string][]string
	}{Body: make([]byte, maxValueCheckElems+1), Headers: map[string][]string{"a": {"b"}}}
	_, err := safeJSONMarshal(body, new(valueCheckBudget))
	assert.NoError(t, err)

	var deep any = 1
	for range maxValueCheckDepth - 10 {
		deep = []any{deep}
	}
	assert.NotContains(t, safeSprintf("%v", deep), "golog:")
	_, err = safeJSONMarshal(deep, new(valueCheckBudget))
	assert.NoError(t, err)
}

// TestNeverHangs_SharedBudgetCalls tests that several AsJSON and Print
// calls of one message share its budget, also for plain nested slices.
func TestNeverHangs_SharedBudgetCalls(t *testing.T) {
	ints := make([]int, 100_000)
	sharedInts := make([][]int, 100)
	for i := range sharedInts {
		sharedInts[i] = ints
	}
	var buf bytes.Buffer
	log := NewLogger(NewConfig(&DefaultLevels, AllLevelsActive, NewJSONWriterConfig(&buf, nil)))
	log.Info("Msg").AsJSON("a", sharedInts).AsJSON("b", sharedInts).Log()
	assert.Contains(t, buf.String(), "too many values to marshal as JSON")
	buf.Reset()
	log.Info("Msg").Print("a", sharedInts).Print("b", sharedInts).Log()
	assert.Contains(t, buf.String(), "too many values to format")
	buf.Reset()
	log.Info("Msg").AsJSON("a", append(sharedInts, sharedInts...)).Log()
	assert.Contains(t, buf.String(), "too many values to marshal as JSON")
}

// testCreds hides its password with its String method only.
type testCreds struct{ User, Pass string }

func (c testCreds) String() string { return c.User + ":***" }

// testNilStringer handles a nil pointer in its String method.
type testNilStringer struct{ s string }

func (p *testNilStringer) String() string {
	if p == nil {
		return "anonymous"
	}
	return p.s
}

// testPtrMarshalString is a named string whose pointer has a MarshalJSON method.
type testPtrMarshalString string

func (*testPtrMarshalString) MarshalJSON() ([]byte, error) { return []byte(`"marshaled"`), nil }

// testArrayHolder is a struct that holds an array.
type testArrayHolder struct{ A [20_000]int }

// TestNeverHangs_PointerShared tests slices, arrays, maps and structs
// that are referenced through pointers many times, which encoding/json
// and the fmt package with %s write for every reference.
func TestNeverHangs_PointerShared(t *testing.T) {
	big := make([]int, 20_000)
	bigArray := &[20_000]int{}
	holder := &testArrayHolder{}
	slicePtrs := make(map[string]*[]int)
	arrayPtrs := make(map[string]*[20_000]int)
	holderPtrs := make(map[string]*testArrayHolder)
	sliceList := make([]*[]int, 1000)
	for i := range 1000 {
		key := strconv.Itoa(i)
		slicePtrs[key] = &big
		arrayPtrs[key] = bigArray
		holderPtrs[key] = holder
		sliceList[i] = &big
	}
	for name, val := range map[string]any{
		"slice pointers":  slicePtrs,
		"array pointers":  arrayPtrs,
		"struct pointers": holderPtrs,
		"slice list":      struct{ L []*[]int }{sliceList},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := safeJSONMarshal(val, new(valueCheckBudget))
			assert.ErrorContains(t, err, "too many values to marshal as JSON")
		})
	}
	assert.Contains(t, safeSprintf(badFormat("%s"), sliceList), "too many values to format")
	// %v formats nested pointers as addresses
	assert.NotContains(t, safeSprintf("%v", sliceList), "golog:")
	// Bytes are written as one base64 string by encoding/json, but as
	// one number per byte by fmt with %v
	bytesList := make([][]byte, 1000)
	for i := range bytesList {
		bytesList[i] = make([]byte, 20_000)
	}
	_, err := safeJSONMarshal(struct{ B [][]byte }{bytesList}, new(valueCheckBudget))
	assert.NoError(t, err)
	assert.Contains(t, safeSprintf("%v", struct{ B [][]byte }{bytesList}), "too many values to format")
}

// TestMessage_Any_NestedStringers tests that a slice nested in a slice,
// whose elements have a String method but no MarshalJSON, MarshalText,
// Error or Log method, is formatted with String by the fmt package, so a
// String method that hides a secret keeps it hidden.
func TestMessage_Any_NestedStringers(t *testing.T) {
	log, _, jsonOut := newTestLogger()
	log.Info("Msg").
		Any("creds", []any{[]testCreds{{"u", "secret"}}}).
		Any("durations", []any{[]time.Duration{time.Second}}).
		Any("named", []testPtrMarshalString{"x"}).
		Log()
	assert.Contains(t, jsonOut.String(), `"creds":["[u:***]"],"durations":["[1s]"],"named":["marshaled"]`)
	assert.NotContains(t, jsonOut.String(), "secret")
}

// TestMessage_Stringer_NilPointer tests that Message.Stringer calls the
// String method of a nil pointer if it has a pointer receiver, which can
// handle nil, and logs null for a value receiver, which would panic.
func TestMessage_Stringer_NilPointer(t *testing.T) {
	log, _, jsonOut := newTestLogger()
	log.Info("Msg").
		Stringer("ptr", (*testNilStringer)(nil)).
		Stringer("val", (*testCreds)(nil)).
		Log()
	assert.Contains(t, jsonOut.String(), `"ptr":"anonymous","val":null`)
}

// TestCheckFormattable_DeepPath tests the slices and maps that
// formatCheck looks up in a map after maxFormatCheckPathLen ones.
func TestCheckFormattable_DeepPath(t *testing.T) {
	shared := []any{1}
	var deep any = []any{shared, shared}
	for range maxFormatCheckPathLen + 20 {
		deep = []any{deep}
	}
	assert.NoError(t, checkFormattable(reflect.ValueOf(deep), printVerb, new(valueCheckBudget)), "a slice shared by siblings below the path limit")

	cyclic := []any{nil}
	cyclic[0] = cyclic
	var deepCycle any = cyclic
	for range maxFormatCheckPathLen + 20 {
		deepCycle = []any{deepCycle}
	}
	assert.ErrorContains(t, checkFormattable(reflect.ValueOf(deepCycle), printVerb, new(valueCheckBudget)), "encountered a cycle")

	// A followed pointer to a slice deeper than the path limit
	var inner any = 1
	for range maxFormatCheckPathLen + 20 {
		inner = []any{inner}
	}
	innerSlice := inner.([]any)
	var outer any = []any{&innerSlice}
	for range maxFormatCheckPathLen + 20 {
		outer = []any{outer}
	}
	assert.NotContains(t, safeSprintf(badFormat("%s"), outer), "golog:")
}

// testKey is a byte array that hides itself with its String method.
type testKey [8]byte

func (testKey) String() string { return "REDACTED" }

// testTokens is a string slice that hides itself with its String method.
type testTokens []string

func (testTokens) String() string { return "REDACTED-TOKENS" }

// testSecretStr is a string that hides itself with its String method.
type testSecretStr string

func (testSecretStr) String() string { return "***" }

// testLogStringer has a Log and a String method.
type testLogStringer struct{ N int }

func (testLogStringer) String() string   { return "string" }
func (l testLogStringer) Log(m *Message) { m.Int("n", l.N) }

// testErrStringer has an Error and a String method.
type testErrStringer struct{}

func (testErrStringer) String() string { return "string" }
func (testErrStringer) Error() string  { return "error" }

// testPtrHolder is a struct with a pointer to an array.
type testPtrHolder struct{ P *[1 << 16]int8 }

// testMarshaledArrays is a struct with arrays that encoding/json
// doesn't write element by element.
type testMarshaledArrays struct {
	X      int
	ID     testTextUUID
	hidden [1 << 16]byte
	Skip   [1 << 16]byte `json:"-"`
}

// TestMessage_Any_NestedStringersRuntime tests that the elements of
// slices nested in slices are formatted with their String method by
// their runtime type, like the fmt package formats them, also for slice
// and array types with a String method, but not elements with a
// MarshalJSON, Log or Error method, which are written with it.
func TestMessage_Any_NestedStringersRuntime(t *testing.T) {
	ts := time.Unix(0, 0).UTC()
	log, _, jsonOut := newTestLogger()
	log.Info("Msg").
		Any("keys", []testKey{{1, 2}}).
		Any("tokens", []any{testTokens{"tok1"}}).
		Any("deep", []any{[]any{testSecretStr("pw1")}}).
		Any("times", []any{[]time.Time{ts}}).
		Any("logs", []any{[]any{testLogStringer{N: 1}}}).
		Any("errs", []any{[]any{testErrStringer{}}}).
		Log()
	assert.Contains(t, jsonOut.String(), `"keys":["REDACTED"],"tokens":["REDACTED-TOKENS"],"deep":[["***"]],"times":[["1970-01-01T00:00:00Z"]],"logs":[[{"n":1}]],"errs":[["error"]]`)
	assert.NotContains(t, jsonOut.String(), "tok1")
	assert.NotContains(t, jsonOut.String(), "pw1")
	// The Any attrib of a sub-logger writes the same
	jsonOut.Reset()
	log.With().Any("deep", []any{[]any{testSecretStr("pw1")}}).SubLogger().Info("Msg").Log()
	assert.Contains(t, jsonOut.String(), `"deep":[["***"]]`)
}

// TestMessage_Stringer_NilPointerPanic tests that Message.Stringer logs
// null for a nil pointer whose String method doesn't handle nil, like
// fmt formats it as <nil>, instead of the recovered panic.
func TestMessage_Stringer_NilPointerPanic(t *testing.T) {
	log, _, jsonOut := newTestLogger()
	log.Info("Msg").Stringer("u", (*url.URL)(nil)).Log()
	assert.Contains(t, jsonOut.String(), `"u":null`)
}

// TestNeverHangs_PointerFields tests structs with pointer fields
// that share an array, which encoding/json writes for every struct,
// and pointers in []any, which fmt formats as addresses with %v.
func TestNeverHangs_PointerFields(t *testing.T) {
	shared := &[1 << 16]int8{}
	holders := make([]testPtrHolder, 300)
	for i := range holders {
		holders[i].P = shared
	}
	_, err := safeJSONMarshal(holders, new(valueCheckBudget))
	assert.ErrorContains(t, err, "too many values to marshal as JSON")

	big := make([]int, 20_000)
	ptrs := make([]any, 1000)
	for i := range ptrs {
		ptrs[i] = &big
	}
	assert.NotContains(t, safeSprintf("%v", ptrs), "golog:")

	nested := &[][100][200]int{{}}
	twoArrays := &struct{ A, B [10_000]int }{}
	byteArray := &[20_000]byte{}
	for name, val := range map[string]any{
		"nested arrays": nested,
		"two arrays":    twoArrays,
		"byte array":    byteArray,
	} {
		t.Run(name, func(t *testing.T) {
			m := make(map[string]any, 1000)
			for i := range 1000 {
				m[strconv.Itoa(i)] = val
			}
			_, err := safeJSONMarshal(m, new(valueCheckBudget))
			assert.ErrorContains(t, err, "too many values to marshal as JSON")
		})
	}
}

// TestNoFalseLimits_ArrayFields tests that arrays that encoding/json
// doesn't write element by element, because they are unexported, left
// out or written by a MarshalText method, are not counted.
func TestNoFalseLimits_ArrayFields(t *testing.T) {
	rows := make([]*testMarshaledArrays, 300)
	for i := range rows {
		rows[i] = &testMarshaledArrays{X: i}
	}
	_, err := safeJSONMarshal(struct{ Rows []*testMarshaledArrays }{rows}, new(valueCheckBudget))
	assert.NoError(t, err)
}

// testPtrRedacted has a String method that shows its secret and
// a MarshalJSON method with pointer receiver that redacts it.
type testPtrRedacted struct{ secret string }

func (r testPtrRedacted) String() string { return "STR:" + r.secret }

func (*testPtrRedacted) MarshalJSON() ([]byte, error) { return []byte(`"redacted"`), nil }

// testPtrRow is a struct with pointer fields to values without arrays.
type testPtrRow struct {
	ID   int
	Name *string
}

// TestRegressions_Master tests values that the final regression review
// found logged worse than before golog v1.4.1.
func TestRegressions_Master(t *testing.T) {
	// A pointer receiver MarshalJSON of an addressable element
	// is used before a value receiver String
	nested := [][]testPtrRedacted{{{secret: "s"}}}
	assert.Equal(t, `"k":[["redacted"]]`, string(NewAny("k", nested).AppendJSON(nil)))

	// Many structs with pointer fields are logged completely
	name := "n"
	rows := make([]testPtrRow, 1_100_000)
	for i := range rows {
		rows[i] = testPtrRow{ID: i, Name: &name}
	}
	_, err := safeJSONMarshal(map[string]any{"rows": rows}, new(valueCheckBudget))
	assert.NoError(t, err)

	// More than maxNestedAttribsMarshals Attribs next to each other
	type holder struct{ L []Attribs }
	h := holder{L: make([]Attribs, 1500)}
	for i := range h.L {
		h.L[i] = Attribs{NewInt("i", int64(i))}
	}
	j, err := Attribs{NewAny("h", h)}.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(j), `{"i":1499}]}`)
	assert.NotContains(t, string(j), "golog:")
}

// TestAttribs_MarshalJSON_Allocs tests that Attribs that can't
// marshal other Attribs are marshalled without tracking them.
func TestAttribs_MarshalJSON_Allocs(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector changes the allocations")
	}
	attribs := Attribs{NewString("s", "v"), NewInt("i", 1), NewAny("a", struct{ X int }{1})}
	appendAllocs := testing.AllocsPerRun(100, func() { _ = attribs.AppendJSON(nil) })
	marshalAllocs := testing.AllocsPerRun(100, func() { _, _ = attribs.MarshalJSON() })
	assert.LessOrEqual(t, marshalAllocs, appendAllocs+1)
}

// testGlobalAttribs are marshalled by testAttribsMarshaler.
var testGlobalAttribs Attribs

// testAttribsMarshaler marshals testGlobalAttribs, which it doesn't hold.
type testAttribsMarshaler struct{ N int }

func (testAttribsMarshaler) MarshalJSON() ([]byte, error) { return json.Marshal(testGlobalAttribs) }

// testNilArrayRow has a pointer to an array that is nil in the rows.
type testNilArrayRow struct {
	ID   int
	Hash *[32]byte
}

// TestRegressions_MasterFinal tests the cases of the final regression
// probe: a MarshalJSON method nested in an Attribs value that marshals
// the Attribs again, nil pointers to arrays in many rows, and Attribs
// with ordinary values that are marshalled without tracking them.
func TestRegressions_MasterFinal(t *testing.T) {
	testGlobalAttribs = Attribs{NewAny("w", []testAttribsMarshaler{{N: 1}})}
	defer func() { testGlobalAttribs = nil }()
	requireReturns(t, func() {
		j, _ := testGlobalAttribs.MarshalJSON()
		assert.Contains(t, string(j), "cycle")
	})

	rows := make([]testNilArrayRow, 600_000)
	_, err := safeJSONMarshal(map[string]any{"rows": rows}, new(valueCheckBudget))
	assert.NoError(t, err)

	for name, val := range map[string]any{
		"time":        time.Now(),
		"slice field": struct{ T []string }{[]string{"a"}},
		"map field":   struct{ M map[string]string }{map[string]string{"a": "b"}},
		"time field":  struct{ W time.Time }{time.Now()},
	} {
		assert.False(t, Attribs{NewAny("v", val)}.mayMarshalAttribs(), name)
	}
	assert.True(t, Attribs{NewAny("v", struct{ A Attribs }{})}.mayMarshalAttribs())
	assert.True(t, Attribs{NewAny("v", []testAttribsMarshaler{})}.mayMarshalAttribs())
}

// testBlob is written by its MarshalJSON method, not its pointer field.
type testBlob struct{ Data *[4096]byte }

func (testBlob) MarshalJSON() ([]byte, error) { return []byte(`"blob"`), nil }

// testGlobalStringerAttribs are marshalled by testAttribsStringer.
var testGlobalStringerAttribs Attribs

// testAttribsStringer marshals testGlobalStringerAttribs in its String method.
type testAttribsStringer struct{}

func (testAttribsStringer) String() string {
	j, _ := json.Marshal(testGlobalStringerAttribs)
	return string(j)
}

// testWide is a type graph where every level references
// the next one twice, which a type walk visits once.
type (
	testWide0 struct{ A, B *testWide1 }
	testWide1 struct{ A, B *testWide2 }
	testWide2 struct{ A, B *testWide3 }
	testWide3 struct{ A, B *testWide4 }
	testWide4 struct{ A, B *testWide5 }
	testWide5 struct{ A, B *testWide6 }
	testWide6 struct{ A, B *testWide7 }
	testWide7 struct{ A, B *testWide8 }
	testWide8 struct{ A, B *testWide9 }
	testWide9 struct {
		A, B *testWide10
	}
	testWide10 struct{ A, B *testWide11 }
	testWide11 struct{ A, B *testWide12 }
	testWide12 struct{ A, B *testWide13 }
	testWide13 struct{ A, B *testWide14 }
	testWide14 struct{ A, B *testWide15 }
	testWide15 struct{ A, B *testWide16 }
	testWide16 struct{ A, B *testWide17 }
	testWide17 struct{ A, B *testWide18 }
	testWide18 struct{ A, B *testWide19 }
	testWide19 struct{ A, B *testWide20 }
	testWide20 struct{ A, B *testWide21 }
	testWide21 struct{ A, B *testWide22 }
	testWide22 struct{ A, B *testWide23 }
	testWide23 struct{ A, B *testWide24 }
	testWide24 struct{ A, B *testWide25 }
	testWide25 struct{ X int }
)

// TestRegressions_MasterLast tests the cases of the last review:
// structs written by their MarshalJSON method with pointer fields,
// a String method that marshals the Attribs holding its value, and
// a large type graph that is walked once per type.
func TestRegressions_MasterLast(t *testing.T) {
	blob := &[4096]byte{}
	blobs := make(map[string]any, 5000)
	for i := range 5000 {
		blobs[strconv.Itoa(i)] = testBlob{Data: blob}
	}
	_, err := safeJSONMarshal(blobs, new(valueCheckBudget))
	assert.NoError(t, err)

	testGlobalStringerAttribs = Attribs{NewAny("s", [][]testAttribsStringer{{{}}})}
	defer func() { testGlobalStringerAttribs = nil }()
	requireReturns(t, func() { _, _ = testGlobalStringerAttribs.MarshalJSON() })

	requireReturns(t, func() { _ = typeMayMarshalAttribs(reflect.TypeFor[*testWide0]()) })
}
