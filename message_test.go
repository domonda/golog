package golog

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/golog/mempool"
)

func newTestConfig(textOut, jsonOut io.Writer) Config {
	format := &Format{
		TimestampFormat: "2006-01-02 15:04:05",
		TimestampKey:    "time",
		LevelKey:        "level",
		PrefixFmt:       "%s: %s",
		MessageKey:      "message",
	}

	textWriterConfig := NewTextWriterConfig(textOut, format, NoColorizer)
	jsonWriterConfig := NewJSONWriterConfig(jsonOut, format)

	return NewConfig(&DefaultLevels, AllLevelsActive, textWriterConfig, jsonWriterConfig)
}

// newTextTestConfig returns a Config with the DefaultLevels, all levels active
// and a TextWriterConfig with the default format and colorizer writing to buf.
func newTextTestConfig() (config Config, buf *bytes.Buffer) {
	buf = bytes.NewBuffer(nil)
	return NewConfig(&DefaultLevels, AllLevelsActive, NewTextWriterConfig(buf, nil, nil)), buf
}

func newTestLogger() (log *Logger, textOut, jsonOut *bytes.Buffer) {
	textOut = bytes.NewBuffer(nil)
	jsonOut = bytes.NewBuffer(nil)
	return NewLogger(newTestConfig(textOut, jsonOut)), textOut, jsonOut
}

func newTestLoggerWithPrefix(prefix string) (log *Logger, textOut, jsonOut *bytes.Buffer) {
	textOut = bytes.NewBuffer(nil)
	jsonOut = bytes.NewBuffer(nil)
	return NewLoggerWithPrefix(newTestConfig(textOut, jsonOut), prefix), textOut, jsonOut
}

// tf is a per-field spec for [buildTestStruct].
type tf struct {
	name  string
	tag   string
	value any
}

// buildTestStruct assembles an instance of a runtime-generated struct from
// the given field specs. Tests use it to put golog-specific json tag
// modifiers (redact, omitnull, etc.) on a struct without triggering static
// analyzers that inspect AST-level struct tag literals (go vet's structtag
// and staticcheck's SA5008). Tags live in runtime [reflect.StructField]
// values, so no analyzer has anything to parse statically.
//
// Each field's type is taken from its value's dynamic type, so pass a zero
// value of the intended field type (e.g. `Timestamp{}` or `(*string)(nil)`)
// when the field should start zero.
func buildTestStruct(fields ...tf) any {
	specs := make([]reflect.StructField, len(fields))
	for i, f := range fields {
		specs[i] = reflect.StructField{
			Name: f.name,
			Type: reflect.TypeOf(f.value),
			Tag:  reflect.StructTag(f.tag),
		}
	}
	v := reflect.New(reflect.StructOf(specs)).Elem()
	for i, f := range fields {
		v.Field(i).Set(reflect.ValueOf(f.value))
	}
	return v.Interface()
}

func checkOutput(t *testing.T, textOut, jsonOut *bytes.Buffer, numMessages int, exptectedTextLine, exptectedJSONLine string) {
	t.Helper()

	textLines := strings.Split(textOut.String(), "\n")
	assert.Len(t, textLines, numMessages+1, "strings.Split created empty last line")
	assert.Equal(t, "", textLines[len(textLines)-1], "strings.Split created empty last line")
	for _, line := range textLines[:numMessages] {
		assert.Equal(t, exptectedTextLine, line)
	}

	jsonLines := strings.Split(jsonOut.String(), "\n")
	assert.Len(t, jsonLines, numMessages+1, "strings.Split created empty last line")
	assert.Equal(t, "", jsonLines[len(jsonLines)-1], "strings.Split created empty last line")
	for _, line := range jsonLines[:numMessages] {
		assert.Equal(t, exptectedJSONLine, line)
		jsonObj := []byte(strings.TrimSuffix(line, ","))
		assert.True(t, json.Valid(jsonObj), "valid JSON message")
	}
}

func TestMessage(t *testing.T) {
	timestamp, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
	ctx := ContextWithTimestamp(context.Background(), timestamp)

	DrainAllMemPools()
	log, textOut, jsonOut := newTestLogger()
	infoLevel := log.Config().InfoLevel()
	mempoolOutput := strings.Builder{}
	mempool.RegisterCallbacksWriterForTest(t, &mempoolOutput)
	numMessages := 3

	for i := range numMessages {
		fmt.Fprintf(&mempoolOutput, "%d:\n", i)
		log.NewMessage(ctx, infoLevel, "My log message").
			Exec(writeMessage).
			Log()
		assert.Zero(t, mempool.NumOutstanding())
	}

	checkOutput(t, textOut, jsonOut, numMessages, exptectedTextMessage, exptectedJSONMessage)

	exptectedMempoolOutput := `0:
Allocated *golog.TextWriter
Allocated *golog.JSONWriter
Allocated *golog.Message
Returned *golog.TextWriter
Returned *golog.JSONWriter
Returned *golog.Message
1:
Reused *golog.TextWriter
Reused *golog.JSONWriter
Reused *golog.Message
Returned *golog.TextWriter
Returned *golog.JSONWriter
Returned *golog.Message
2:
Reused *golog.TextWriter
Reused *golog.JSONWriter
Reused *golog.Message
Returned *golog.TextWriter
Returned *golog.JSONWriter
Returned *golog.Message
`
	if !mempool.RaceEnabled { // The race detector makes sync.Pool drop items instead of reusing them
		assert.Equal(t, exptectedMempoolOutput, mempoolOutput.String())
	}
}

func TestMessageSubLogger(t *testing.T) {
	timestamp, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
	ctx := ContextWithTimestamp(context.Background(), timestamp)

	t.Run("Sub-logger", func(t *testing.T) {
		DrainAllMemPools()
		log, textOut, jsonOut := newTestLogger()
		infoLevel := log.Config().InfoLevel()
		mempoolOutput := strings.Builder{}
		mempool.RegisterCallbacksWriterForTest(t, &mempoolOutput)
		numMessages := 3

		subLog := log.With().
			Str("SuperStr", "SuperStr").
			Strs("SuperStrs", []string{"A", "B", "C"}).
			IntPtr("SuperNilInt", nil).
			SubLogger()
		for i := range numMessages {
			fmt.Fprintf(&mempoolOutput, "%d:\n", i)
			subLog.NewMessage(ctx, infoLevel, "My log message").
				Exec(writeMessage).
				Log()
		}

		checkOutput(t, textOut, jsonOut, numMessages, exptectedTextMessageSub, exptectedJSONMessageSub)

		fmt.Fprintf(&mempoolOutput, "RemoveAttribs:\n")
		subLog.RemoveAttribs()
		// No outstanding mempool items after writing messages
		// and removeing the attribs from the sub-logger
		assert.Zero(t, mempool.NumOutstanding())

		exptectedMempoolOutput := `Allocated *golog.Message
Allocated *golog.String
Allocated []golog.Attrib len:0 cap:16
Allocated *golog.Strings
Allocated *golog.Nil
Returned *golog.Message
0:
Allocated *golog.TextWriter
Allocated *golog.JSONWriter
Reused *golog.Message
Allocated []golog.Attrib len:3 cap:16
Allocated *golog.String
Allocated *golog.Strings
Allocated *golog.Nil
Returned *golog.TextWriter
Returned *golog.JSONWriter
Returned *golog.String
Returned *golog.Strings
Returned *golog.Nil
Returned []golog.Attrib
Returned *golog.Message
1:
Reused *golog.TextWriter
Reused *golog.JSONWriter
Reused *golog.Message
Reused []golog.Attrib len:3 cap:3
Reused *golog.String
Reused *golog.Strings
Reused *golog.Nil
Returned *golog.TextWriter
Returned *golog.JSONWriter
Returned *golog.String
Returned *golog.Strings
Returned *golog.Nil
Returned []golog.Attrib
Returned *golog.Message
2:
Reused *golog.TextWriter
Reused *golog.JSONWriter
Reused *golog.Message
Reused []golog.Attrib len:3 cap:3
Reused *golog.String
Reused *golog.Strings
Reused *golog.Nil
Returned *golog.TextWriter
Returned *golog.JSONWriter
Returned *golog.String
Returned *golog.Strings
Returned *golog.Nil
Returned []golog.Attrib
Returned *golog.Message
RemoveAttribs:
Returned *golog.String
Returned *golog.Strings
Returned *golog.Nil
Returned []golog.Attrib
`
		if !mempool.RaceEnabled { // The race detector makes sync.Pool drop items instead of reusing them
			assert.Equal(t, exptectedMempoolOutput, mempoolOutput.String())
		}
	})

	t.Run("Sub-sub-logger", func(t *testing.T) {
		DrainAllMemPools()
		log, textOut, jsonOut := newTestLogger()
		infoLevel := log.Config().InfoLevel()
		mempoolOutput := strings.Builder{}
		mempool.RegisterCallbacksWriterForTest(t, &mempoolOutput)
		numMessages := 2

		subLog := log.With().
			UUID("RequestID", MustParseUUID("62d38a15-8fc2-4520-b768-9d5d08d2c498")).
			SubLogger()
		subSubLog := subLog.With().
			Str("SuperStr", "SuperStr").
			Strs("SuperStrs", []string{"A", "B", "C"}).
			IntPtr("SuperNilInt", nil).
			UUID("RequestID", MustParseUUID("46f70c54-043e-476e-8e69-06b0a593a57a")). // Different from the parent, ignored
			SubLogger()
		subSubSubLog := subSubLog.With().
			IntPtr("SuperNilInt", new(int)). // Different from the parent, ignored
			SubLogger()
		for i := range numMessages {
			fmt.Fprintf(&mempoolOutput, "%d:\n", i)
			subSubSubLog.NewMessage(ctx, infoLevel, "My log message").
				Exec(writeMessage).
				Log()
		}

		checkOutput(t, textOut, jsonOut, numMessages, exptectedTextMessageSubSub, exptectedJSONMessageSubSub)

		fmt.Fprintf(&mempoolOutput, "RemoveAttribs:\n")
		subSubSubLog.RemoveAttribs()
		subSubLog.RemoveAttribs()
		subLog.RemoveAttribs()
		// No outstanding mempool items after writing messages
		// and removeing the attribs from the sub-loggers
		assert.Zero(t, mempool.NumOutstanding())

		exptectedMempoolOutput := `Allocated *golog.Message
Allocated *golog.UUID
Allocated []golog.Attrib len:0 cap:16
Returned *golog.Message
Allocated []golog.Attrib len:1 cap:16
Allocated *golog.UUID
Reused *golog.Message
Allocated *golog.String
Allocated *golog.Strings
Allocated *golog.Nil
Returned *golog.Message
Allocated []golog.Attrib len:4 cap:16
Allocated *golog.UUID
Allocated *golog.String
Allocated *golog.Strings
Allocated *golog.Nil
Reused *golog.Message
Returned *golog.Message
0:
Allocated *golog.TextWriter
Allocated *golog.JSONWriter
Reused *golog.Message
Allocated []golog.Attrib len:4 cap:16
Allocated *golog.UUID
Allocated *golog.String
Allocated *golog.Strings
Allocated *golog.Nil
Returned *golog.TextWriter
Returned *golog.JSONWriter
Returned *golog.UUID
Returned *golog.String
Returned *golog.Strings
Returned *golog.Nil
Returned []golog.Attrib
Returned *golog.Message
1:
Reused *golog.TextWriter
Reused *golog.JSONWriter
Reused *golog.Message
Reused []golog.Attrib len:4 cap:4
Reused *golog.UUID
Reused *golog.String
Reused *golog.Strings
Reused *golog.Nil
Returned *golog.TextWriter
Returned *golog.JSONWriter
Returned *golog.UUID
Returned *golog.String
Returned *golog.Strings
Returned *golog.Nil
Returned []golog.Attrib
Returned *golog.Message
RemoveAttribs:
Returned *golog.UUID
Returned *golog.String
Returned *golog.Strings
Returned *golog.Nil
Returned []golog.Attrib
Returned *golog.UUID
Returned *golog.String
Returned *golog.Strings
Returned *golog.Nil
Returned []golog.Attrib
Returned *golog.UUID
Returned []golog.Attrib
`
		// t.Fatal(mempoolOutput.String()) // Debug mempoolOutput
		if !mempool.RaceEnabled { // The race detector makes sync.Pool drop items instead of reusing them
			assert.Equal(t, exptectedMempoolOutput, mempoolOutput.String())
		}
	})
}

func TestMessageSubLoggerContext(t *testing.T) {
	timestamp, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
	ctx := ContextWithTimestamp(context.Background(), timestamp)

	t.Run("Sub-logger-context", func(t *testing.T) {
		DrainAllMemPools()
		log, textOut, jsonOut := newTestLogger()
		infoLevel := log.Config().InfoLevel()
		mempoolOutput := strings.Builder{}
		mempool.RegisterCallbacksWriterForTest(t, &mempoolOutput)
		numMessages := 3

		subLog, ctx := log.With().
			Str("SuperStr", "SuperStr").
			Strs("SuperStrs", []string{"A", "B", "C"}).
			IntPtr("SuperNilInt", nil).
			SubLoggerContext(ctx)
		for i := range numMessages {
			fmt.Fprintf(&mempoolOutput, "%d:\n", i)
			subLog.NewMessage(ctx, infoLevel, "My log message").
				Exec(writeMessage).
				Log()
		}

		checkOutput(t, textOut, jsonOut, numMessages, exptectedTextMessageSub, exptectedJSONMessageSub)

		fmt.Fprintf(&mempoolOutput, "RemoveAttribs:\n")
		subLog.RemoveAttribs()
		// 4 outstanding mempool items that have been added to the context
		// and not returned to the pool
		assert.Equal(t, 4, mempool.NumOutstanding())

		exptectedMempoolOutput := `Allocated *golog.Message
Allocated *golog.String
Allocated []golog.Attrib len:0 cap:16
Allocated *golog.Strings
Allocated *golog.Nil
Allocated []golog.Attrib len:3 cap:16
Allocated *golog.String
Allocated *golog.Strings
Allocated *golog.Nil
Returned *golog.Message
0:
Allocated *golog.TextWriter
Allocated *golog.JSONWriter
Reused *golog.Message
Allocated []golog.Attrib len:3 cap:16
Allocated *golog.String
Allocated *golog.Strings
Allocated *golog.Nil
Returned *golog.TextWriter
Returned *golog.JSONWriter
Returned *golog.String
Returned *golog.Strings
Returned *golog.Nil
Returned []golog.Attrib
Returned *golog.Message
1:
Reused *golog.TextWriter
Reused *golog.JSONWriter
Reused *golog.Message
Reused []golog.Attrib len:3 cap:3
Reused *golog.String
Reused *golog.Strings
Reused *golog.Nil
Returned *golog.TextWriter
Returned *golog.JSONWriter
Returned *golog.String
Returned *golog.Strings
Returned *golog.Nil
Returned []golog.Attrib
Returned *golog.Message
2:
Reused *golog.TextWriter
Reused *golog.JSONWriter
Reused *golog.Message
Reused []golog.Attrib len:3 cap:3
Reused *golog.String
Reused *golog.Strings
Reused *golog.Nil
Returned *golog.TextWriter
Returned *golog.JSONWriter
Returned *golog.String
Returned *golog.Strings
Returned *golog.Nil
Returned []golog.Attrib
Returned *golog.Message
RemoveAttribs:
Returned *golog.String
Returned *golog.Strings
Returned *golog.Nil
Returned []golog.Attrib
`
		// t.Fatal(mempoolOutput.String())
		if !mempool.RaceEnabled { // The race detector makes sync.Pool drop items instead of reusing them
			assert.Equal(t, exptectedMempoolOutput, mempoolOutput.String())
		}
	})
}

func writeMessage(message *Message) {
	uuid := MustParseUUID("b14882b9-bfdd-45a4-9c84-1d717211c050")
	uuids := [][16]byte{
		MustParseUUID("fab60526-bf52-4ec2-9db3-f5860250de5c"),
		MustParseUUID("78adb219-460c-41e9-ac39-12d4d0420aa0"),
	}

	message.
		Nil("Nil").
		Bool("True", true).
		Bool("False", false).
		Bools("Bools", []bool{true, false, true}).
		Int("Int", -123).
		Ints("Ints", []int{-1, 0, 1, 123}).
		Int8("Int8", -123).
		Int8s("Int8s", []int8{-1, 0, 1, 123}).
		Int16("Int16", -123).
		Int16s("Int16s", []int16{-1, 0, 1, 123}).
		Int32("Int32", -123).
		Int32s("Int32s", []int32{-1, 0, 1, 123}).
		Int64("Int64", -123).
		Int64s("Int64s", []int64{-1, 0, 1, 123}).
		Uint("Uint", 123).
		Uints("Uints", []uint{0, 1, 123}).
		Uint8("Uint8", 123).
		Uint8s("Uint8s", []uint8{0, 1, 123}).
		Uint16("Uint16", 123).
		Uint16s("Uint16s", []uint16{0, 1, 123}).
		Uint32("Uint32", 123).
		Uint32s("Uint32s", []uint32{0, 1, 123}).
		Uint64("Uint64", 123).
		Uint64s("Uint64s", []uint64{0, 1, 123}).
		Float32("Float32", -1.5).
		Float32s("Float32s", []float32{-1.5, 0, float32(math.NaN()), float32(math.Inf(+1)), float32(math.Inf(-1))}).
		Float("Float", 123).
		Floats("Floats", []float64{-1.9999, 0, math.NaN(), math.Inf(+1), math.Inf(-1)}).
		Str("Str", "Hello\n\"World\"!").
		Strs("Strs", []string{"A", "B", "C"}).
		Err(errors.New("this is an error!")).
		Error("Error", errors.New("this is an error!")).
		Errors("Errors", []error{errors.New(`error "A"`), errors.New(`error "B"`)}).
		Print("PrintSingle", "one arg").
		Print("PrintMulti", false, 123, 0.5, "string", errors.New("error")).
		UUID("UUID", uuid).
		UUIDs("UUIDs", uuids).
		JSON("JSON", []byte(`[{"a": 1, "b":[2,3],"c" : null,"d":{"x":1.5}} , null]`)).
		JSON("InvalidJSON", []byte(`"a":1`))
}

const (
	exptectedTextMessage       = exptectedTextMessageInto + exptectedTextMessageValues
	exptectedTextMessageSub    = exptectedTextMessageInto + exptectedTextMessageSuperValues + exptectedTextMessageValues
	exptectedTextMessageSubSub = exptectedTextMessageInto + exptectedTextMessageSuperSuperValues + exptectedTextMessageSuperValues + exptectedTextMessageValues

	exptectedTextMessageInto = `2006-01-02 15:04:05 |INFO | My log message`

	exptectedTextMessageSuperSuperValues = ` RequestID=62d38a15-8fc2-4520-b768-9d5d08d2c498`
	exptectedTextMessageSuperValues      = ` SuperStr="SuperStr" SuperStrs=["A","B","C"] SuperNilInt=nil`

	exptectedTextMessageValues = ` ` +
		`Nil=nil ` +
		`True=true ` +
		`False=false ` +
		`Bools=[true,false,true] ` +
		`Int=-123 ` +
		`Ints=[-1,0,1,123] ` +
		`Int8=-123 ` +
		`Int8s=[-1,0,1,123] ` +
		`Int16=-123 ` +
		`Int16s=[-1,0,1,123] ` +
		`Int32=-123 ` +
		`Int32s=[-1,0,1,123] ` +
		`Int64=-123 ` +
		`Int64s=[-1,0,1,123] ` +
		`Uint=123 ` +
		`Uints=[0,1,123] ` +
		`Uint8=123 ` +
		`Uint8s=[0,1,123] ` +
		`Uint16=123 ` +
		`Uint16s=[0,1,123] ` +
		`Uint32=123 ` +
		`Uint32s=[0,1,123] ` +
		`Uint64=123 ` +
		`Uint64s=[0,1,123] ` +
		`Float32=-1.5 ` +
		`Float32s=[-1.5,0,NaN,+Inf,-Inf] ` +
		`Float=123 ` +
		`Floats=[-1.9999,0,NaN,+Inf,-Inf] ` +
		`Str="Hello\n\"World\"!" ` +
		`Strs=["A","B","C"] ` +
		`error=` + "`" + `this is an error!` + "`" + ` ` +
		`Error=` + "`" + `this is an error!` + "`" + ` ` +
		"Errors=[`error \"A\"`,`error \"B\"`] " +
		`PrintSingle="one arg" ` +
		`PrintMulti=["false","123","0.5","string","error"] ` +
		`UUID=b14882b9-bfdd-45a4-9c84-1d717211c050 ` +
		`UUIDs=[fab60526-bf52-4ec2-9db3-f5860250de5c,78adb219-460c-41e9-ac39-12d4d0420aa0] ` +
		`JSON=[{"a":1,"b":[2,3],"c":null,"d":{"x":1.5}},null] ` +
		"InvalidJSON=`" + `"a":1` + "`"
)

const (
	exptectedJSONMessage       = exptectedJSONMessageIntro + exptectedJSONMessageValus
	exptectedJSONMessageSub    = exptectedJSONMessageIntro + exptectedJSONMessageSuperValues + exptectedJSONMessageValus
	exptectedJSONMessageSubSub = exptectedJSONMessageIntro + exptectedJSONMessageSuperSuperValues + exptectedJSONMessageSuperValues + exptectedJSONMessageValus

	exptectedJSONMessageIntro = `{` +
		`"time":"2006-01-02 15:04:05","level":"INFO","message":"My log message",`

	exptectedJSONMessageSuperSuperValues = `"RequestID":"62d38a15-8fc2-4520-b768-9d5d08d2c498",`
	exptectedJSONMessageSuperValues      = `"SuperStr":"SuperStr","SuperStrs":["A","B","C"],"SuperNilInt":null,`

	exptectedJSONMessageValus = `"Nil":null,` +
		`"True":true,` +
		`"False":false,` +
		`"Bools":[true,false,true],` +
		`"Int":-123,` +
		`"Ints":[-1,0,1,123],` +
		`"Int8":-123,` +
		`"Int8s":[-1,0,1,123],` +
		`"Int16":-123,` +
		`"Int16s":[-1,0,1,123],` +
		`"Int32":-123,` +
		`"Int32s":[-1,0,1,123],` +
		`"Int64":-123,` +
		`"Int64s":[-1,0,1,123],` +
		`"Uint":123,` +
		`"Uints":[0,1,123],` +
		`"Uint8":123,` +
		`"Uint8s":[0,1,123],` +
		`"Uint16":123,` +
		`"Uint16s":[0,1,123],` +
		`"Uint32":123,` +
		`"Uint32s":[0,1,123],` +
		`"Uint64":123,` +
		`"Uint64s":[0,1,123],` +
		`"Float32":-1.5,` +
		`"Float32s":[-1.5,0,"NaN","+Inf","-Inf"],` +
		`"Float":123,` +
		`"Floats":[-1.9999,0,"NaN","+Inf","-Inf"],` +
		`"Str":"Hello\n\"World\"!",` +
		`"Strs":["A","B","C"],` +
		`"error":"this is an error!",` +
		`"Error":"this is an error!",` +
		`"Errors":["error \"A\"","error \"B\""],` +
		`"PrintSingle":"one arg",` +
		`"PrintMulti":["false","123","0.5","string","error"],` +
		`"UUID":"b14882b9-bfdd-45a4-9c84-1d717211c050",` +
		`"UUIDs":["fab60526-bf52-4ec2-9db3-f5860250de5c","78adb219-460c-41e9-ac39-12d4d0420aa0"],` +
		`"JSON":[{"a":1,"b":[2,3],"c":null,"d":{"x":1.5}},null],` +
		`"InvalidJSON":"\"a\":1"` +
		"}"
)

func TestMessage_Any(t *testing.T) {
	timestamp, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
	ctx := ContextWithTimestamp(context.Background(), timestamp)

	log, textOut, jsonOut := newTestLogger()

	textMsg := `2006-01-02 15:04:05 |INFO | Msg`

	log.NewMessage(ctx, log.Config().InfoLevel(), "Msg").
		Any("int", -100).
		Log()
	assert.Equal(t, fmt.Sprintf("%s %s\n", textMsg, `int=-100`), textOut.String())
	textOut.Reset()
	jsonOut.Reset()

	var (
		uuid    = MustParseUUID("b14882b9-bfdd-45a4-9c84-1d717211c050")
		uuidNil [16]byte
	)

	log.NewMessage(ctx, log.Config().InfoLevel(), "Msg").
		Any("uuid", uuid).
		Any("uuidNil", uuidNil).
		Log()
	assert.Equal(t, fmt.Sprintf("%s %s\n", textMsg, `uuid=b14882b9-bfdd-45a4-9c84-1d717211c050 uuidNil=nil`), textOut.String())
	textOut.Reset()
	jsonOut.Reset()
}

func TestMessage_SubLoggerContext(t *testing.T) {
	timestamp, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
	ctx := ContextWithTimestamp(context.Background(), timestamp)

	uuid := MustParseUUID("a547276f-b02b-4e7d-b67e-c6deb07567da")
	uuid2 := MustParseUUID("064c6bc6-3ec1-4cda-83e7-67815af25a7f")

	log, textOut, jsonOut := newTestLoggerWithPrefix("pkg")
	infoLevel := log.Config().InfoLevel()

	log, ctx = log.With().
		UUID("uuid", uuid).
		SubLoggerContext(ctx)
	log.NewMessage(ctx, infoLevel, "Msg").
		UUID("uuid", uuid2). // Will be ignored because a "uuid" value is already in the sub-logger
		Log()

	textMsg := `2006-01-02 15:04:05 |INFO | pkg: Msg uuid=a547276f-b02b-4e7d-b67e-c6deb07567da` + "\n"
	jsonMsg := `{"time":"2006-01-02 15:04:05","level":"INFO","message":"pkg: Msg","uuid":"a547276f-b02b-4e7d-b67e-c6deb07567da"}` + "\n"

	assert.Equal(t, textMsg, textOut.String())
	assert.Equal(t, jsonMsg, jsonOut.String())
	textOut.Reset()
	jsonOut.Reset()

	log.NewMessage(ctx, infoLevel, "Msg").
		Ctx(ctx).            // Same as above but with ctx that also holds the values in addition to sub-logger
		UUID("uuid", uuid2). // Will be ignored because a "uuid" value is already in the sub-logger
		Log()

	assert.Equal(t, textMsg, textOut.String())
	assert.Equal(t, jsonMsg, jsonOut.String())
	textOut.Reset()
	jsonOut.Reset()

	log.NewMessage(ctx, infoLevel, "Msg").
		Ctx(context.Background()). // Same as above but with empty context
		UUID("uuid", uuid2).       // Will be ignored because a "uuid" value is already in the sub-logger
		Log()

	assert.Equal(t, textMsg, textOut.String())
	assert.Equal(t, jsonMsg, jsonOut.String())
	textOut.Reset()
	jsonOut.Reset()

	{
		uuid3 := MustParseUUID("0ecabfaf-132c-4552-a272-64b43e05dc28")

		log, ctx := log.With().
			UUID("uuid", uuid3). // Does still not overwrite with uuid3
			SubLoggerContext(ctx)
		log.NewMessage(ctx, infoLevel, "Msg").
			UUID("uuid", uuid2). // Will be ignored because a "uuid" value is already in the sub-logger
			Log()

		textMsg := `2006-01-02 15:04:05 |INFO | pkg: Msg uuid=a547276f-b02b-4e7d-b67e-c6deb07567da` + "\n"
		jsonMsg := `{"time":"2006-01-02 15:04:05","level":"INFO","message":"pkg: Msg","uuid":"a547276f-b02b-4e7d-b67e-c6deb07567da"}` + "\n"

		assert.Equal(t, textMsg, textOut.String())
		assert.Equal(t, jsonMsg, jsonOut.String())
		textOut.Reset()
		jsonOut.Reset()

		log.NewMessage(ctx, infoLevel, "Msg").
			Ctx(ctx).            // Same as above but with ctx that also holds the values in addition to sub-logger
			UUID("uuid", uuid2). // Will be ignored because a "uuid" value is already in the sub-logger
			Log()

		assert.Equal(t, textMsg, textOut.String())
		assert.Equal(t, jsonMsg, jsonOut.String())
		textOut.Reset()
		jsonOut.Reset()
	}
}

// TestMessage_IsAttribRecorder tests that only a Message created by
// Logger.With records attribs, and that a message whose writers all filtered
// out its level is not mistaken for one. Such a message has no writers
// either, but it is owned by the caller, who still logs it: SubLogger would
// fabricate a sub-logger from its attribs and put the message back into the
// mempool, so the Log call of the caller would put the same message back a
// second time and the mempool could hand it to two callers at once.
func TestMessage_IsAttribRecorder(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	// All levels are active at the config, but the writer filters out Info,
	// so an Info message is created without any writer
	config := NewConfig(
		&DefaultLevels,
		AllLevelsActive,
		NewTextWriterConfig(buf, nil, nil, LevelFilterOut(DefaultLevels.Info)),
	)
	log := NewLogger(config)

	recorder := log.With()
	require.True(t, recorder.IsAttribRecorder(), "message created by Logger.With")

	filtered := log.Info("filtered out")
	require.NotNil(t, filtered, "the level is active at the config")
	require.Empty(t, filtered.writers, "every writer filtered out the level")
	require.False(t, filtered.IsAttribRecorder(), "message created by a level method")

	// Must return the logger unchanged instead of a sub-logger built from
	// the attribs of the message that the caller still owns
	require.Same(t, log, filtered.SubLogger(), "SubLogger of a message that records nothing")

	ctx := ContextWithAttribs(context.Background(), NewString("ctxStr", "ctx"))
	subLog, subCtx := filtered.SubLoggerContext(ctx)
	require.Same(t, log, subLog, "SubLoggerContext of a message that records nothing")
	require.Equal(t, AttribsFromContext(ctx), AttribsFromContext(subCtx), "ctx attribs are passed through")

	// The message was not put back into the mempool, so it is still usable
	filtered.Str("str", "str").Log()
	require.Empty(t, buf.String(), "nothing is written for a filtered out level")

	recorder.SubLogger() // Put the recorder message back into the mempool
}

// TestMessage_SubLoggerContext_CtxAttribs tests that the sub-logger and the
// context returned by SubLoggerContext carry the same union of attribs:
// the attribs recorded at the message plus those from the passed context.
// The attribs from the context stay context attribs of the sub-logger, so
// that it still logs them when used with an unrelated context, but loses
// them to an attrib with the same key from a fresher context.
func TestMessage_SubLoggerContext_CtxAttribs(t *testing.T) {
	timestamp, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")

	log, textOut, jsonOut := newTestLoggerWithPrefix("pkg")
	infoLevel := log.Config().InfoLevel()

	ctx := ContextWithAttribs(
		context.Background(),
		NewString("ctxStr", "ctx"),
		NewString("bothStr", "ctx"), // Shadowed by the message attrib with the same key
	)

	subLog, subCtx := log.With().
		Str("msgStr", "msg").
		Str("bothStr", "msg").
		SubLoggerContext(ctx)

	// The message attribs become logger attribs, the attribs from the ctx
	// that they don't shadow become the ctx attribs of the sub-logger,
	// and the returned context holds the union of both in the same order.
	expectedAttribs := Attribs{
		NewString("msgStr", "msg"),
		NewString("bothStr", "msg"),
	}
	expectedCtxAttribs := Attribs{
		NewString("ctxStr", "ctx"),
	}
	expectedUnion := Attribs{
		NewString("msgStr", "msg"),
		NewString("bothStr", "msg"),
		NewString("ctxStr", "ctx"),
	}
	assert.Equal(t, expectedAttribs, subLog.Attribs(), "sub-logger attribs")
	assert.Equal(t, expectedCtxAttribs, subLog.CtxAttribs(), "sub-logger ctx attribs")
	assert.Equal(t, expectedUnion, AttribsFromContext(subCtx), "attribs of returned context")

	// Logging with a context without attribs must still log the ctx attribs
	// because the sub-logger inherited them.
	subLog.NewMessageAt(context.Background(), timestamp, infoLevel, "Msg").Log()

	textMsg := `2006-01-02 15:04:05 |INFO | pkg: Msg msgStr="msg" bothStr="msg" ctxStr="ctx"` + "\n"
	jsonMsg := `{"time":"2006-01-02 15:04:05","level":"INFO","message":"pkg: Msg","msgStr":"msg","bothStr":"msg","ctxStr":"ctx"}` + "\n"
	assert.Equal(t, textMsg, textOut.String())
	assert.Equal(t, jsonMsg, jsonOut.String())
	textOut.Reset()
	jsonOut.Reset()

	// Logging with the returned context must not repeat any attrib
	subLog.NewMessageAt(subCtx, timestamp, infoLevel, "Msg").Log()

	assert.Equal(t, textMsg, textOut.String())
	assert.Equal(t, jsonMsg, jsonOut.String())
	textOut.Reset()
	jsonOut.Reset()

	// The attribs of the returned context must be owned by it and not
	// freed together with the attribs of the sub-logger.
	subLog.RemoveAttribs()
	assert.Nil(t, subLog.Attribs(), "attribs removed")
	assert.Nil(t, subLog.CtxAttribs(), "ctx attribs removed")
	assert.Equal(t, expectedUnion, AttribsFromContext(subCtx), "attribs of returned context after Logger.RemoveAttribs")

	// A message without recorded attribs must still pass
	// the ctx attribs on to the sub-logger
	t.Run("no message attribs", func(t *testing.T) {
		subLog, subCtx := log.With().SubLoggerContext(ctx)

		expectedCtxAttribs := Attribs{
			NewString("ctxStr", "ctx"),
			NewString("bothStr", "ctx"),
		}
		assert.Nil(t, subLog.Attribs(), "sub-logger attribs")
		assert.Equal(t, expectedCtxAttribs, subLog.CtxAttribs(), "sub-logger ctx attribs")
		assert.Equal(t, expectedCtxAttribs, AttribsFromContext(subCtx), "attribs of returned context")

		subLog.NewMessageAt(context.Background(), timestamp, infoLevel, "Msg").Log()

		textMsg := `2006-01-02 15:04:05 |INFO | pkg: Msg ctxStr="ctx" bothStr="ctx"` + "\n"
		jsonMsg := `{"time":"2006-01-02 15:04:05","level":"INFO","message":"pkg: Msg","ctxStr":"ctx","bothStr":"ctx"}` + "\n"
		assert.Equal(t, textMsg, textOut.String())
		assert.Equal(t, jsonMsg, jsonOut.String())
		textOut.Reset()
		jsonOut.Reset()
	})

	// The attribs inherited from a ctx are logged after the attribs of the
	// context that the message is logged with, so a fresher attrib with the
	// same key wins over the inherited one instead of being shadowed by a
	// snapshot of a context that the sub-logger may have outlived.
	t.Run("fresher ctx takes precedence over inherited ctx attribs", func(t *testing.T) {
		subLog, _ := log.With().Str("msgStr", "msg").SubLoggerContext(ctx)

		otherCtx := ContextWithAttribs(
			context.Background(),
			NewString("ctxStr", "other"),   // Wins over the inherited ctxStr
			NewString("otherStr", "other"), // Not inherited, so logged as well
		)
		subLog.NewMessageAt(otherCtx, timestamp, infoLevel, "Msg").Log()

		textMsg := `2006-01-02 15:04:05 |INFO | pkg: Msg msgStr="msg" ctxStr="other" otherStr="other" bothStr="ctx"` + "\n"
		jsonMsg := `{"time":"2006-01-02 15:04:05","level":"INFO","message":"pkg: Msg","msgStr":"msg","ctxStr":"other","otherStr":"other","bothStr":"ctx"}` + "\n"
		assert.Equal(t, textMsg, textOut.String())
		assert.Equal(t, jsonMsg, jsonOut.String())
		textOut.Reset()
		jsonOut.Reset()
	})

	// Once logged, an inherited ctx attrib blocks later attribs with the same
	// key like a logger attrib does, so neither Message.Ctx with the returned
	// or another context nor a field method writes its key a second time.
	t.Run("logged inherited ctx attribs are not repeated", func(t *testing.T) {
		subLog, subCtx := log.With().Str("msgStr", "msg").SubLoggerContext(ctx)

		otherCtx := ContextWithAttribs(context.Background(), NewString("ctxStr", "other"))
		subLog.NewMessageAt(context.Background(), timestamp, infoLevel, "Msg").
			Ctx(subCtx).
			Ctx(otherCtx).
			Str("bothStr", "field").
			Str("fieldStr", "field").
			Log()

		textMsg := `2006-01-02 15:04:05 |INFO | pkg: Msg msgStr="msg" ctxStr="ctx" bothStr="ctx" fieldStr="field"` + "\n"
		jsonMsg := `{"time":"2006-01-02 15:04:05","level":"INFO","message":"pkg: Msg","msgStr":"msg","ctxStr":"ctx","bothStr":"ctx","fieldStr":"field"}` + "\n"
		assert.Equal(t, textMsg, textOut.String())
		assert.Equal(t, jsonMsg, jsonOut.String())
		textOut.Reset()
		jsonOut.Reset()
	})

	// The attribs recorded at the message are logger attribs of the
	// sub-logger, so they win over an attrib with the same key
	// from any context the sub-logger is used with.
	t.Run("message attribs take precedence over every ctx", func(t *testing.T) {
		subLog, _ := log.With().Str("msgStr", "msg").SubLoggerContext(ctx)

		otherCtx := ContextWithAttribs(context.Background(), NewString("msgStr", "other"))
		subLog.NewMessageAt(otherCtx, timestamp, infoLevel, "Msg").Log()

		textMsg := `2006-01-02 15:04:05 |INFO | pkg: Msg msgStr="msg" ctxStr="ctx" bothStr="ctx"` + "\n"
		jsonMsg := `{"time":"2006-01-02 15:04:05","level":"INFO","message":"pkg: Msg","msgStr":"msg","ctxStr":"ctx","bothStr":"ctx"}` + "\n"
		assert.Equal(t, textMsg, textOut.String())
		assert.Equal(t, jsonMsg, jsonOut.String())
		textOut.Reset()
		jsonOut.Reset()
	})

	// An attrib of a sub-logger shadows an inherited ctx attrib with the same
	// key, so the inherited one must not be kept: it would never be logged
	// and Logger.CtxAttribs would report an attrib that has no effect.
	t.Run("logger attribs replace inherited ctx attribs", func(t *testing.T) {
		subLog, _ := log.With().Str("msgStr", "msg").SubLoggerContext(ctx)
		require.Len(t, subLog.CtxAttribs(), 2, "ctxStr and bothStr inherited from the ctx")

		for _, shadowing := range []struct {
			name   string
			subLog *Logger
		}{
			{"Message.SubLogger", subLog.With().Str("ctxStr", "sub").SubLogger()},
			{"Logger.WithClonedAttribs", subLog.WithClonedAttribs(NewString("ctxStr", "sub"))},
		} {
			t.Run(shadowing.name, func(t *testing.T) {
				assert.Equal(t,
					Attribs{NewString("msgStr", "msg"), NewString("ctxStr", "sub")},
					shadowing.subLog.Attribs(),
					"logger attribs",
				)
				assert.Equal(t,
					Attribs{NewString("bothStr", "ctx")},
					shadowing.subLog.CtxAttribs(),
					"only the not shadowed ctx attrib is inherited",
				)

				shadowing.subLog.NewMessageAt(context.Background(), timestamp, infoLevel, "Msg").Log()

				textMsg := `2006-01-02 15:04:05 |INFO | pkg: Msg msgStr="msg" ctxStr="sub" bothStr="ctx"` + "\n"
				assert.Equal(t, textMsg, textOut.String())
				textOut.Reset()
				jsonOut.Reset()
			})
		}
	})

	// A sub-logger of a sub-logger created from a context must keep the
	// inherited ctx attribs, and the context returned for it must have them
	// too, so that both still carry the same union of attribs.
	t.Run("chained SubLoggerContext", func(t *testing.T) {
		subLog, _ := log.With().Str("msgStr", "msg").SubLoggerContext(ctx)

		otherCtx := ContextWithAttribs(context.Background(), NewString("otherStr", "other"))
		subSubLog, subSubCtx := subLog.With().Str("subStr", "sub").SubLoggerContext(otherCtx)

		assert.Equal(t,
			Attribs{NewString("msgStr", "msg"), NewString("subStr", "sub")},
			subSubLog.Attribs(),
			"sub-sub-logger attribs",
		)
		assert.Equal(t,
			Attribs{NewString("otherStr", "other"), NewString("ctxStr", "ctx"), NewString("bothStr", "ctx")},
			subSubLog.CtxAttribs(),
			"inherited ctx attribs of the sub-sub-logger",
		)
		assert.Equal(t,
			Attribs{
				NewString("msgStr", "msg"), NewString("subStr", "sub"),
				NewString("otherStr", "other"), NewString("ctxStr", "ctx"), NewString("bothStr", "ctx"),
			},
			AttribsFromContext(subSubCtx),
			"attribs of returned context",
		)

		subSubLog.NewMessageAt(context.Background(), timestamp, infoLevel, "Msg").Log()

		textMsg := `2006-01-02 15:04:05 |INFO | pkg: Msg msgStr="msg" subStr="sub" otherStr="other" ctxStr="ctx" bothStr="ctx"` + "\n"
		assert.Equal(t, textMsg, textOut.String())
		textOut.Reset()
		jsonOut.Reset()
	})
}

func TestMessage_SubContext(t *testing.T) {
	log, _, _ := newTestLogger()

	ctx := log.With().
		Str("str", "A").
		SubContext(context.Background())

	ctx = log.With().
		Str("str", "B"). // B shadows A
		SubContext(ctx)

	values := AttribsFromContext(ctx)
	if values.Len() != 1 {
		t.Fatalf("expected 1 values, got %d", values.Len())
	}
	expected := NewString("str", "B")
	got := values.Get("str")
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %#v, got %#v", expected, got)
	}
}

func TestMessage_Ctx(t *testing.T) {
	timestamp, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")

	log, textOut, jsonOut := newTestLoggerWithPrefix("pkg")
	infoLevel := log.Config().InfoLevel()

	ctx := log.With().
		Int("int", 1).
		SubContext(context.Background())

	log.NewMessageAt(context.Background(), timestamp, infoLevel, "Msg").
		Ctx(ctx). // Logs int=1
		Log()

	textMsg := `2006-01-02 15:04:05 |INFO | pkg: Msg int=1` + "\n"
	jsonMsg := `{"time":"2006-01-02 15:04:05","level":"INFO","message":"pkg: Msg","int":1}` + "\n"

	assert.Equal(t, textMsg, textOut.String())
	assert.Equal(t, jsonMsg, jsonOut.String())
	textOut.Reset()
	jsonOut.Reset()

	log.NewMessageAt(context.Background(), timestamp, infoLevel, "Msg").
		Ctx(ctx).      // Logs int=1
		Int("int", 2). // Dropped because int was already written from the ctx
		Log()

	textMsg = `2006-01-02 15:04:05 |INFO | pkg: Msg int=1` + "\n"
	jsonMsg = `{"time":"2006-01-02 15:04:05","level":"INFO","message":"pkg: Msg","int":1}` + "\n"

	assert.Equal(t, textMsg, textOut.String())
	assert.Equal(t, jsonMsg, jsonOut.String())
	textOut.Reset()
	jsonOut.Reset()

	ctx = log.With().
		Int("int", 3). // Overwrites int value in ctx
		SubContext(ctx)

	log.NewMessageAt(context.Background(), timestamp, infoLevel, "Msg").
		Ctx(ctx).      // Logs int=3
		Int("int", 4). // Dropped because int was already written from the ctx
		Log()

	textMsg = `2006-01-02 15:04:05 |INFO | pkg: Msg int=3` + "\n"
	jsonMsg = `{"time":"2006-01-02 15:04:05","level":"INFO","message":"pkg: Msg","int":3}` + "\n"

	assert.Equal(t, textMsg, textOut.String())
	assert.Equal(t, jsonMsg, jsonOut.String())
	textOut.Reset()
	jsonOut.Reset()
}

// TestMessage_Ctx_DropsWrittenKeys tests that no key is written twice once
// it was written from the context of the message or from a context passed
// to Message.Ctx: a later Message.Ctx or Message method with the same key
// is dropped, while new keys are still written.
func TestMessage_Ctx_DropsWrittenKeys(t *testing.T) {
	timestamp, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
	log, textOut, jsonOut := newTestLoggerWithPrefix("pkg")
	infoLevel := log.Config().InfoLevel()

	ctx1 := ContextWithAttribs(context.Background(), NewString("requestID", "R1"), NewInt("n", 1))
	ctx2 := ContextWithAttribs(context.Background(), NewString("requestID", "R2"), NewString("user", "U2"))

	for _, tc := range []struct {
		name string
		log  func(*Message) *Message
		ctx  context.Context
		text string
		json string
	}{
		{
			name: "message ctx then same ctx",
			ctx:  ctx1,
			log:  func(m *Message) *Message { return m.Ctx(ctx1) },
			text: `requestID="R1" n=1`,
			json: `"requestID":"R1","n":1`,
		},
		{
			name: "message ctx then field with ctx key",
			ctx:  ctx1,
			log:  func(m *Message) *Message { return m.Str("requestID", "X").Int("n", 2).Str("new", "N") },
			text: `requestID="R1" n=1 new="N"`,
			json: `"requestID":"R1","n":1,"new":"N"`,
		},
		{
			name: "message ctx then other ctx",
			ctx:  ctx1,
			log:  func(m *Message) *Message { return m.Ctx(ctx2).Str("user", "X").Ctx(ctx2) },
			text: `requestID="R1" n=1 user="U2"`,
			json: `"requestID":"R1","n":1,"user":"U2"`,
		},
		{
			name: "Ctx without message ctx then field",
			ctx:  context.Background(),
			log:  func(m *Message) *Message { return m.Ctx(ctx1).Str("requestID", "X").Ctx(ctx2) },
			text: `requestID="R1" n=1 user="U2"`,
			json: `"requestID":"R1","n":1,"user":"U2"`,
		},
		{
			name: "Ctx with empty ctx",
			ctx:  context.Background(),
			log:  func(m *Message) *Message { return m.Ctx(context.Background()).Str("a", "A") },
			text: `a="A"`,
			json: `"a":"A"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			DrainAllMemPools()
			mempool.RegisterCallbacksWriterForTest(t, io.Discard)

			tc.log(log.NewMessageAt(tc.ctx, timestamp, infoLevel, "Msg")).Log()

			assert.Equal(t, `2006-01-02 15:04:05 |INFO | pkg: Msg `+tc.text+"\n", textOut.String())
			assert.Equal(t, `{"time":"2006-01-02 15:04:05","level":"INFO","message":"pkg: Msg",`+tc.json+"}\n", jsonOut.String())
			// Recording the keys of a further ctx clones pooled attribs,
			// which the message must return to the mempool
			assert.Zero(t, mempool.NumOutstanding(), "every pooled attrib and slice was returned")
			textOut.Reset()
			jsonOut.Reset()
		})
	}

	// A message recording attribs for a sub-logger records the ctx attribs
	// like attribs of its own and still drops repeated keys
	t.Run("attrib recorder", func(t *testing.T) {
		subLog := log.With().Ctx(ctx1).Str("requestID", "X").Ctx(ctx2).SubLogger()
		assert.Equal(t, Attribs{NewString("requestID", "R1"), NewInt("n", 1), NewString("user", "U2")}, subLog.Attribs())
		subLog.RemoveAttribs()
	})
}

// TestMessage_Any_Time tests that a time.Time logged with Message.Any is
// written like with Message.Time and Message.Times: using the TimeFormat and
// Location configured for the writer, and null for the zero time, instead of
// the RFC3339Nano UTC string of time.Time.MarshalJSON.
func TestMessage_Any_Time(t *testing.T) {
	loc := time.FixedZone("UTC+2", 2*60*60)
	format := &Format{TimestampFormat: "2006-01-02 15:04:05", MessageKey: "message", TimeFormat: "2006-01-02 15:04:05.000 -0700", Location: loc}
	textOut, jsonOut := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	log := NewLogger(NewConfig(&DefaultLevels, AllLevelsActive,
		NewTextWriterConfig(textOut, format, NoColorizer),
		NewJSONWriterConfig(jsonOut, format),
	))
	timestamp, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
	infoLevel := log.Config().InfoLevel()
	tm := time.Date(2024, 1, 2, 3, 4, 5, 6000000, time.UTC)

	for _, tc := range []struct {
		name     string
		any      func(*Message) *Message
		expected func(*Message) *Message
		json     string
	}{
		{
			name:     "time.Time",
			any:      func(m *Message) *Message { return m.Any("t", tm) },
			expected: func(m *Message) *Message { return m.Time("t", tm) },
			json:     `"t":"2024-01-02 05:04:05.006 +0200"`,
		},
		{
			name:     "*time.Time",
			any:      func(m *Message) *Message { return m.Any("t", &tm) },
			expected: func(m *Message) *Message { return m.Time("t", tm) },
			json:     `"t":"2024-01-02 05:04:05.006 +0200"`,
		},
		{
			name:     "zero time.Time",
			any:      func(m *Message) *Message { return m.Any("t", time.Time{}) },
			expected: func(m *Message) *Message { return m.Time("t", time.Time{}) },
			json:     `"t":null`,
		},
		{
			name:     "[]time.Time",
			any:      func(m *Message) *Message { return m.Any("t", []time.Time{tm, tm.Add(time.Second)}) },
			expected: func(m *Message) *Message { return m.Times("t", []time.Time{tm, tm.Add(time.Second)}) },
			json:     `"t":["2024-01-02 05:04:05.006 +0200","2024-01-02 05:04:06.006 +0200"]`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.expected(log.NewMessageAt(context.Background(), timestamp, infoLevel, "Msg")).Log()
			expectedText, expectedJSON := textOut.String(), jsonOut.String()
			textOut.Reset()
			jsonOut.Reset()

			tc.any(log.NewMessageAt(context.Background(), timestamp, infoLevel, "Msg")).Log()
			assert.Equal(t, expectedText, textOut.String(), "text like the typed method")
			assert.Equal(t, expectedJSON, jsonOut.String(), "JSON like the typed method")
			assert.Contains(t, jsonOut.String(), tc.json)
			textOut.Reset()
			jsonOut.Reset()
		})
	}
}

// TestMessage_Any_Bytes tests that bytes logged with Message.Any are written
// like with Message.StrBytes: as string if they are valid UTF-8, else base64
// encoded, instead of as a slice of numbers.
type testNamedBytes []byte

// Byte slice types with a format of their own, which must be used
// instead of their bytes, because it can redact a secret
type (
	testSecretJSONBytes  []byte // MarshalJSON
	testSecretErrorBytes []byte // error
	testSecretLogBytes   []byte // Loggable
)

func (testSecretJSONBytes) MarshalJSON() ([]byte, error) { return []byte(`"***"`), nil }
func (testSecretErrorBytes) Error() string               { return "***" }
func (testSecretLogBytes) Log(m *Message)                { m.Str("secret", "***") }

// testPtrSecretBytes redacts itself with a String method with pointer
// receiver, which a value of the type doesn't have
type testPtrSecretBytes []byte

func (*testPtrSecretBytes) String() string { return "***" }

// testPtrSecretUser logs a pointer to its redacting password field
type testPtrSecretUser struct{ pw testPtrSecretBytes }

func (u *testPtrSecretUser) Log(m *Message) { m.Any("pw", &u.pw) }

// testCycleA and testCycleB point to each other
type (
	testCycleA *testCycleB
	testCycleB *testCycleA
)

// testNullableBytes is null when empty
type testNullableBytes []byte

func (b testNullableBytes) IsNull() bool { return len(b) == 0 }

// testNullableID has an IsNull method with value receiver,
// which panics when called on a nil pointer
type testNullableID int

func (id testNullableID) IsNull() bool { return id == 0 }

func TestMessage_Any_Bytes(t *testing.T) {
	timestamp, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
	log, textOut, jsonOut := newTestLogger()
	infoLevel := log.Config().InfoLevel()
	text := []byte("ab")
	binary := []byte{0xff, 0xfe}
	var nilBytes []byte

	for _, tc := range []struct {
		name     string
		any      func(*Message) *Message
		expected func(*Message) *Message
		json     string
	}{
		{
			name:     "text",
			any:      func(m *Message) *Message { return m.Any("b", text) },
			expected: func(m *Message) *Message { return m.StrBytes("b", text) },
			json:     `"b":"ab"`,
		},
		{
			name:     "binary",
			any:      func(m *Message) *Message { return m.Any("b", binary) },
			expected: func(m *Message) *Message { return m.StrBytes("b", binary) },
			json:     `"b":"__4"`,
		},
		{
			name:     "nil",
			any:      func(m *Message) *Message { return m.Any("b", nilBytes) },
			expected: func(m *Message) *Message { return m.StrBytes("b", nil) },
			json:     `"b":null`,
		},
		{
			name:     "pointer",
			any:      func(m *Message) *Message { return m.Any("b", &text) },
			expected: func(m *Message) *Message { return m.StrBytes("b", text) },
			json:     `"b":"ab"`,
		},
		{
			name:     "nil pointer",
			any:      func(m *Message) *Message { return m.Any("b", (*[]byte)(nil)) },
			expected: func(m *Message) *Message { return m.Nil("b") },
			json:     `"b":null`,
		},
		{
			name:     "named type",
			any:      func(m *Message) *Message { return m.Any("b", testNamedBytes(binary)) },
			expected: func(m *Message) *Message { return m.StrBytes("b", binary) },
			json:     `"b":"__4"`,
		},
		{
			name:     "pointer to named type",
			any:      func(m *Message) *Message { nb := testNamedBytes(text); return m.Any("b", &nb) },
			expected: func(m *Message) *Message { return m.StrBytes("b", text) },
			json:     `"b":"ab"`,
		},
		{
			name: "struct field",
			any: func(m *Message) *Message {
				return m.StructFields(struct {
					B []byte `json:"B"`
				}{B: binary})
			},
			expected: func(m *Message) *Message { return m.StrBytes("B", binary) },
			json:     `"B":"__4"`,
		},
		{
			// json.RawMessage holds JSON, no text or binary data
			name:     "json.RawMessage",
			any:      func(m *Message) *Message { return m.Any("b", json.RawMessage(`{ "a": 1 }`)) },
			expected: func(m *Message) *Message { return m.JSON("b", []byte(`{"a":1}`)) },
			json:     `"b":{"a":1}`,
		},
		{
			// Message.StrBytes has no slice variant to compare with
			name: "slice of bytes",
			any:  func(m *Message) *Message { return m.Any("b", [][]byte{text, binary, nil}) },
			json: `"b":["ab","__4",null]`,
		},
		{
			name: "slice of named bytes",
			any:  func(m *Message) *Message { return m.Any("b", []testNamedBytes{text, binary}) },
			json: `"b":["ab","__4"]`,
		},
		// Byte slice types with a format of their own are logged with it
		{
			name:     "net.IP with MarshalText",
			any:      func(m *Message) *Message { return m.Any("b", net.IPv4(192, 168, 1, 1)) },
			expected: func(m *Message) *Message { return m.Str("b", "192.168.1.1") },
			json:     `"b":"192.168.1.1"`,
		},
		{
			name:     "nil *net.IP",
			any:      func(m *Message) *Message { return m.Any("b", (*net.IP)(nil)) },
			expected: func(m *Message) *Message { return m.Nil("b") },
			json:     `"b":null`,
		},
		{
			name: "slice of net.IP",
			any:  func(m *Message) *Message { return m.Any("b", []net.IP{net.IPv4(10, 0, 0, 1), net.IPv6loopback}) },
			json: `"b":["10.0.0.1","::1"]`,
		},
		{
			name:     "net.HardwareAddr with String",
			any:      func(m *Message) *Message { return m.Any("b", net.HardwareAddr{1, 2, 3, 4, 5, 6}) },
			expected: func(m *Message) *Message { return m.Str("b", "01:02:03:04:05:06") },
			json:     `"b":"01:02:03:04:05:06"`,
		},
		{
			name:     "MarshalJSON redacts",
			any:      func(m *Message) *Message { return m.Any("b", testSecretJSONBytes("hunter2")) },
			expected: func(m *Message) *Message { return m.JSON("b", []byte(`"***"`)) },
			json:     `"b":"***"`,
		},
		{
			name: "error redacts",
			any:  func(m *Message) *Message { return m.Any("b", testSecretErrorBytes("hunter2")) },
			json: `"b":"***"`,
		},
		{
			name: "Loggable redacts",
			any:  func(m *Message) *Message { return m.Any("b", testSecretLogBytes("hunter2")) },
			json: `"b":{"secret":"***"}`,
		},
		{
			// Error has a value receiver, which panics for a nil pointer
			name:     "nil pointer to error bytes",
			any:      func(m *Message) *Message { return m.Any("b", (*testSecretErrorBytes)(nil)) },
			expected: func(m *Message) *Message { return m.Nil("b") },
			json:     `"b":null`,
		},
		{
			name: "nil pointer to error bytes in []any",
			any:  func(m *Message) *Message { return m.Any("b", []any{(*testSecretErrorBytes)(nil)}) },
			json: `"b":[null]`,
		},
		{
			// The format of the type is used through several pointers
			name: "pointer to pointer to MarshalJSON",
			any: func(m *Message) *Message {
				secret := testSecretJSONBytes("hunter2")
				ptr := &secret
				return m.Any("b", &ptr)
			},
			expected: func(m *Message) *Message { return m.JSON("b", []byte(`"***"`)) },
			json:     `"b":"***"`,
		},
		{
			name:     "IsNull",
			any:      func(m *Message) *Message { return m.Any("b", testNullableBytes{}) },
			expected: func(m *Message) *Message { return m.Nil("b") },
			json:     `"b":null`,
		},
		{
			name:     "nil pointer to IsNull with value receiver",
			any:      func(m *Message) *Message { return m.Any("b", (*testNullableID)(nil)) },
			expected: func(m *Message) *Message { return m.Nil("b") },
			json:     `"b":null`,
		},
		// Format methods with pointer receiver are used through pointers
		// and for addressable values like slice elements, like encoding/json
		{
			name: "pointer to pointer receiver String",
			any: func(m *Message) *Message {
				secret := testPtrSecretBytes("hunter2")
				return m.Any("b", &secret)
			},
			expected: func(m *Message) *Message { return m.Str("b", "***") },
			json:     `"b":"***"`,
		},
		{
			name: "pointer to pointer to pointer receiver String",
			any: func(m *Message) *Message {
				secret := testPtrSecretBytes("hunter2")
				ptr := &secret
				return m.Any("b", &ptr)
			},
			expected: func(m *Message) *Message { return m.Str("b", "***") },
			json:     `"b":"***"`,
		},
		{
			name: "slice of pointer receiver String",
			any:  func(m *Message) *Message { return m.Any("b", []testPtrSecretBytes{[]byte("hunter2")}) },
			json: `"b":["***"]`,
		},
		{
			name: "Loggable logging pointer to pointer receiver String",
			any:  func(m *Message) *Message { return m.Any("b", &testPtrSecretUser{pw: []byte("hunter2")}) },
			json: `"b":{"pw":"***"}`,
		},
		{
			// Pointer types that point to each other must not hang
			name:     "pointer type cycle",
			any:      func(m *Message) *Message { return m.Any("b", testCycleA(nil)) },
			expected: func(m *Message) *Message { return m.Nil("b") },
			json:     `"b":null`,
		},
		{
			name: "redacting element of []any",
			any: func(m *Message) *Message {
				return m.Any("b", []any{testSecretJSONBytes("hunter2"), testSecretErrorBytes("hunter2")})
			},
			json: `"b":["***","***"]`,
		},
		{
			name: "slice of json.RawMessage",
			any: func(m *Message) *Message {
				return m.Any("b", []json.RawMessage{json.RawMessage(`[1, 2]`), nil, json.RawMessage(`{`)})
			},
			json: `"b":[[1,2],null,"{"]`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.any(log.NewMessageAt(context.Background(), timestamp, infoLevel, "Msg")).Log()
			assert.Contains(t, jsonOut.String(), tc.json)
			if tc.expected != nil {
				gotText, gotJSON := textOut.String(), jsonOut.String()
				textOut.Reset()
				jsonOut.Reset()
				tc.expected(log.NewMessageAt(context.Background(), timestamp, infoLevel, "Msg")).Log()
				assert.Equal(t, textOut.String(), gotText, "text like Message.StrBytes")
				assert.Equal(t, jsonOut.String(), gotJSON, "JSON like Message.StrBytes")
			}
			textOut.Reset()
			jsonOut.Reset()
		})
	}
}

// TestMessage_ByteMethods tests every Message method taking bytes: they all
// log bytes as string if they are valid UTF-8, else base64 encoded, and nil
// bytes as null, except Message.Bytes, which always logs binary data base64
// encoded, and Message.JSON, which logs JSON. An attrib recorder records
// the same values for a sub-logger, and a key of the logger is not written
// again.
func TestMessage_ByteMethods(t *testing.T) {
	timestamp, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
	log, _, jsonOut := newTestLogger()
	infoLevel := log.Config().InfoLevel()
	replacementChar := []byte("a\uFFFDb") // Valid UTF-8 of the Unicode replacement character
	binary := []byte{0xff, 0xfe}

	for _, tc := range []struct {
		name string
		log  func(*Message) *Message
		json string // empty for no attrib
	}{
		{name: "Bytes text", log: func(m *Message) *Message { return m.Bytes("b", []byte("ab")) }, json: `"b":"YWI"`},
		{name: "Bytes binary", log: func(m *Message) *Message { return m.Bytes("b", binary) }, json: `"b":"__4"`},
		{name: "Bytes nil", log: func(m *Message) *Message { return m.Bytes("b", nil) }, json: `"b":null`},
		{name: "StrBytes text", log: func(m *Message) *Message { return m.StrBytes("b", []byte("ab")) }, json: `"b":"ab"`},
		{name: "StrBytes replacement char", log: func(m *Message) *Message { return m.StrBytes("b", replacementChar) }, json: "\"b\":\"a\uFFFDb\""},
		{name: "StrBytes binary", log: func(m *Message) *Message { return m.StrBytes("b", binary) }, json: `"b":"__4"`},
		{name: "StrBytes nil", log: func(m *Message) *Message { return m.StrBytes("b", nil) }, json: `"b":null`},
		{name: "StrBytesMax truncated", log: func(m *Message) *Message { return m.StrBytesMax("b", []byte("äbc"), 2) }, json: `"b":"äb"`},
		{name: "StrBytesMax shorter", log: func(m *Message) *Message { return m.StrBytesMax("b", []byte("ab"), 5) }, json: `"b":"ab"`},
		{name: "StrBytesMax no max", log: func(m *Message) *Message { return m.StrBytesMax("b", []byte("abc"), 0) }, json: `"b":"abc"`},
		{name: "StrBytesMax replacement char", log: func(m *Message) *Message { return m.StrBytesMax("b", replacementChar, 5) }, json: "\"b\":\"a\uFFFDb\""},
		{name: "StrBytesMax binary truncated", log: func(m *Message) *Message { return m.StrBytesMax("b", binary, 2) }, json: `"b":"__"`},
		{name: "StrBytesMax nil", log: func(m *Message) *Message { return m.StrBytesMax("b", nil, 2) }, json: `"b":null`},
		// Only the logged runes are checked for valid UTF-8, so a text
		// cut within a rune after them is no binary data
		{name: "StrBytesMax invalid after max", log: func(m *Message) *Message { return m.StrBytesMax("b", []byte("hello world\xe2\x82"), 5) }, json: `"b":"hello"`},
		{name: "StrBytesMax invalid within max", log: func(m *Message) *Message { return m.StrBytesMax("b", []byte("he\xffllo"), 3) }, json: `"b":"aGX"`},
		{name: "JSON", log: func(m *Message) *Message { return m.JSON("b", []byte(`{ "a": [1, 2] }`)) }, json: `"b":{"a":[1,2]}`},
		{name: "JSON invalid", log: func(m *Message) *Message { return m.JSON("b", []byte(`{`)) }, json: `"b":"{"`},
		{name: "JSON nil", log: func(m *Message) *Message { return m.JSON("b", nil) }, json: `"b":null`},
		{name: "Any nil", log: func(m *Message) *Message { return m.Any("b", nil) }, json: `"b":null`},
		// Uint8s logs the same Go type as []byte as numbers
		{name: "Uint8s", log: func(m *Message) *Message { return m.Uint8s("b", []uint8("ab")) }, json: `"b":[97,98]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.log(log.NewMessageAt(context.Background(), timestamp, infoLevel, "Msg")).Log()
			assert.Equal(t, `{"time":"2006-01-02 15:04:05","level":"INFO","message":"Msg",`+tc.json+"}\n", jsonOut.String())
			jsonOut.Reset()

			// A key that the logger already wrote is not written again
			withB := log.WithClonedAttribs(NewString("b", "first"))
			tc.log(withB.NewMessageAt(context.Background(), timestamp, infoLevel, "Msg")).Log()
			assert.Equal(t, `{"time":"2006-01-02 15:04:05","level":"INFO","message":"Msg","b":"first"}`+"\n", jsonOut.String())
			jsonOut.Reset()
			withB.RemoveAttribs()

			// A sub-logger logs the recorded value the same way
			subLog := tc.log(log.With()).SubLogger()
			subLog.NewMessageAt(context.Background(), timestamp, infoLevel, "Msg").Log()
			assert.Equal(t, `{"time":"2006-01-02 15:04:05","level":"INFO","message":"Msg",`+tc.json+"}\n", jsonOut.String())
			jsonOut.Reset()
			subLog.RemoveAttribs()

			// A nil Message is safe to use
			assert.Nil(t, tc.log(nil))
		})
	}
}

// TestMessage_Any_SliceElements tests the elements of a []any logged with
// Message.Any: nil and values whose IsNull method returns true as null,
// errors as their message, and other values with their own type instead
// of formatted as string.
func TestMessage_Any_SliceElements(t *testing.T) {
	timestamp, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
	log, _, jsonOut := newTestLogger()

	log.NewMessageAt(context.Background(), timestamp, log.Config().InfoLevel(), "Msg").
		Any("s", []any{nil, errors.New("e"), Timestamp{}, []byte("ab"), 1, -2, uint(3), 1.5, true, "x", map[string]int{"a": 1}}).
		Log()
	assert.Equal(t, `{"time":"2006-01-02 15:04:05","level":"INFO","message":"Msg","s":[null,"e",null,"ab",1,-2,3,1.5,true,"x",{"a":1}]}`+"\n", jsonOut.String())
}

// TestByteSliceOf tests which values count as bytes for Message.Any
// and the Any attrib: byte slices of any named type, pointers to them and
// interface values holding them, but no json.RawMessage, which holds JSON.
func TestByteSliceOf(t *testing.T) {
	text := []byte("ab")
	var nilInterface any
	var interfaceWithBytes any = text
	for _, tc := range []struct {
		name   string
		val    reflect.Value
		want   []byte
		wantOK bool
	}{
		{name: "[]byte", val: reflect.ValueOf(text), want: text, wantOK: true},
		{name: "named", val: reflect.ValueOf(testNamedBytes(text)), want: text, wantOK: true},
		{name: "pointer", val: reflect.ValueOf(&text), want: text, wantOK: true},
		{name: "nil pointer", val: reflect.ValueOf((*[]byte)(nil)), want: nil, wantOK: true},
		{name: "interface", val: reflect.ValueOf(&interfaceWithBytes).Elem(), want: text, wantOK: true},
		{name: "nil interface", val: reflect.ValueOf(&nilInterface).Elem(), want: nil, wantOK: false},
		{name: "json.RawMessage", val: reflect.ValueOf(json.RawMessage(`1`)), want: nil, wantOK: false},
		{name: "string", val: reflect.ValueOf("ab"), want: nil, wantOK: false},
		{name: "[]int", val: reflect.ValueOf([]int{1}), want: nil, wantOK: false},
		{name: "byte array", val: reflect.ValueOf([2]byte{1, 2}), want: nil, wantOK: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := byteSliceOf(tc.val)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

type testLoggable struct{ val string }

func (l testLoggable) Log(m *Message) {
	if l.val != "" {
		m.Str("inner", l.val).Int("n", 1)
	}
}

// TestMessage_Any_Loggable tests that the attribs a Loggable logs are
// written as one JSON object under the key passed to Message.Any, also
// inside a slice. It used to call Loggable.Log once per writer while every
// call wrote to all writers, which left the key without a value and
// repeated the attribs: invalid JSON.
func TestMessage_Any_Loggable(t *testing.T) {
	timestamp, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
	log, textOut, jsonOut := newTestLogger()
	infoLevel := log.Config().InfoLevel()
	loggable := testLoggable{val: "v"}

	for _, tc := range []struct {
		name string
		val  any
		text string
		json string
	}{
		{name: "value", val: loggable, text: `l={"inner":"v","n":1}`, json: `"l":{"inner":"v","n":1}`},
		{name: "pointer", val: &loggable, text: `l={"inner":"v","n":1}`, json: `"l":{"inner":"v","n":1}`},
		{name: "nil pointer", val: (*testLoggable)(nil), text: `l=nil`, json: `"l":null`},
		{name: "no attribs", val: testLoggable{}, text: `l=null`, json: `"l":null`},
		{name: "Attribs", val: Attribs{NewString("a", "1"), NewBool("b", true)}, text: `l={"a":"1","b":true}`, json: `"l":{"a":"1","b":true}`},
		{name: "Attrib", val: NewInt("a", 1), text: `l={"a":1}`, json: `"l":{"a":1}`},
		{name: "in slice", val: []any{loggable, 2}, text: `l=[{"inner":"v","n":1},2]`, json: `"l":[{"inner":"v","n":1},2]`},
		// A nil pointer in a []any is an interface value, not a pointer,
		// and calling its value receiver Log method used to panic
		{name: "nil pointer in slice", val: []any{(*testLoggable)(nil), 2}, text: `l=[nil,2]`, json: `"l":[null,2]`},
		// A Loggable recorded with Message.Any by a Loggable is written
		// as JSON object too, not by encoding/json as [{}]
		{name: "nested", val: Attribs{NewAny("a", Attribs{NewInt("b", 1)})}, text: `l={"a":{"b":1}}`, json: `"l":{"a":{"b":1}}`},
		// Invalid JSON recorded by a Loggable is written as string
		// like Message.JSON does, instead of breaking the JSON line
		{name: "invalid nested JSON", val: Attribs{NewAny("a", json.RawMessage("no json"))}, text: `l={"a":"no json"}`, json: `"l":{"a":"no json"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			DrainAllMemPools()
			mempool.RegisterCallbacksWriterForTest(t, io.Discard)

			log.NewMessageAt(context.Background(), timestamp, infoLevel, "Msg").Any("l", tc.val).Log()
			assert.Equal(t, `2006-01-02 15:04:05 |INFO | Msg `+tc.text+"\n", textOut.String())
			assert.Equal(t, `{"time":"2006-01-02 15:04:05","level":"INFO","message":"Msg",`+tc.json+"}\n", jsonOut.String())
			assert.True(t, json.Valid(jsonOut.Bytes()), "valid JSON")
			assert.Zero(t, mempool.NumOutstanding(), "the recorded attribs were returned to the mempool")
			textOut.Reset()
			jsonOut.Reset()

			// A sub-logger writes the recorded Loggable the same way
			subLog := log.With().Any("l", tc.val).SubLogger()
			subLog.NewMessageAt(context.Background(), timestamp, infoLevel, "Msg").Log()
			assert.Equal(t, `{"time":"2006-01-02 15:04:05","level":"INFO","message":"Msg",`+tc.json+"}\n", jsonOut.String())
			subLog.RemoveAttribs()
			textOut.Reset()
			jsonOut.Reset()
		})
	}

	// Message.Loggable logs the attribs as attribs of the message,
	// and nothing for a nil Loggable
	log.NewMessageAt(context.Background(), timestamp, infoLevel, "Msg").Loggable(loggable).Loggable(nil).Log()
	assert.Equal(t, `{"time":"2006-01-02 15:04:05","level":"INFO","message":"Msg","inner":"v","n":1}`+"\n", jsonOut.String())
}

type testNamedUUID [16]byte

// TestMessage_Any_Kinds tests how Message.Any writes values by their
// reflect kind: arrays element by element unless they are UUIDs, a slice
// nested in a slice formatted as string instead of recursing further,
// nil pointers as null, and an error for a map that can't be marshalled.
func TestMessage_Any_Kinds(t *testing.T) {
	timestamp, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
	log, _, jsonOut := newTestLogger()
	uuid := MustParseUUID("a547276f-b02b-4e7d-b67e-c6deb07567da")

	for _, tc := range []struct {
		name string
		val  any
		json string
	}{
		{name: "array", val: [2]int{1, 2}, json: `"v":[1,2]`},
		{name: "named UUID", val: testNamedUUID(uuid), json: `"v":"a547276f-b02b-4e7d-b67e-c6deb07567da"`},
		{name: "nil named UUID", val: testNamedUUID{}, json: `"v":null`},
		{name: "UUIDs in slice", val: []testNamedUUID{testNamedUUID(uuid), {}}, json: `"v":["a547276f-b02b-4e7d-b67e-c6deb07567da",null]`},
		{name: "nested slice", val: [][]int{{1, 2}, {3}}, json: `"v":["[1 2]","[3]"]`},
		{name: "nested array", val: [][2]int{{1, 2}}, json: `"v":["[1 2]"]`},
		{name: "nil pointer element", val: []*int{nil}, json: `"v":[null]`},
		{name: "unmarshalable map", val: map[string]any{"c": make(chan int)}, json: `"v":"error while marshalling map[string]interface {} as JSON for logging: json: unsupported type: chan int"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			log.NewMessageAt(context.Background(), timestamp, log.Config().InfoLevel(), "Msg").Any("v", tc.val).Log()
			assert.Equal(t, `{"time":"2006-01-02 15:04:05","level":"INFO","message":"Msg",`+tc.json+"}\n", jsonOut.String())
			jsonOut.Reset()
		})
	}
}

// TestBinaryEncoder tests that every place logging binary data uses the
// configured BinaryEncoder: Message.Bytes, the fallback for bytes that are
// no valid UTF-8 in Message.StrBytes, Message.StrBytesMax and Message.Any,
// and the Any attrib. Bytes that are valid UTF-8 stay strings.
func TestBinaryEncoder(t *testing.T) {
	defaultEncoder := BinaryEncoder
	t.Cleanup(func() { BinaryEncoder = defaultEncoder })
	BinaryEncoder = hex.EncodeToString

	timestamp, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
	log, _, jsonOut := newTestLogger()
	binary := []byte{0xff, 0xfe}

	log.NewMessageAt(context.Background(), timestamp, log.Config().InfoLevel(), "Msg").
		Bytes("bytes", []byte("ab")).
		StrBytes("strBytes", binary).
		StrBytesMax("strBytesMax", binary, 3).
		Any("any", binary).
		Any("anyText", []byte("ab")).
		Log()
	assert.Equal(t, `{"time":"2006-01-02 15:04:05","level":"INFO","message":"Msg","bytes":"6162","strBytes":"fffe","strBytesMax":"fff","any":"fffe","anyText":"ab"}`+"\n", jsonOut.String())

	data, err := Attribs{NewAny("any", binary)}.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `{"any":"fffe"}`, string(data))
	assert.Equal(t, "fffe", NewAny("any", binary).ValueString())
}

// TestMessage_NilSafety tests that the Message methods dealing with attribs
// and contexts are safe to call on a nil Message, which inactive log levels
// return, so callers never have to check for nil.
func TestMessage_NilSafety(t *testing.T) {
	var m *Message
	ctx := ContextWithAttribs(context.Background(), NewString("a", "1"))

	assert.Nil(t, m.Ctx(ctx))
	assert.Nil(t, m.Loggable(Attribs{NewString("a", "1")}))
	assert.Nil(t, m.SubLogger())
	subLog, subCtx := m.SubLoggerContext(ctx)
	assert.Nil(t, subLog)
	assert.Equal(t, ctx, subCtx, "passed ctx is returned unchanged")
	assert.Equal(t, ctx, m.SubContext(ctx), "passed ctx is returned unchanged")
}

// func ExampleMessage_CallStack() {
// 	at, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")

// 	format := &Format{
// 		TimestampFormat: "2006-01-02 15:04:05",
// 		TimestampKey:    "time",
// 		LevelKey:        "level",
// 		PrefixSep:       ": ",
// 		MessageKey:      "message",
// 	}

// 	textWriter := NewTextWriter(os.Stdout, format, NoColorizer)

// 	config := NewConfig(&DefaultLevels, AllLevelsActive, textWriter)
// 	log := NewLogger(config)

// 	log.NewMessage(ctx, log.Config().Info(), "CallStack Example").
// 		CallStack("stack").
// 		Log()

// 	// Output:
// }

func TestMessage_StructFields(t *testing.T) {
	timestamp, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
	ctx := ContextWithTimestamp(context.Background(), timestamp)

	log, textOut, jsonOut := newTestLogger()
	infoLevel := log.Config().InfoLevel()

	t.Run("basic struct fields", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		// Fields use an empty json tag to fall back to the Go field name
		// under the new tag-driven rules for StructFields.
		type BasicStruct struct {
			Name    string `json:""`
			Age     int    `json:""`
			Active  bool   `json:""`
			Skipped string // no tag → not logged under the new rules
			private string //nolint:unused
		}

		s := BasicStruct{
			Name:    "John",
			Age:     30,
			Active:  true,
			Skipped: "should_not_appear",
			private: "hidden",
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			StructFields(s).
			Log()

		assert.Contains(t, textOut.String(), `Name="John"`)
		assert.Contains(t, textOut.String(), `Age=30`)
		assert.Contains(t, textOut.String(), `Active=true`)
		assert.NotContains(t, textOut.String(), `Skipped`)
		assert.NotContains(t, textOut.String(), `should_not_appear`)
		assert.NotContains(t, textOut.String(), `private`)
		assert.NotContains(t, textOut.String(), `hidden`)
	})

	t.Run("redact tag", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		// Canonical form for redact is now `golog:",redact"`: empty name
		// falls back to the Go field name, `redact` is a modifier.
		type UserWithSecrets struct {
			Username string `golog:""`
			Password string `golog:",redact"`
			APIKey   string `golog:",redact"`
			Email    string `golog:""`
		}

		u := UserWithSecrets{
			Username: "john_doe",
			Password: "secret123",
			APIKey:   "sk-abc123",
			Email:    "john@example.com",
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			StructFields(u).
			Log()

		assert.Contains(t, textOut.String(), `Username="john_doe"`)
		assert.Contains(t, textOut.String(), `Password="***REDACTED***"`)
		assert.Contains(t, textOut.String(), `APIKey="***REDACTED***"`)
		assert.Contains(t, textOut.String(), `Email="john@example.com"`)
		assert.NotContains(t, textOut.String(), `secret123`)
		assert.NotContains(t, textOut.String(), `sk-abc123`)

		// Also check JSON output
		assert.Contains(t, jsonOut.String(), `"Password":"***REDACTED***"`)
		assert.Contains(t, jsonOut.String(), `"APIKey":"***REDACTED***"`)
	})

	t.Run("embedded struct", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		type Embedded struct {
			EmbeddedField string `json:""`
		}

		type OuterStruct struct {
			Embedded
			OuterField string `json:""`
		}

		s := OuterStruct{
			Embedded:   Embedded{EmbeddedField: "embedded_value"},
			OuterField: "outer_value",
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			StructFields(s).
			Log()

		assert.Contains(t, textOut.String(), `EmbeddedField="embedded_value"`)
		assert.Contains(t, textOut.String(), `OuterField="outer_value"`)
	})

	t.Run("pointer to struct", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		type SimpleStruct struct {
			Value string `json:""`
		}

		s := &SimpleStruct{Value: "pointer_value"}

		log.NewMessage(ctx, infoLevel, "Msg").
			StructFields(s).
			Log()

		assert.Contains(t, textOut.String(), `Value="pointer_value"`)
	})

	t.Run("nil struct pointer", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		type SimpleStruct struct {
			Value string
		}

		var s *SimpleStruct

		log.NewMessage(ctx, infoLevel, "Msg").
			StructFields(s).
			Log()

		// Should not contain any field from the struct
		assert.NotContains(t, textOut.String(), `Value=`)
	})

	t.Run("untagged struct logs nothing", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		// Fields with none of the default keyTags (golog, log, json) are
		// skipped entirely. This is intentional — differs from encoding/json.
		type NoTagsStruct struct {
			A string
			B int
			C bool
		}

		s := NoTagsStruct{A: "x", B: 1, C: true}

		log.NewMessage(ctx, infoLevel, "Msg").
			StructFields(s).
			Log()

		assert.NotContains(t, textOut.String(), `A=`)
		assert.NotContains(t, textOut.String(), `B=`)
		assert.NotContains(t, textOut.String(), `C=`)
	})

	t.Run("multi-keyTag fallback order", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		// StructFields prefers golog, then log, then json for naming.
		type MixedTags struct {
			A string `golog:"aa"`
			B string `log:"bb"`
			C string `json:"cc"`
			D string `log:"dd_log" json:"dd_json"`     // log wins over json
			E string `golog:"ee_golog" json:"ee_json"` // golog wins over json
		}

		s := MixedTags{A: "a_val", B: "b_val", C: "c_val", D: "d_val", E: "e_val"}

		log.NewMessage(ctx, infoLevel, "Msg").
			StructFields(s).
			Log()

		assert.Contains(t, textOut.String(), `aa="a_val"`)
		assert.Contains(t, textOut.String(), `bb="b_val"`)
		assert.Contains(t, textOut.String(), `cc="c_val"`)
		assert.Contains(t, textOut.String(), `dd_log="d_val"`)
		assert.NotContains(t, textOut.String(), `dd_json`)
		assert.Contains(t, textOut.String(), `ee_golog="e_val"`)
		assert.NotContains(t, textOut.String(), `ee_json`)
	})

	t.Run("winning tag controls modifiers", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		// golog wins for naming, so json's omitempty is silently discarded.
		// The field is emitted even though its value is empty.
		type Mixed struct {
			A string `golog:"a" json:"b,omitempty"`
		}

		s := Mixed{A: ""}

		log.NewMessage(ctx, infoLevel, "Msg").
			StructFields(s).
			Log()

		assert.Contains(t, textOut.String(), `a=""`)
		assert.NotContains(t, textOut.String(), `b=`)
	})

	t.Run("golog dash skips the field", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		type DashedStruct struct {
			Kept    string `golog:""`
			Skipped string `golog:"-"`
		}

		s := DashedStruct{Kept: "k", Skipped: "s"}

		log.NewMessage(ctx, infoLevel, "Msg").
			StructFields(s).
			Log()

		assert.Contains(t, textOut.String(), `Kept="k"`)
		assert.NotContains(t, textOut.String(), `Skipped`)
	})

	t.Run("omitempty in golog tag", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		type E struct {
			Str   string            `golog:",omitempty"`
			Int   int               `golog:",omitempty"`
			Bool  bool              `golog:",omitempty"`
			Slice []int             `golog:",omitempty"`
			Map   map[string]string `golog:",omitempty"`
			Ptr   *string           `golog:",omitempty"`
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			StructFields(E{}).
			Log()

		// All zero values → all skipped.
		assert.NotContains(t, textOut.String(), `Str=`)
		assert.NotContains(t, textOut.String(), `Int=`)
		assert.NotContains(t, textOut.String(), `Bool=`)
		assert.NotContains(t, textOut.String(), `Slice=`)
		assert.NotContains(t, textOut.String(), `Map=`)
		assert.NotContains(t, textOut.String(), `Ptr=`)

		textOut.Reset()
		jsonOut.Reset()

		strPtr := "p"
		log.NewMessage(ctx, infoLevel, "Msg").
			StructFields(E{
				Str:   "s",
				Int:   1,
				Bool:  true,
				Slice: []int{1},
				Map:   map[string]string{"k": "v"},
				Ptr:   &strPtr,
			}).
			Log()

		// All non-zero → all present.
		assert.Contains(t, textOut.String(), `Str="s"`)
		assert.Contains(t, textOut.String(), `Int=1`)
		assert.Contains(t, textOut.String(), `Bool=true`)
		assert.Contains(t, textOut.String(), `Slice=`)
		assert.Contains(t, textOut.String(), `Map=`)
		assert.Contains(t, textOut.String(), `Ptr=`)
	})

	t.Run("omitzero suppresses time.Time via IsZero method", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		type Z struct {
			When time.Time `golog:",omitzero"`
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			StructFields(Z{}).
			Log()

		// zero time.Time has IsZero()==true → suppressed
		assert.NotContains(t, textOut.String(), `When=`)

		textOut.Reset()
		jsonOut.Reset()

		log.NewMessage(ctx, infoLevel, "Msg").
			StructFields(Z{When: time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)}).
			Log()

		assert.Contains(t, textOut.String(), `When=`)
	})

	t.Run("omitnull suppresses zero golog.Timestamp via IsNull method", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		type N struct {
			When Timestamp `golog:",omitnull"`
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			StructFields(N{}).
			Log()

		// zero Timestamp has IsNull()==true → suppressed
		assert.NotContains(t, textOut.String(), `When=`)

		textOut.Reset()
		jsonOut.Reset()

		log.NewMessage(ctx, infoLevel, "Msg").
			StructFields(N{When: Timestamp{Time: time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)}}).
			Log()

		assert.Contains(t, textOut.String(), `When=`)
	})

	t.Run("omitnull falls back to omitzero when no IsNull method", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		// time.Time has IsZero() but no IsNull() — omitnull falls back to
		// the omitzero path and suppresses the zero value.
		type F struct {
			When time.Time `golog:",omitnull"`
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			StructFields(F{}).
			Log()

		assert.NotContains(t, textOut.String(), `When=`)
	})

	t.Run("nil pointer fields do not panic with omit modifiers", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		// Regression: without the nil-pointer short-circuit in
		// shouldOmitStructField and isZeroValueForOmitzero, a nil
		// *Timestamp with ,omitnull or a nil *time.Time with ,omitzero
		// panics because the value-receiver IsNull / IsZero method
		// auto-dereferences the nil pointer. Both must now be suppressed
		// silently.
		type Event struct {
			ProcessedAt *time.Time `golog:",omitzero"`
			DeletedAt   *Timestamp `golog:",omitnull"`
			UpdatedAt   *time.Time `golog:",omitnull"` // omitnull → omitzero fallback on nil
		}

		assert.NotPanics(t, func() {
			log.NewMessage(ctx, infoLevel, "Msg").
				StructFields(Event{}).
				Log()
		}, "nil pointer fields with omit modifiers must not panic")

		assert.NotContains(t, textOut.String(), `ProcessedAt=`)
		assert.NotContains(t, textOut.String(), `DeletedAt=`)
		assert.NotContains(t, textOut.String(), `UpdatedAt=`)

		// Non-nil pointer to a non-zero value must still log.
		textOut.Reset()
		jsonOut.Reset()
		t1 := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)
		ts := Timestamp{Time: t1}
		log.NewMessage(ctx, infoLevel, "Msg").
			StructFields(Event{ProcessedAt: &t1, DeletedAt: &ts, UpdatedAt: &t1}).
			Log()
		assert.Contains(t, textOut.String(), `ProcessedAt=`)
		assert.Contains(t, textOut.String(), `DeletedAt=`)
		assert.Contains(t, textOut.String(), `UpdatedAt=`)
	})

	t.Run("omitempty wins over redact for empty values", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		// Empty field with both redact and omitempty is suppressed, not
		// emitted with ***REDACTED***. Suppression wins — matches
		// encoding/json omitempty precedence.
		type R struct {
			Token string `golog:",redact,omitempty"`
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			StructFields(R{Token: ""}).
			Log()

		assert.NotContains(t, textOut.String(), `Token=`)
		assert.NotContains(t, textOut.String(), `REDACTED`)

		textOut.Reset()
		jsonOut.Reset()

		log.NewMessage(ctx, infoLevel, "Msg").
			StructFields(R{Token: "secret"}).
			Log()

		assert.Contains(t, textOut.String(), `Token="***REDACTED***"`)
		assert.NotContains(t, textOut.String(), `secret`)
	})
}

func TestMessage_TaggedStructFields(t *testing.T) {
	timestamp, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
	ctx := ContextWithTimestamp(context.Background(), timestamp)

	log, textOut, jsonOut := newTestLogger()
	infoLevel := log.Config().InfoLevel()

	t.Run("json tag", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		type JSONTaggedStruct struct {
			UserName string `json:"user_name"`
			UserAge  int    `json:"user_age"`
			NoTag    string // no json tag at all → skipped under the new rules
			Ignored  string `json:"-"`
			// Empty json tag value → encoding/json.Marshal parity:
			// fall back to the Go field name ("Empty") as the log key.
			Empty string `json:""`
		}

		s := JSONTaggedStruct{
			UserName: "John",
			UserAge:  30,
			NoTag:    "no_tag_value",
			Ignored:  "ignored_value",
			Empty:    "empty_value",
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(s, "json").
			Log()

		assert.Contains(t, textOut.String(), `user_name="John"`)
		assert.Contains(t, textOut.String(), `user_age=30`)
		assert.Contains(t, textOut.String(), `Empty="empty_value"`) // json.Marshal parity: empty tag → Go field name
		assert.NotContains(t, textOut.String(), `UserName`)
		assert.NotContains(t, textOut.String(), `NoTag`)
		assert.NotContains(t, textOut.String(), `no_tag_value`)
		assert.NotContains(t, textOut.String(), `Ignored`)
		assert.NotContains(t, textOut.String(), `ignored_value`)
	})

	t.Run("json tag with options", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		type JSONTagWithOptions struct {
			Name   string `json:"name,omitempty"`
			Status string `json:"status,omitempty,string"`
		}

		s := JSONTagWithOptions{
			Name:   "Jane",
			Status: "active",
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(s, "json").
			Log()

		// Should use tag value before the first comma
		assert.Contains(t, textOut.String(), `name="Jane"`)
		assert.Contains(t, textOut.String(), `status="active"`)
	})

	t.Run("redact with json tag", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		// Under the unified rules modifiers come from the winning tag,
		// so when TaggedStructFields is called with "json" the redact
		// modifier must live inside the json tag. "redacted" is accepted
		// as a synonym for "redact". The struct type is built at runtime
		// via buildTestStruct so that golog-specific modifiers never live
		// in static struct tag literals (see buildTestStruct doc).
		s := buildTestStruct(
			tf{"Username", `json:"username"`, "admin"},
			tf{"Password", `json:"password,redact"`, "super_secret"},
			tf{"Token", `json:"token,redacted"`, "tok_xyz789"},
		)

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(s, "json").
			Log()

		assert.Contains(t, textOut.String(), `username="admin"`)
		assert.Contains(t, textOut.String(), `password="***REDACTED***"`)
		assert.Contains(t, textOut.String(), `token="***REDACTED***"`)
		assert.NotContains(t, textOut.String(), `super_secret`)
		assert.NotContains(t, textOut.String(), `tok_xyz789`)
	})

	t.Run("golog tag as key tag does not redact", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		// When keyTag is "golog", the tag value is used as the field key,
		// not as a redaction directive
		type GologTaggedStruct struct {
			Password string `golog:"secret_field"`
			APIKey   string `golog:"api_key"`
			NoTag    string
		}

		s := GologTaggedStruct{
			Password: "my_password",
			APIKey:   "my_api_key",
			NoTag:    "no_tag_value",
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(s, "golog").
			Log()

		// Values should NOT be redacted when using golog as the key tag
		assert.Contains(t, textOut.String(), `secret_field="my_password"`)
		assert.Contains(t, textOut.String(), `api_key="my_api_key"`)
		assert.NotContains(t, textOut.String(), `***REDACTED***`)
		assert.NotContains(t, textOut.String(), `NoTag`)
		assert.NotContains(t, textOut.String(), `no_tag_value`)
	})

	t.Run("custom tag", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		type CustomTaggedStruct struct {
			Field1 string `log:"field_one"`
			Field2 int    `log:"field_two"`
			Field3 string `json:"json_field"` // Different tag, should be ignored
		}

		s := CustomTaggedStruct{
			Field1: "value1",
			Field2: 42,
			Field3: "json_value",
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(s, "log").
			Log()

		assert.Contains(t, textOut.String(), `field_one="value1"`)
		assert.Contains(t, textOut.String(), `field_two=42`)
		assert.NotContains(t, textOut.String(), `Field3`)
		assert.NotContains(t, textOut.String(), `json_field`)
		assert.NotContains(t, textOut.String(), `json_value`)
	})

	t.Run("embedded struct with tags", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		type EmbeddedTagged struct {
			EmbeddedField string `json:"embedded_field"`
		}

		type OuterTagged struct {
			EmbeddedTagged
			OuterField string `json:"outer_field"`
		}

		s := OuterTagged{
			EmbeddedTagged: EmbeddedTagged{EmbeddedField: "embedded_value"},
			OuterField:     "outer_value",
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(s, "json").
			Log()

		assert.Contains(t, textOut.String(), `embedded_field="embedded_value"`)
		assert.Contains(t, textOut.String(), `outer_field="outer_value"`)
	})

	t.Run("nil struct", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(nil, "json").
			Log()

		// Should not panic and should produce minimal output
		assert.NotContains(t, textOut.String(), `=`)
	})

	// Whitespace trimming in tag values is exercised exhaustively at the
	// parser level in TestParseStructFieldDirectives. Struct tags with
	// leading or trailing spaces would trigger `go vet`'s structtag check,
	// so we do not test whitespace-in-tag-value via real struct tags here.

	t.Run("json omitempty parity", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		type OE struct {
			Present string `json:"present,omitempty"`
			Missing string `json:"missing,omitempty"`
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(OE{Present: "hi", Missing: ""}, "json").
			Log()

		assert.Contains(t, textOut.String(), `present="hi"`)
		assert.NotContains(t, textOut.String(), `missing`)
	})

	t.Run("json omitempty with empty name falls back to field name", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		type OEF struct {
			Present string `json:",omitempty"`
			Missing string `json:",omitempty"`
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(OEF{Present: "hi", Missing: ""}, "json").
			Log()

		// Present falls back to field name "Present" and is emitted.
		// Missing falls back to "Missing" but omitempty suppresses it.
		assert.Contains(t, textOut.String(), `Present="hi"`)
		assert.NotContains(t, textOut.String(), `Missing`)
	})

	t.Run("json omitzero with time.Time", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		type OZ struct {
			When time.Time `json:"when,omitzero"`
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(OZ{}, "json").
			Log()
		assert.NotContains(t, textOut.String(), `when=`)

		textOut.Reset()
		jsonOut.Reset()

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(OZ{When: time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)}, "json").
			Log()
		assert.Contains(t, textOut.String(), `when=`)
	})

	t.Run("json omitnull with Timestamp", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(
				buildTestStruct(tf{"At", `json:"at,omitnull"`, Timestamp{}}),
				"json",
			).
			Log()
		assert.NotContains(t, textOut.String(), `at=`)

		textOut.Reset()
		jsonOut.Reset()

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(
				buildTestStruct(tf{
					"At", `json:"at,omitnull"`,
					Timestamp{Time: time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)},
				}),
				"json",
			).
			Log()
		assert.Contains(t, textOut.String(), `at=`)
	})

	t.Run("json omitnull falls back to omitzero when no IsNull method", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(
				buildTestStruct(tf{"When", `json:"when,omitnull"`, time.Time{}}),
				"json",
			).
			Log()
		assert.NotContains(t, textOut.String(), `when=`)
	})

	t.Run("redact omitempty precedence in json tag", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(
				buildTestStruct(tf{"Token", `json:"token,redact,omitempty"`, ""}),
				"json",
			).
			Log()
		assert.NotContains(t, textOut.String(), `token=`)
		assert.NotContains(t, textOut.String(), `REDACTED`)

		textOut.Reset()
		jsonOut.Reset()

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(
				buildTestStruct(tf{"Token", `json:"token,redact,omitempty"`, "secret"}),
				"json",
			).
			Log()
		assert.Contains(t, textOut.String(), `token="***REDACTED***"`)
		assert.NotContains(t, textOut.String(), `secret`)
	})

	t.Run("json dash with trailing modifier is literal name", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		// `json:"-,omitempty"` means the literal field name "-" with the
		// omitempty modifier — matches encoding/json.Marshal. Only a bare
		// `json:"-"` with no comma skips.
		type DashNames struct {
			Literal string `json:"-,omitempty"`
			Normal  string `json:"normal"`
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(DashNames{Literal: "kept", Normal: "n"}, "json").
			Log()

		assert.Contains(t, textOut.String(), `-="kept"`)
		assert.Contains(t, textOut.String(), `normal="n"`)

		textOut.Reset()
		jsonOut.Reset()

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(DashNames{Literal: "", Normal: "n"}, "json").
			Log()

		// Empty literal "-" field → omitempty suppresses it.
		assert.NotContains(t, textOut.String(), `-=`)
		assert.Contains(t, textOut.String(), `normal="n"`)
	})

	t.Run("empty keyTag is wildcard that logs every exported field by Go name", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		// `TaggedStructFields(s, "")` is the escape hatch for "log every
		// exported field by its Go name", regardless of whether the field
		// carries a struct tag. This restores the pre-tag-driven
		// StructFields behavior for callers that want it.
		type Wild struct {
			A string
			B int
			C bool
			d string //nolint:unused
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(Wild{A: "x", B: 1, C: true, d: "hidden"}, "").
			Log()

		assert.Contains(t, textOut.String(), `A="x"`)
		assert.Contains(t, textOut.String(), `B=1`)
		assert.Contains(t, textOut.String(), `C=true`)
		assert.NotContains(t, textOut.String(), `d=`)
		assert.NotContains(t, textOut.String(), `hidden`)
	})

	t.Run("empty keyTag ignores struct tags on the fields", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		// Empty keyTag is a wildcard: it short-circuits the tag lookup
		// and uses the Go field name even when the field HAS a tag like
		// `json:"aaa"`. This is the semantic "log every field, no
		// renames".
		type Named struct {
			Foo string `json:"aaa"`
			Bar int    `json:"bbb,omitempty"`
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(Named{Foo: "v", Bar: 0}, "").
			Log()

		assert.Contains(t, textOut.String(), `Foo="v"`)
		// Bar is zero, but the empty-keyTag wildcard has no modifiers,
		// so omitempty from the json tag does NOT apply.
		assert.Contains(t, textOut.String(), `Bar=0`)
		assert.NotContains(t, textOut.String(), `aaa`)
		assert.NotContains(t, textOut.String(), `bbb`)
	})

	t.Run("unknown modifier is ignored silently", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		// Tag string is built at runtime so no analyzer can pattern-match
		// a compile-time `json:"..."` literal containing "wibble".
		unknownTag := fmt.Sprintf(`json:%q`, "x,wibble,omitempty")

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(
				buildTestStruct(tf{"X", unknownTag, "val"}),
				"json",
			).
			Log()

		assert.Contains(t, textOut.String(), `x="val"`)
	})
}

// TestSubLoggerContextWithWriterConfigsFromContext tests that SubLoggerContext
// does not create a self-referential DerivedConfig cycle when the context
// contains additional WriterConfigs. A self-referential cycle causes infinite
// recursion (stack overflow) when any level method like InfoLevel() is called.
func TestSubLoggerContextWithWriterConfigsFromContext(t *testing.T) {
	if os.Getenv("TEST_SUBLOGGER_CONTEXT_LOOP") == "1" {
		config := newTestConfig(io.Discard, io.Discard)
		log := NewLogger(config)
		// Add a new WriterConfig to the context that is not already
		// in the logger's config, so ConfigWithAdditionalWriterConfigs
		// creates a new DerivedConfig instead of returning the parent.
		extraWriter := NewJSONWriterConfig(io.Discard, nil)
		ctx := ContextWithAdditionalWriterConfigs(t.Context(), extraWriter)
		subLog, _ := log.With().Str("key", "value").SubLoggerContext(ctx)
		// Calling Info triggers InfoLevel() on the config.
		// With the bug, this causes infinite recursion and a stack overflow.
		subLog.Info("test").Log()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestSubLoggerContextWithWriterConfigsFromContext$", "-test.timeout=10s")
	cmd.Env = append(os.Environ(), "TEST_SUBLOGGER_CONTEXT_LOOP=1")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "SubLoggerContext with additional WriterConfigs from context caused stack overflow:\n%s", output)
}

// TestMessage_SubLoggerContext_WriterConfigsFromCtx tests that a sub-logger
// from SubLoggerContext also writes to the writer configs added to the
// passed context, and keeps the config of the parent logger when the context
// adds no writer config the parent doesn't already have.
func TestMessage_SubLoggerContext_WriterConfigsFromCtx(t *testing.T) {
	format := &Format{TimestampFormat: "2006-01-02 15:04:05", TimestampKey: "time", LevelKey: "level", MessageKey: "message"}
	mainOut := bytes.NewBuffer(nil)
	mainConfig := NewJSONWriterConfig(mainOut, format)
	log := NewLogger(NewConfig(&DefaultLevels, AllLevelsActive, mainConfig))

	t.Run("additional writer config", func(t *testing.T) {
		extraOut := bytes.NewBuffer(nil)
		ctx := ContextWithAdditionalWriterConfigs(context.Background(), NewJSONWriterConfig(extraOut, format))
		subLog, _ := log.With().Str("key", "value").SubLoggerContext(ctx)

		subLog.Info("Msg").Log()
		assert.Contains(t, mainOut.String(), `"message":"Msg","key":"value"`, "written by the parent's writer")
		assert.Contains(t, extraOut.String(), `"message":"Msg","key":"value"`, "written by the writer from the ctx")
		assert.Len(t, subLog.Config().WriterConfigs(), 2)
		mainOut.Reset()
	})

	t.Run("writer config already in parent", func(t *testing.T) {
		ctx := ContextWithAdditionalWriterConfigs(context.Background(), mainConfig)
		subLog, _ := log.With().Str("key", "value").SubLoggerContext(ctx)

		assert.Same(t, log.Config(), subLog.Config(), "parent config is kept")
		subLog.Info("Msg").Log()
		assert.Equal(t, 1, strings.Count(mainOut.String(), `"message":"Msg"`), "written once")
		mainOut.Reset()
	})
}

func TestMessageManyWriters(t *testing.T) {
	timestamp, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
	ctx := ContextWithTimestamp(context.Background(), timestamp)

	// Create 5 writers (exceeds writersArray[4] capacity)
	var outputs [5]*bytes.Buffer
	var writerConfigs []WriterConfig
	for i := range 5 {
		outputs[i] = bytes.NewBuffer(nil)
		writerConfigs = append(writerConfigs, NewJSONWriterConfig(outputs[i], nil))
	}

	config := NewConfig(&DefaultLevels, AllLevelsActive, writerConfigs...)
	log := NewLogger(config)

	log.NewMessage(ctx, log.Config().InfoLevel(), "Test message").
		Str("key", "value").
		Log()

	// Verify all 5 writers received the message
	for i, out := range outputs {
		assert.Contains(t, out.String(), `"message":"Test message"`, "writer %d", i)
		assert.Contains(t, out.String(), `"key":"value"`, "writer %d", i)
	}
}
