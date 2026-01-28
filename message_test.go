package golog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
	assert.Equal(t, exptectedMempoolOutput, mempoolOutput.String())
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
Allocated []golog.Attrib len:1 cap:16
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
		assert.Equal(t, exptectedMempoolOutput, mempoolOutput.String())
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
Allocated []golog.Attrib len:1 cap:16
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
		assert.Equal(t, exptectedMempoolOutput, mempoolOutput.String())
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
Allocated []golog.Attrib len:1 cap:16
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
		assert.Equal(t, exptectedMempoolOutput, mempoolOutput.String())
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
		Int("int", 2). // Logs int=2 because the previous write of int=1 is not checked
		Log()

	textMsg = `2006-01-02 15:04:05 |INFO | pkg: Msg int=1 int=2` + "\n"
	jsonMsg = `{"time":"2006-01-02 15:04:05","level":"INFO","message":"pkg: Msg","int":1,"int":2}` + "\n"

	assert.Equal(t, textMsg, textOut.String())
	assert.Equal(t, jsonMsg, jsonOut.String())
	textOut.Reset()
	jsonOut.Reset()

	ctx = log.With().
		Int("int", 3). // Overwrites int value in ctx
		SubContext(ctx)

	log.NewMessageAt(context.Background(), timestamp, infoLevel, "Msg").
		Ctx(ctx).      // Logs int=3
		Int("int", 4). // Logs int=4 because the previous write of int=3 is not checked
		Log()

	textMsg = `2006-01-02 15:04:05 |INFO | pkg: Msg int=3 int=4` + "\n"
	jsonMsg = `{"time":"2006-01-02 15:04:05","level":"INFO","message":"pkg: Msg","int":3,"int":4}` + "\n"

	assert.Equal(t, textMsg, textOut.String())
	assert.Equal(t, jsonMsg, jsonOut.String())
	textOut.Reset()
	jsonOut.Reset()
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

		type BasicStruct struct {
			Name    string
			Age     int
			Active  bool
			private string //nolint:unused
		}

		s := BasicStruct{
			Name:    "John",
			Age:     30,
			Active:  true,
			private: "hidden",
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			StructFields(s).
			Log()

		assert.Contains(t, textOut.String(), `Name="John"`)
		assert.Contains(t, textOut.String(), `Age=30`)
		assert.Contains(t, textOut.String(), `Active=true`)
		assert.NotContains(t, textOut.String(), `private`)
		assert.NotContains(t, textOut.String(), `hidden`)
	})

	t.Run("redact tag", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		type UserWithSecrets struct {
			Username string
			Password string `golog:"redact"`
			APIKey   string `golog:"redact"`
			Email    string
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
			EmbeddedField string
		}

		type OuterStruct struct {
			Embedded
			OuterField string
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
			Value string
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
			NoTag    string
			Ignored  string `json:"-"`
			Empty    string `json:""`
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
		assert.NotContains(t, textOut.String(), `UserName`)
		assert.NotContains(t, textOut.String(), `NoTag`)
		assert.NotContains(t, textOut.String(), `no_tag_value`)
		assert.NotContains(t, textOut.String(), `Ignored`)
		assert.NotContains(t, textOut.String(), `ignored_value`)
		assert.NotContains(t, textOut.String(), `Empty`)
		assert.NotContains(t, textOut.String(), `empty_value`)
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

		type SecureStruct struct {
			Username string `json:"username"`
			Password string `json:"password" golog:"redact"`
			Token    string `json:"token" golog:"redact"`
		}

		s := SecureStruct{
			Username: "admin",
			Password: "super_secret",
			Token:    "tok_xyz789",
		}

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

	t.Run("whitespace in tag value", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		type WhitespaceTagStruct struct {
			Field1 string `json:"  field_one  "`
			Field2 string `json:"   "`
		}

		s := WhitespaceTagStruct{
			Field1: "value1",
			Field2: "value2",
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(s, "json").
			Log()

		// Trimmed tag value should be used, empty tag after trim should be ignored
		assert.Contains(t, textOut.String(), `field_one="value1"`)
		assert.NotContains(t, textOut.String(), `Field2`)
		assert.NotContains(t, textOut.String(), `value2`)
	})

	t.Run("redact with whitespace", func(t *testing.T) {
		textOut.Reset()
		jsonOut.Reset()

		type WhitespaceRedactStruct struct {
			Password string `json:"password" golog:" redact "`
		}

		s := WhitespaceRedactStruct{
			Password: "secret",
		}

		log.NewMessage(ctx, infoLevel, "Msg").
			TaggedStructFields(s, "json").
			Log()

		// Should still redact even with whitespace around "redact"
		assert.Contains(t, textOut.String(), `password="***REDACTED***"`)
		assert.NotContains(t, textOut.String(), `secret`)
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
