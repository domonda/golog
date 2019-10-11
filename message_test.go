package golog

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/domonda/go-types/uu"
)

func TestMessage(t *testing.T) {
	at, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")

	format := &Format{
		TimestampFormat: "2006-01-02 15:04:05",
		TimestampKey:    "time",
		LevelKey:        "level",
		MessageKey:      "message",
	}

	textOutput := bytes.NewBuffer(nil)
	textFormatter := NewTextFormatter(textOutput, format, NoColorizer)

	jsonOutput := bytes.NewBuffer(nil)
	jsonFormatter := NewJSONFormatter(jsonOutput, format)

	log := NewLogger(DefaultLevels, LevelFilterNone, textFormatter, jsonFormatter)

	numLines := 10
	for i := 0; i < numLines; i++ {
		log.NewMessageAt(at, log.GetLevelInfo(), "My log message").Exec(writeMessage).Log()
	}

	checkOutput := func(exptectedTextLine, exptectedJSONLine string) {
		textLines := strings.Split(textOutput.String(), "\n")
		assert.Len(t, textLines, numLines+1, "strings.Split created empty last line")
		assert.Equal(t, "", textLines[len(textLines)-1], "strings.Split created empty last line")
		for _, line := range textLines[:numLines] {
			assert.Equal(t, exptectedTextLine, line)
		}

		jsonLines := strings.Split(jsonOutput.String(), "\n")
		assert.Len(t, jsonLines, numLines+1, "strings.Split created empty last line")
		assert.Equal(t, "", jsonLines[len(jsonLines)-1], "strings.Split created empty last line")
		for _, line := range jsonLines[:numLines] {
			assert.Equal(t, exptectedJSONLine, line)
			jsonObj := []byte(strings.TrimSuffix(line, ","))
			assert.True(t, json.Valid(jsonObj), "valid JSON message")
		}
	}

	checkOutput(exptectedTextMessage, exptectedJSONMessage)

	// Test sub-logger

	textOutput.Reset()
	jsonOutput.Reset()

	subLog := log.Record().Str("SuperStr", "SuperStr").Strs("SuperStrs", []string{"A", "B", "C"}).IntPtr("SuperNilInt", nil).NewLogger()
	for i := 0; i < numLines; i++ {
		subLog.NewMessageAt(at, log.GetLevelInfo(), "My log message").Exec(writeMessage).Log()
	}

	checkOutput(exptectedTextMessageSub, exptectedJSONMessageSub)

	// Test sub-sub-logger

	textOutput.Reset()
	jsonOutput.Reset()

	subLog = log.Record().UUID("RequestID", uu.IDMustFromString("62d38a15-8fc2-4520-b768-9d5d08d2c498")).NewLogger()
	subSubLog := subLog.Record().Str("SuperStr", "SuperStr").Strs("SuperStrs", []string{"A", "B", "C"}).IntPtr("SuperNilInt", nil).NewLogger()
	for i := 0; i < numLines; i++ {
		subSubLog.NewMessageAt(at, log.GetLevelInfo(), "My log message").Exec(writeMessage).Log()
	}

	checkOutput(exptectedTextMessageSubSub, exptectedJSONMessageSubSub)

	// fmt.Println(textOutput.String())
	// fmt.Println(jsonOutput.String())
	// t.FailNow()
}

func writeMessage(message *Message) {
	uuid := uu.IDMustFromString("b14882b9-bfdd-45a4-9c84-1d717211c050")
	uuids := [][16]byte{
		uu.IDMustFromString("fab60526-bf52-4ec2-9db3-f5860250de5c"),
		uu.IDMustFromString("78adb219-460c-41e9-ac39-12d4d0420aa0"),
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
		Err("Err", errors.New("this is an error!")).
		Errs("Errs", []error{errors.New("error \"A\""), errors.New("error \"B\"")}).
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
		`Err="this is an error!" ` +
		`Errs=["error \"A\"","error \"B\""] ` +
		`PrintSingle="one arg" ` +
		`PrintMulti=["false","123","0.5","string","error"] ` +
		`UUID=b14882b9-bfdd-45a4-9c84-1d717211c050 ` +
		`UUIDs=[fab60526-bf52-4ec2-9db3-f5860250de5c,78adb219-460c-41e9-ac39-12d4d0420aa0] ` +
		`JSON=[{"a":1,"b":[2,3],"c":null,"d":{"x":1.5}},null] ` +
		`InvalidJSON=`
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
		`"Err":"this is an error!",` +
		`"Errs":["error \"A\"","error \"B\""],` +
		`"PrintSingle":"one arg",` +
		`"PrintMulti":["false","123","0.5","string","error"],` +
		`"UUID":"b14882b9-bfdd-45a4-9c84-1d717211c050",` +
		`"UUIDs":["fab60526-bf52-4ec2-9db3-f5860250de5c","78adb219-460c-41e9-ac39-12d4d0420aa0"],` +
		`"JSON":[{"a":1,"b":[2,3],"c":null,"d":{"x":1.5}},null],` +
		`"InvalidJSON":null` +
		"},"
)
