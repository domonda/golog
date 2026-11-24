package golog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/golog/mempool"
)

// Test Nil attrib
func TestNilAttrib(t *testing.T) {
	t.Run("NewNil creates attrib", func(t *testing.T) {
		a := NewNil("testKey")
		require.NotNil(t, a)
		assert.Equal(t, "testKey", a.Key())
		assert.Nil(t, a.Value())
		assert.Equal(t, "<nil>", a.ValueString())
	})

	t.Run("Clone creates independent copy", func(t *testing.T) {
		a := NewNil("key1")
		clone := a.Clone()
		require.NotNil(t, clone)
		assert.Equal(t, a.Key(), clone.Key())
	})

	t.Run("String returns formatted string", func(t *testing.T) {
		a := NewNil("myKey")
		assert.Equal(t, "Nil{myKey}", a.String())
	})

	t.Run("AppendJSON appends correct JSON", func(t *testing.T) {
		a := NewNil("key")
		buf := a.AppendJSON([]byte(`{`))
		assert.Equal(t, `{"key":null`, string(buf))
	})
}

// Test Any attrib
func TestAnyAttrib(t *testing.T) {
	t.Run("with string value", func(t *testing.T) {
		a := NewAny("key", "value")
		assert.Equal(t, "key", a.Key())
		assert.Equal(t, "value", a.Value())
	})

	t.Run("with int value", func(t *testing.T) {
		a := NewAny("key", 42)
		assert.Equal(t, 42, a.Value())
	})

	t.Run("with nil value", func(t *testing.T) {
		a := NewAny("key", nil)
		assert.Nil(t, a.Value())
	})

	t.Run("Clone creates copy", func(t *testing.T) {
		a := NewAny("key", "value")
		clone := a.Clone()
		assert.Equal(t, a.Key(), clone.Key())
		assert.Equal(t, a.Value(), clone.(*Any).Value())
	})

	t.Run("AppendJSON handles various types", func(t *testing.T) {
		tests := []struct {
			name     string
			val      any
			expected string
		}{
			{"nil", nil, `"key":null`},
			{"bool", true, `"key":true`},
			{"int", 42, `"key":42`},
			{"string", "hello", `"key":"hello"`},
			{"float64", 3.14, `"key":3.14`},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				a := NewAny("key", tt.val)
				buf := a.AppendJSON([]byte(`{`))
				assert.Equal(t, `{`+tt.expected, string(buf))
			})
		}
	})

	t.Run("String returns formatted output", func(t *testing.T) {
		a := NewAny("key", "val")
		assert.Contains(t, a.String(), "Any")
		assert.Contains(t, a.String(), "key")
	})
}

// Test Bool attrib
func TestBoolAttrib(t *testing.T) {
	t.Run("NewBool with true", func(t *testing.T) {
		a := NewBool("active", true)
		assert.Equal(t, "active", a.Key())
		assert.Equal(t, true, a.Value())
		assert.Equal(t, true, a.ValueBool())
		assert.Equal(t, "true", a.ValueString())
	})

	t.Run("NewBool with false", func(t *testing.T) {
		a := NewBool("active", false)
		assert.Equal(t, false, a.ValueBool())
		assert.Equal(t, "false", a.ValueString())
	})

	t.Run("Clone creates copy", func(t *testing.T) {
		a := NewBool("key", true)
		clone := a.Clone().(*Bool)
		assert.Equal(t, a.ValueBool(), clone.ValueBool())
	})

	t.Run("AppendJSON", func(t *testing.T) {
		a := NewBool("key", true)
		buf := a.AppendJSON([]byte(`{`))
		assert.Equal(t, `{"key":true`, string(buf))

		a2 := NewBool("key", false)
		buf2 := a2.AppendJSON([]byte(`{`))
		assert.Equal(t, `{"key":false`, string(buf2))
	})

	t.Run("String", func(t *testing.T) {
		a := NewBool("myBool", true)
		assert.Equal(t, `Bool{"myBool": true}`, a.String())
	})
}

// Test Bools attrib
func TestBoolsAttrib(t *testing.T) {
	t.Run("NewBools", func(t *testing.T) {
		vals := []bool{true, false, true}
		a := NewBools("flags", vals)
		assert.Equal(t, "flags", a.Key())
		assert.Equal(t, vals, a.ValueBools())
		assert.Equal(t, 3, a.Len())
	})

	t.Run("NewBoolsCopy creates copy", func(t *testing.T) {
		vals := []bool{true, false}
		a := NewBoolsCopy("flags", vals)
		vals[0] = false
		assert.True(t, a.ValueBools()[0]) // Original should be unchanged
	})

	t.Run("Clone", func(t *testing.T) {
		a := NewBools("key", []bool{true, false})
		clone := a.Clone().(*Bools)
		assert.Equal(t, a.Len(), clone.Len())
	})

	t.Run("AppendJSON", func(t *testing.T) {
		a := NewBools("key", []bool{true, false, true})
		buf := a.AppendJSON([]byte(`{`))
		assert.Equal(t, `{"key":[true,false,true]`, string(buf))
	})
}

// Test Int attrib
func TestIntAttrib(t *testing.T) {
	t.Run("NewInt", func(t *testing.T) {
		a := NewInt("count", -123)
		assert.Equal(t, "count", a.Key())
		assert.Equal(t, int64(-123), a.ValueInt())
	})

	t.Run("Clone", func(t *testing.T) {
		a := NewInt("key", 42)
		clone := a.Clone().(*Int)
		assert.Equal(t, a.ValueInt(), clone.ValueInt())
	})

	t.Run("AppendJSON", func(t *testing.T) {
		a := NewInt("key", -123)
		buf := a.AppendJSON([]byte(`{`))
		assert.Equal(t, `{"key":-123`, string(buf))
	})

	t.Run("String", func(t *testing.T) {
		a := NewInt("num", 42)
		assert.Contains(t, a.String(), "Int")
		assert.Contains(t, a.String(), "42")
	})
}

// Test Ints attrib
func TestIntsAttrib(t *testing.T) {
	t.Run("NewInts", func(t *testing.T) {
		vals := []int64{1, 2, 3}
		a := NewInts("numbers", vals)
		assert.Equal(t, vals, a.ValueInts())
		assert.Equal(t, 3, a.Len())
	})

	t.Run("NewIntsCopy with int slice", func(t *testing.T) {
		vals := []int{1, 2, 3}
		a := NewIntsCopy("numbers", vals)
		assert.Equal(t, int64(1), a.ValueInts()[0])
	})

	t.Run("NewIntsCopy with nil", func(t *testing.T) {
		a := NewIntsCopy[int]("numbers", nil)
		assert.Nil(t, a.ValueInts())
	})

	t.Run("AppendJSON", func(t *testing.T) {
		a := NewInts("key", []int64{-1, 0, 1})
		buf := a.AppendJSON([]byte(`{`))
		assert.Equal(t, `{"key":[-1,0,1]`, string(buf))
	})
}

// Test Uint attrib
func TestUintAttrib(t *testing.T) {
	t.Run("NewUint", func(t *testing.T) {
		a := NewUint("count", 123)
		assert.Equal(t, "count", a.Key())
		assert.Equal(t, uint64(123), a.ValueUint())
	})

	t.Run("Clone", func(t *testing.T) {
		a := NewUint("key", 42)
		clone := a.Clone().(*Uint)
		assert.Equal(t, a.ValueUint(), clone.ValueUint())
	})

	t.Run("AppendJSON", func(t *testing.T) {
		a := NewUint("key", 123)
		buf := a.AppendJSON([]byte(`{`))
		assert.Equal(t, `{"key":123`, string(buf))
	})
}

// Test Uints attrib
func TestUintsAttrib(t *testing.T) {
	t.Run("NewUints", func(t *testing.T) {
		vals := []uint64{1, 2, 3}
		a := NewUints("numbers", vals)
		assert.Equal(t, vals, a.ValueUints())
		assert.Equal(t, 3, a.Len())
	})

	t.Run("NewUintsCopy with nil", func(t *testing.T) {
		a := NewUintsCopy[uint]("numbers", nil)
		assert.Nil(t, a.ValueUints())
	})

	t.Run("AppendJSON", func(t *testing.T) {
		a := NewUints("key", []uint64{0, 1, 123})
		buf := a.AppendJSON([]byte(`{`))
		assert.Equal(t, `{"key":[0,1,123]`, string(buf))
	})
}

// Test Float attrib
func TestFloatAttrib(t *testing.T) {
	t.Run("NewFloat", func(t *testing.T) {
		a := NewFloat("price", 3.14)
		assert.Equal(t, "price", a.Key())
		assert.Equal(t, 3.14, a.ValueFloat())
	})

	t.Run("Clone", func(t *testing.T) {
		a := NewFloat("key", 2.71)
		clone := a.Clone().(*Float)
		assert.Equal(t, a.ValueFloat(), clone.ValueFloat())
	})

	t.Run("AppendJSON", func(t *testing.T) {
		a := NewFloat("key", -1.5)
		buf := a.AppendJSON([]byte(`{`))
		assert.Equal(t, `{"key":-1.5`, string(buf))
	})
}

// Test Floats attrib
func TestFloatsAttrib(t *testing.T) {
	t.Run("NewFloats", func(t *testing.T) {
		vals := []float64{1.1, 2.2, 3.3}
		a := NewFloats("values", vals)
		assert.Equal(t, vals, a.ValueFloats())
		assert.Equal(t, 3, a.Len())
	})

	t.Run("NewFloatsCopy with nil", func(t *testing.T) {
		a := NewFloatsCopy[float64]("values", nil)
		assert.Nil(t, a.ValueFloats())
	})

	t.Run("AppendJSON", func(t *testing.T) {
		a := NewFloats("key", []float64{-1.5, 0, 1.5})
		buf := a.AppendJSON([]byte(`{`))
		assert.Equal(t, `{"key":[-1.5,0,1.5]`, string(buf))
	})
}

// Test String attrib
func TestStringAttrib(t *testing.T) {
	t.Run("NewString", func(t *testing.T) {
		a := NewString("name", "John")
		assert.Equal(t, "name", a.Key())
		assert.Equal(t, "John", a.Value())
		assert.Equal(t, "John", a.ValueString())
	})

	t.Run("Clone", func(t *testing.T) {
		a := NewString("key", "value")
		clone := a.Clone().(*String)
		assert.Equal(t, a.ValueString(), clone.ValueString())
	})

	t.Run("AppendJSON", func(t *testing.T) {
		a := NewString("key", "hello")
		buf := a.AppendJSON([]byte(`{`))
		assert.Equal(t, `{"key":"hello"`, string(buf))
	})

	t.Run("AppendJSON with special characters", func(t *testing.T) {
		a := NewString("key", `hello "world"`)
		buf := a.AppendJSON([]byte(`{`))
		assert.Contains(t, string(buf), `"key":"hello \"world\""`)
	})

	t.Run("String", func(t *testing.T) {
		a := NewString("myKey", "myValue")
		assert.Equal(t, `String{"myKey": "myValue"}`, a.String())
	})
}

// Test Strings attrib
func TestStringsAttrib(t *testing.T) {
	t.Run("NewStrings", func(t *testing.T) {
		vals := []string{"a", "b", "c"}
		a := NewStrings("tags", vals)
		assert.Equal(t, vals, a.ValueStrings())
		assert.Equal(t, 3, a.Len())
	})

	t.Run("NewStringsCopy with nil", func(t *testing.T) {
		a := NewStringsCopy[string]("tags", nil)
		assert.Nil(t, a.ValueStrings())
	})

	t.Run("AppendJSON", func(t *testing.T) {
		a := NewStrings("key", []string{"a", "b", "c"})
		buf := a.AppendJSON([]byte(`{`))
		assert.Equal(t, `{"key":["a","b","c"]`, string(buf))
	})
}

// Test Error attrib
func TestErrorAttrib(t *testing.T) {
	t.Run("NewError with error", func(t *testing.T) {
		err := errors.New("test error")
		a := NewError("error", err)
		assert.Equal(t, "error", a.Key())
		assert.Equal(t, err, a.Value())
		assert.Equal(t, "test error", a.ValueString())
	})

	t.Run("NewError with nil", func(t *testing.T) {
		a := NewError("error", nil)
		assert.Nil(t, a.Value())
		assert.Equal(t, "<nil>", a.ValueString())
	})

	t.Run("Clone", func(t *testing.T) {
		err := errors.New("test")
		a := NewError("key", err)
		clone := a.Clone().(*Error)
		assert.Equal(t, a.ValueString(), clone.ValueString())
	})

	t.Run("AppendJSON with error", func(t *testing.T) {
		a := NewError("key", errors.New("test error"))
		buf := a.AppendJSON([]byte(`{`))
		assert.Equal(t, `{"key":"test error"`, string(buf))
	})

	t.Run("AppendJSON with nil error", func(t *testing.T) {
		a := NewError("key", nil)
		buf := a.AppendJSON([]byte(`{`))
		assert.Equal(t, `{"key":null`, string(buf))
	})
}

// Test Errors attrib
func TestErrorsAttrib(t *testing.T) {
	t.Run("NewErrors", func(t *testing.T) {
		errs := []error{errors.New("err1"), errors.New("err2")}
		a := NewErrors("errors", errs)
		assert.Equal(t, 2, a.Len())
	})

	t.Run("NewErrorsCopy", func(t *testing.T) {
		errs := []error{errors.New("err1")}
		a := NewErrorsCopy("errors", errs)
		assert.Equal(t, 1, a.Len())
	})

	t.Run("ValueString with empty errors", func(t *testing.T) {
		a := NewErrors("errors", []error{})
		assert.Equal(t, "<nil>", a.ValueString())
	})

	t.Run("AppendJSON", func(t *testing.T) {
		errs := []error{errors.New("err1"), errors.New("err2")}
		a := NewErrors("key", errs)
		buf := a.AppendJSON([]byte(`{`))
		assert.Equal(t, `{"key":["err1","err2"]`, string(buf))
	})

	t.Run("AppendJSON with nil error in slice", func(t *testing.T) {
		errs := []error{errors.New("err1"), nil, errors.New("err2")}
		a := NewErrors("key", errs)
		buf := a.AppendJSON([]byte(`{`))
		assert.Equal(t, `{"key":["err1",null,"err2"]`, string(buf))
	})
}

// Test UUID attrib
func TestUUIDAttrib(t *testing.T) {
	t.Run("NewUUID", func(t *testing.T) {
		uuid := MustParseUUID("a547276f-b02b-4e7d-b67e-c6deb07567da")
		a := NewUUID("requestID", uuid)
		assert.Equal(t, "requestID", a.Key())
		assert.Equal(t, uuid, a.ValueUUID())
		assert.Equal(t, "a547276f-b02b-4e7d-b67e-c6deb07567da", a.ValueString())
	})

	t.Run("NewUUIDv4 creates random UUID", func(t *testing.T) {
		a := NewUUIDv4("id")
		assert.Equal(t, "id", a.Key())
		assert.NotEqual(t, [16]byte{}, a.ValueUUID())
	})

	t.Run("Clone", func(t *testing.T) {
		uuid := MustParseUUID("a547276f-b02b-4e7d-b67e-c6deb07567da")
		a := NewUUID("key", uuid)
		clone := a.Clone().(*UUID)
		assert.Equal(t, a.ValueUUID(), clone.ValueUUID())
	})

	t.Run("AppendJSON", func(t *testing.T) {
		uuid := MustParseUUID("a547276f-b02b-4e7d-b67e-c6deb07567da")
		a := NewUUID("key", uuid)
		buf := a.AppendJSON([]byte(`{`))
		assert.Equal(t, `{"key":"a547276f-b02b-4e7d-b67e-c6deb07567da"`, string(buf))
	})
}

// Test UUIDs attrib
func TestUUIDsAttrib(t *testing.T) {
	t.Run("NewUUIDs", func(t *testing.T) {
		uuids := [][16]byte{
			MustParseUUID("a547276f-b02b-4e7d-b67e-c6deb07567da"),
			MustParseUUID("b547276f-b02b-4e7d-b67e-c6deb07567da"),
		}
		a := NewUUIDs("ids", uuids)
		assert.Equal(t, 2, a.Len())
		assert.Equal(t, uuids, a.ValueUUIDs())
	})

	t.Run("NewUUIDsCopy with nil", func(t *testing.T) {
		a := NewUUIDsCopy[([16]byte)]("ids", nil)
		assert.Nil(t, a.ValueUUIDs())
	})

	t.Run("ValueString", func(t *testing.T) {
		uuids := [][16]byte{
			MustParseUUID("a547276f-b02b-4e7d-b67e-c6deb07567da"),
		}
		a := NewUUIDs("ids", uuids)
		assert.Contains(t, a.ValueString(), "a547276f-b02b-4e7d-b67e-c6deb07567da")
	})

	t.Run("AppendJSON", func(t *testing.T) {
		uuids := [][16]byte{
			MustParseUUID("a547276f-b02b-4e7d-b67e-c6deb07567da"),
			MustParseUUID("b547276f-b02b-4e7d-b67e-c6deb07567da"),
		}
		a := NewUUIDs("key", uuids)
		buf := a.AppendJSON([]byte(`{`))
		expected := `{"key":["a547276f-b02b-4e7d-b67e-c6deb07567da","b547276f-b02b-4e7d-b67e-c6deb07567da"]`
		assert.Equal(t, expected, string(buf))
	})
}

// Test JSON attrib
func TestJSONAttrib(t *testing.T) {
	t.Run("NewJSON", func(t *testing.T) {
		raw := json.RawMessage(`{"nested": "value"}`)
		a := NewJSON("data", raw)
		assert.Equal(t, "data", a.Key())
		assert.Equal(t, raw, a.ValueJSON())
		assert.Equal(t, `{"nested": "value"}`, a.ValueString())
	})

	t.Run("Clone", func(t *testing.T) {
		raw := json.RawMessage(`{"key": "value"}`)
		a := NewJSON("data", raw)
		clone := a.Clone().(*JSON)
		assert.Equal(t, string(a.ValueJSON()), string(clone.ValueJSON()))
	})

	t.Run("AppendJSON", func(t *testing.T) {
		raw := json.RawMessage(`{"nested":"value"}`)
		a := NewJSON("key", raw)
		buf := a.AppendJSON([]byte(`{`))
		assert.Equal(t, `{"key":{"nested":"value"}`, string(buf))
	})

	t.Run("String", func(t *testing.T) {
		raw := json.RawMessage(`{"a":1}`)
		a := NewJSON("data", raw)
		assert.Contains(t, a.String(), "JSON")
		assert.Contains(t, a.String(), `{"a":1}`)
	})
}

// Test Attrib interface implementations
func TestAttribInterfaceCompliance(t *testing.T) {
	var _ Attrib = &Nil{}
	var _ Attrib = &Any{}
	var _ Attrib = &Bool{}
	var _ Attrib = &Int{}
	var _ Attrib = &Uint{}
	var _ Attrib = &Float{}
	var _ Attrib = &String{}
	var _ Attrib = &Error{}
	var _ Attrib = &UUID{}
	var _ Attrib = &JSON{}
}

func TestSliceAttribInterfaceCompliance(t *testing.T) {
	var _ SliceAttrib = &Bools{}
	var _ SliceAttrib = &Ints{}
	var _ SliceAttrib = &Uints{}
	var _ SliceAttrib = &Floats{}
	var _ SliceAttrib = &Strings{}
	var _ SliceAttrib = &Errors{}
	var _ SliceAttrib = &UUIDs{}
}

// attribTestCase describes the expected behavior of one Attrib value
// across every method of the Attrib interface.
type attribTestCase struct {
	name        string
	newAttrib   func() Attrib
	key         string
	value       any
	valueString string
	str         string // String() result
	json        string // AppendJSON result
	logText     string // text writer output after the message text
	logJSON     string // JSON writer output after the message text
	len         int    // SliceAttrib.Len(), -1 for scalar attribs
}

var (
	attribTestTime1 = time.Date(2024, 1, 2, 3, 4, 5, 6000000, time.UTC)
	attribTestTime2 = time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	attribTestUUID1 = MustParseUUID("a547276f-b02b-4e7d-b67e-c6deb07567da")
	attribTestUUID2 = MustParseUUID("b14882b9-bfdd-45a4-9c84-1d717211c050")
	attribTestErr1  = errors.New(`err "A"`)
	attribTestErr2  = errors.New("err B")
)

func attribTestCases() []attribTestCase {
	return []attribTestCase{
		{
			name:        "Nil",
			newAttrib:   func() Attrib { return NewNil("k") },
			key:         "k",
			value:       nil,
			valueString: "<nil>",
			str:         "Nil{k}",
			json:        `"k":null`,
			logText:     `k=nil`,
			logJSON:     `"k":null`,
			len:         -1,
		},
		{
			name:        "Any",
			newAttrib:   func() Attrib { return NewAny("k", "v") },
			key:         "k",
			value:       "v",
			valueString: `"v"`,
			str:         `Any{"k": "v"}`,
			json:        `"k":"v"`,
			logText:     `k="v"`,
			logJSON:     `"k":"v"`,
			len:         -1,
		},
		{
			name:        "Bool",
			newAttrib:   func() Attrib { return NewBool("k", true) },
			key:         "k",
			value:       true,
			valueString: "true",
			str:         `Bool{"k": true}`,
			json:        `"k":true`,
			logText:     `k=true`,
			logJSON:     `"k":true`,
			len:         -1,
		},
		{
			name:        "Bools",
			newAttrib:   func() Attrib { return NewBools("k", []bool{true, false}) },
			key:         "k",
			value:       []bool{true, false},
			valueString: "[]bool{true, false}",
			str:         `Bools{"k": []bool{true, false}}`,
			json:        `"k":[true,false]`,
			logText:     `k=[true,false]`,
			logJSON:     `"k":[true,false]`,
			len:         2,
		},
		{
			name:        "Int",
			newAttrib:   func() Attrib { return NewInt("k", -42) },
			key:         "k",
			value:       int64(-42),
			valueString: "-42",
			str:         `Int{"k": -42}`,
			json:        `"k":-42`,
			logText:     `k=-42`,
			logJSON:     `"k":-42`,
			len:         -1,
		},
		{
			name:        "Ints",
			newAttrib:   func() Attrib { return NewInts("k", []int64{-1, 2}) },
			key:         "k",
			value:       []int64{-1, 2},
			valueString: "[]int64{-1, 2}",
			str:         `Ints{"k": []int64{-1, 2}}`,
			json:        `"k":[-1,2]`,
			logText:     `k=[-1,2]`,
			logJSON:     `"k":[-1,2]`,
			len:         2,
		},
		{
			name:        "Uint",
			newAttrib:   func() Attrib { return NewUint("k", 42) },
			key:         "k",
			value:       uint64(42),
			valueString: "0x2a",
			str:         `Uint{"k": 0x2a}`,
			json:        `"k":42`,
			logText:     `k=42`,
			logJSON:     `"k":42`,
			len:         -1,
		},
		{
			name:        "Uints",
			newAttrib:   func() Attrib { return NewUints("k", []uint64{0, 42}) },
			key:         "k",
			value:       []uint64{0, 42},
			valueString: "[]uint64{0x0, 0x2a}",
			str:         `Uints{"k": []uint64{0x0, 0x2a}}`,
			json:        `"k":[0,42]`,
			logText:     `k=[0,42]`,
			logJSON:     `"k":[0,42]`,
			len:         2,
		},
		{
			name:        "Float",
			newAttrib:   func() Attrib { return NewFloat("k", -1.5) },
			key:         "k",
			value:       -1.5,
			valueString: "-1.5",
			str:         `Float{"k": -1.5}`,
			json:        `"k":-1.5`,
			logText:     `k=-1.5`,
			logJSON:     `"k":-1.5`,
			len:         -1,
		},
		{
			name:        "Floats",
			newAttrib:   func() Attrib { return NewFloats("k", []float64{-1.5, 2}) },
			key:         "k",
			value:       []float64{-1.5, 2},
			valueString: "[]float64{-1.5, 2}",
			str:         `Floats{"k": []float64{-1.5, 2}}`,
			json:        `"k":[-1.5,2]`,
			logText:     `k=[-1.5,2]`,
			logJSON:     `"k":[-1.5,2]`,
			len:         2,
		},
		{
			name:        "String",
			newAttrib:   func() Attrib { return NewString("k", `say "hi"`) },
			key:         "k",
			value:       `say "hi"`,
			valueString: `say "hi"`,
			str:         `String{"k": "say \"hi\""}`,
			json:        `"k":"say \"hi\""`,
			logText:     `k="say \"hi\""`,
			logJSON:     `"k":"say \"hi\""`,
			len:         -1,
		},
		{
			name:        "Strings",
			newAttrib:   func() Attrib { return NewStrings("k", []string{"a", "b c"}) },
			key:         "k",
			value:       []string{"a", "b c"},
			valueString: `[]string{"a", "b c"}`,
			str:         `Strings{"k": []string{"a", "b c"}}`,
			json:        `"k":["a","b c"]`,
			logText:     `k=["a","b c"]`,
			logJSON:     `"k":["a","b c"]`,
			len:         2,
		},
		{
			name:        "Error",
			newAttrib:   func() Attrib { return NewError("k", attribTestErr1) },
			key:         "k",
			value:       attribTestErr1,
			valueString: `err "A"`,
			str:         `Error{"k": "err \"A\""}`,
			json:        `"k":"err \"A\""`,
			logText:     "k=`err \"A\"`",
			logJSON:     `"k":"err \"A\""`,
			len:         -1,
		},
		{
			name:        "Error/nil",
			newAttrib:   func() Attrib { return NewError("k", nil) },
			key:         "k",
			value:       nil,
			valueString: "<nil>",
			str:         `Error{"k": "<nil>"}`,
			json:        `"k":null`,
			logText:     `k=nil`,
			logJSON:     `"k":null`,
			len:         -1,
		},
		{
			name:        "Errors",
			newAttrib:   func() Attrib { return NewErrors("k", []error{attribTestErr1, nil, attribTestErr2}) },
			key:         "k",
			value:       []error{attribTestErr1, nil, attribTestErr2},
			valueString: "err \"A\"\nerr B",
			str:         `Errors{"k": "err \"A\"\nerr B"}`,
			json:        `"k":["err \"A\"",null,"err B"]`,
			logText:     "k=[`err \"A\"`,nil,`err B`]",
			logJSON:     `"k":["err \"A\"",null,"err B"]`,
			len:         3,
		},
		{
			name:        "Errors/empty",
			newAttrib:   func() Attrib { return NewErrors("k", nil) },
			key:         "k",
			value:       []error(nil),
			valueString: "<nil>",
			str:         `Errors{"k": "<nil>"}`,
			json:        `"k":[]`,
			logText:     `k=[]`,
			logJSON:     `"k":[]`,
			len:         0,
		},
		{
			name:        "Time",
			newAttrib:   func() Attrib { return NewTime("k", attribTestTime1) },
			key:         "k",
			value:       attribTestTime1,
			valueString: "2024-01-02T03:04:05.006Z",
			str:         `Time{"k": 2024-01-02T03:04:05.006Z}`,
			json:        `"k":"2024-01-02T03:04:05.006Z"`,
			logText:     `k="2024-01-02T03:04:05.006Z"`,
			logJSON:     `"k":"2024-01-02T03:04:05.006Z"`,
			len:         -1,
		},
		{
			name:        "Times",
			newAttrib:   func() Attrib { return NewTimes("k", []time.Time{attribTestTime1, attribTestTime2}) },
			key:         "k",
			value:       []time.Time{attribTestTime1, attribTestTime2},
			valueString: "[]time.Time{time.Date(2024, time.January, 2, 3, 4, 5, 6000000, time.UTC), time.Date(2025, time.December, 31, 23, 59, 59, 0, time.UTC)}",
			str:         `Times{"k": []time.Time{time.Date(2024, time.January, 2, 3, 4, 5, 6000000, time.UTC), time.Date(2025, time.December, 31, 23, 59, 59, 0, time.UTC)}}`,
			json:        `"k":["2024-01-02T03:04:05.006Z","2025-12-31T23:59:59Z"]`,
			logText:     `k=["2024-01-02T03:04:05.006Z","2025-12-31T23:59:59Z"]`,
			logJSON:     `"k":["2024-01-02T03:04:05.006Z","2025-12-31T23:59:59Z"]`,
			len:         2,
		},
		{
			name:        "UUID",
			newAttrib:   func() Attrib { return NewUUID("k", attribTestUUID1) },
			key:         "k",
			value:       attribTestUUID1,
			valueString: "a547276f-b02b-4e7d-b67e-c6deb07567da",
			str:         `UUID{"k": a547276f-b02b-4e7d-b67e-c6deb07567da}`,
			json:        `"k":"a547276f-b02b-4e7d-b67e-c6deb07567da"`,
			logText:     `k=a547276f-b02b-4e7d-b67e-c6deb07567da`,
			logJSON:     `"k":"a547276f-b02b-4e7d-b67e-c6deb07567da"`,
			len:         -1,
		},
		{
			name:        "UUIDs",
			newAttrib:   func() Attrib { return NewUUIDs("k", [][16]byte{attribTestUUID1, attribTestUUID2}) },
			key:         "k",
			value:       [][16]byte{attribTestUUID1, attribTestUUID2},
			valueString: "[a547276f-b02b-4e7d-b67e-c6deb07567da, b14882b9-bfdd-45a4-9c84-1d717211c050]",
			str:         `UUIDs{"k": [a547276f-b02b-4e7d-b67e-c6deb07567da, b14882b9-bfdd-45a4-9c84-1d717211c050]}`,
			json:        `"k":["a547276f-b02b-4e7d-b67e-c6deb07567da","b14882b9-bfdd-45a4-9c84-1d717211c050"]`,
			logText:     `k=[a547276f-b02b-4e7d-b67e-c6deb07567da,b14882b9-bfdd-45a4-9c84-1d717211c050]`,
			logJSON:     `"k":["a547276f-b02b-4e7d-b67e-c6deb07567da","b14882b9-bfdd-45a4-9c84-1d717211c050"]`,
			len:         2,
		},
		{
			name:        "UUIDs/empty",
			newAttrib:   func() Attrib { return NewUUIDs("k", nil) },
			key:         "k",
			value:       [][16]byte(nil),
			valueString: "[]",
			str:         `UUIDs{"k": []}`,
			json:        `"k":[]`,
			logText:     `k=[]`,
			logJSON:     `"k":[]`,
			len:         0,
		},
		{
			name:        "JSON",
			newAttrib:   func() Attrib { return NewJSON("k", json.RawMessage(`{"a":[1,null]}`)) },
			key:         "k",
			value:       json.RawMessage(`{"a":[1,null]}`),
			valueString: `{"a":[1,null]}`,
			str:         `JSON{"k": {"a":[1,null]}}`,
			json:        `"k":{"a":[1,null]}`,
			logText:     `k={"a":[1,null]}`,
			logJSON:     `"k":{"a":[1,null]}`,
			len:         -1,
		},
	}
}

// TestAttribImplementations pins the observable behavior of every Attrib
// type: the accessors, the String and JSON formats used by Attribs.MarshalJSON,
// that Clone returns an equal but separate pooled instance, that Free
// returns the attrib to its pool and that Log writes the same key/value
// to the text and JSON writers.
func TestAttribImplementations(t *testing.T) {
	for _, tc := range attribTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("accessors", func(t *testing.T) {
				a := tc.newAttrib()
				defer a.Free()

				assert.Equal(t, tc.key, a.Key())
				assert.Equal(t, tc.value, a.Value())
				assert.Equal(t, tc.valueString, a.ValueString())
				assert.Equal(t, tc.str, a.String())
				assert.Equal(t, `{`+tc.json, string(a.AppendJSON([]byte(`{`))), "AppendJSON appends to the passed buffer")
				assert.True(t, json.Valid([]byte(`{`+tc.json+`}`)), "AppendJSON result is valid JSON")

				slice, isSlice := a.(SliceAttrib)
				if tc.len < 0 {
					assert.False(t, isSlice, "scalar attrib must not implement SliceAttrib")
				} else if assert.True(t, isSlice, "slice attrib must implement SliceAttrib") {
					assert.Equal(t, tc.len, slice.Len())
				}
			})

			t.Run("Clone", func(t *testing.T) {
				a := tc.newAttrib()
				defer a.Free()
				clone := a.Clone()
				defer clone.Free()

				assert.NotSame(t, a, clone, "Clone returns a new instance")
				assert.IsType(t, a, clone)
				assert.Equal(t, a.Key(), clone.Key())
				assert.Equal(t, a.Value(), clone.Value())
				assert.Equal(t, string(a.AppendJSON(nil)), string(clone.AppendJSON(nil)))
			})

			t.Run("Free", func(t *testing.T) {
				DrainAllMemPools()
				var mempoolOutput strings.Builder
				mempool.RegisterCallbacksWriterForTest(t, &mempoolOutput)

				a := tc.newAttrib()
				clone := a.Clone()
				require.Equal(t, 2, mempool.NumOutstanding(), "attrib and its clone come from the pool")
				clone.Free()
				a.Free()
				assert.Zero(t, mempool.NumOutstanding(), "Free returns the attrib to its pool")
				assert.Contains(t, mempoolOutput.String(), fmt.Sprintf("Returned %T\n", a))
			})

			t.Run("Log", func(t *testing.T) {
				ctx := ContextWithTimestamp(context.Background(), time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC))
				log, textOut, jsonOut := newTestLogger()
				a := tc.newAttrib()
				defer a.Free()

				m := log.NewMessage(ctx, log.Config().InfoLevel(), "msg")
				a.Log(m)
				m.Log()

				assert.Equal(t, "2006-01-02 15:04:05 |INFO | msg "+tc.logText+"\n", textOut.String())
				assert.Equal(t, `{"time":"2006-01-02 15:04:05","level":"INFO","message":"msg",`+tc.logJSON+"}\n", jsonOut.String())
			})
		})
	}
}

// TestAny_AppendJSON pins the JSON encoding of every dynamic value type
// Any.AppendJSON handles without reflection, plus the json.Marshal
// fallback and its error case, because Attribs.MarshalJSON relies on it.
func TestAny_AppendJSON(t *testing.T) {
	type point struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	tests := []struct {
		name string
		val  any
		want string
	}{
		{name: "nil", val: nil, want: `null`},
		{name: "bool", val: false, want: `false`},
		{name: "int", val: int(-1), want: `-1`},
		{name: "int8", val: int8(-8), want: `-8`},
		{name: "int16", val: int16(-16), want: `-16`},
		{name: "int32", val: int32(-32), want: `-32`},
		{name: "int64", val: int64(-64), want: `-64`},
		{name: "uint", val: uint(1), want: `1`},
		{name: "uint8", val: uint8(8), want: `8`},
		{name: "uint16", val: uint16(16), want: `16`},
		{name: "uint32", val: uint32(32), want: `32`},
		{name: "uint64", val: uint64(18446744073709551615), want: `18446744073709551615`},
		{name: "float32", val: float32(0.25), want: `0.25`},
		{name: "float64", val: float64(-1.5), want: `-1.5`},
		{name: "string", val: "a\"b", want: `"a\"b"`},
		// []byte like Message.StrBytes: text as string, binary as base64
		{name: "[]byte", val: []byte("ab"), want: `"ab"`},
		{name: "[]byte binary", val: []byte{0xff, 0xfe}, want: `"__4"`},
		{name: "nil []byte", val: []byte(nil), want: `null`},
		{name: "pointer to []byte", val: &attribTestBinary, want: `"__4"`},
		{name: "nil pointer to []byte", val: (*[]byte)(nil), want: `null`},
		{name: "time.Time", val: attribTestTime2, want: `"2025-12-31T23:59:59Z"`},
		// DefaultTimeFormat like the Time attrib, so sub-seconds are kept
		{name: "time.Time with sub-seconds", val: attribTestTime1, want: `"2024-01-02T03:04:05.006Z"`},
		{name: "[16]byte", val: attribTestUUID1, want: `"a547276f-b02b-4e7d-b67e-c6deb07567da"`},
		{name: "json.RawMessage", val: json.RawMessage(`{"a":1}`), want: `{"a":1}`},
		// Compacted like Message.JSON, a line break would split the log line
		{name: "pretty json.RawMessage", val: json.RawMessage("{\n  \"a\": 1\n}"), want: `{"a":1}`},
		// Null like Message.Time and Message.Any
		{name: "zero time.Time", val: time.Time{}, want: `null`},
		{name: "json.Marshal fallback", val: point{X: 1, Y: 2}, want: `{"x":1,"y":2}`},
		{name: "json.Marshal error", val: make(chan int), want: `"json: unsupported type: chan int"`},
		// A Loggable as JSON object of the attribs it logs like Message.Any,
		// and a nil pointer as null instead of calling its Log method
		{name: "Loggable", val: testLoggable{val: "v"}, want: `{"inner":"v","n":1}`},
		{name: "nil Loggable pointer", val: (*testLoggable)(nil), want: `null`},
		// An error as its message like Message.Any, encoding/json would write {}
		{name: "error", val: errors.New("boom"), want: `"boom"`},
		{name: "nil error pointer", val: (*os.PathError)(nil), want: `null`},
		{name: "nil [16]byte", val: [16]byte{}, want: `null`},
		{name: "invalid json.RawMessage", val: json.RawMessage(`{`), want: `"{"`},
		// Every element of a []any like a single value,
		// for the mixed slices that a CallbackWriter passes to its callback
		{
			name: "[]any",
			val: []any{
				nil, int64(1), "a", errors.New("boom"), attribTestUUID1, [16]byte{},
				attribTestTime2, time.Time{}, json.RawMessage("{\n \"b\": 2\n}"),
				testLoggable{val: "v"}, []byte("ab"), []any{true},
			},
			want: `[null,1,"a","boom","a547276f-b02b-4e7d-b67e-c6deb07567da",null,` +
				`"2025-12-31T23:59:59Z",null,{"b":2},{"inner":"v","n":1},"ab",[true]]`,
		},
		{name: "empty []any", val: []any{}, want: `[]`},
		// A []any as first element gets no comma before it
		{name: "nested []any", val: []any{[]any{1, 2}, "x", []any{}}, want: `[[1,2],"x",[]]`},
		// Byte slice types with a format of their own are written with it
		{name: "net.IP", val: net.IPv4(10, 1, 2, 3), want: `"10.1.2.3"`},
		{name: "net.HardwareAddr", val: net.HardwareAddr{1, 2, 3, 4, 5, 6}, want: `"01:02:03:04:05:06"`},
		{name: "nil *net.HardwareAddr", val: (*net.HardwareAddr)(nil), want: `null`},
		{name: "MarshalJSON redacts", val: testSecretJSONBytes("hunter2"), want: `"***"`},
		{name: "error redacts", val: testSecretErrorBytes("hunter2"), want: `"***"`},
		{name: "Loggable redacts", val: testSecretLogBytes("hunter2"), want: `{"secret":"***"}`},
		{name: "IsNull bytes", val: testNullableBytes{}, want: `null`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAny("k", tt.val)
			defer a.Free()

			got := string(a.AppendJSON([]byte(`{`)))
			assert.Equal(t, `{"k":`+tt.want, got)
			assert.True(t, json.Valid([]byte(got+`}`)), "AppendJSON result is valid JSON")
		})
	}
}

// TestNewCopyConstructors protects the ownership contract of the New*Copy
// constructors: the attrib must not alias the caller's slice, so later
// changes to it don't alter the attrib, a nil slice stays nil, and the
// generic variants convert the element type.
func TestNewCopyConstructors(t *testing.T) {
	type myString string
	type myUUID [16]byte

	tests := []struct {
		name string
		// newCopy creates the attrib from a fresh source slice and returns
		// a function that mutates that source slice afterwards
		newCopy func() (a Attrib, mutateSource func())
		want    any
		newNil  func() Attrib
		wantNil any
	}{
		{
			name: "NewBoolsCopy",
			newCopy: func() (Attrib, func()) {
				src := []bool{true, false}
				return NewBoolsCopy("k", src), func() { src[0] = false }
			},
			want:    []bool{true, false},
			newNil:  func() Attrib { return NewBoolsCopy("k", nil) },
			wantNil: []bool(nil),
		},
		{
			name: "NewIntsCopy",
			newCopy: func() (Attrib, func()) {
				src := []int8{-128, 127}
				return NewIntsCopy("k", src), func() { src[0] = 0 }
			},
			want:    []int64{-128, 127},
			newNil:  func() Attrib { return NewIntsCopy[int8]("k", nil) },
			wantNil: []int64(nil),
		},
		{
			name: "NewUintsCopy",
			newCopy: func() (Attrib, func()) {
				src := []uint16{0, 65535}
				return NewUintsCopy("k", src), func() { src[1] = 0 }
			},
			want:    []uint64{0, 65535},
			newNil:  func() Attrib { return NewUintsCopy[uint16]("k", nil) },
			wantNil: []uint64(nil),
		},
		{
			name: "NewFloatsCopy",
			newCopy: func() (Attrib, func()) {
				src := []float32{-1.5, 0.25}
				return NewFloatsCopy("k", src), func() { src[0] = 0 }
			},
			want:    []float64{-1.5, 0.25},
			newNil:  func() Attrib { return NewFloatsCopy[float32]("k", nil) },
			wantNil: []float64(nil),
		},
		{
			name: "NewStringsCopy",
			newCopy: func() (Attrib, func()) {
				src := []myString{"a", "b"}
				return NewStringsCopy("k", src), func() { src[0] = "x" }
			},
			want:    []string{"a", "b"},
			newNil:  func() Attrib { return NewStringsCopy[myString]("k", nil) },
			wantNil: []string(nil),
		},
		{
			name: "NewErrorsCopy",
			newCopy: func() (Attrib, func()) {
				src := []error{attribTestErr1, nil}
				return NewErrorsCopy("k", src), func() { src[0] = attribTestErr2 }
			},
			want:    []error{attribTestErr1, nil},
			newNil:  func() Attrib { return NewErrorsCopy("k", nil) },
			wantNil: []error(nil),
		},
		{
			name: "NewTimesCopy",
			newCopy: func() (Attrib, func()) {
				src := []time.Time{attribTestTime1, attribTestTime2}
				return NewTimesCopy("k", src), func() { src[0] = time.Time{} }
			},
			want:    []time.Time{attribTestTime1, attribTestTime2},
			newNil:  func() Attrib { return NewTimesCopy("k", nil) },
			wantNil: []time.Time(nil),
		},
		{
			name: "NewUUIDsCopy",
			newCopy: func() (Attrib, func()) {
				src := []myUUID{attribTestUUID1, attribTestUUID2}
				return NewUUIDsCopy("k", src), func() { src[0] = myUUID{} }
			},
			want:    [][16]byte{attribTestUUID1, attribTestUUID2},
			newNil:  func() Attrib { return NewUUIDsCopy[myUUID]("k", nil) },
			wantNil: [][16]byte(nil),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, mutateSource := tt.newCopy()
			defer a.Free()
			mutateSource()
			assert.Equal(t, "k", a.Key())
			assert.Equal(t, tt.want, a.Value(), "attrib is independent of the source slice")

			nilAttrib := tt.newNil()
			defer nilAttrib.Free()
			assert.Equal(t, tt.wantNil, nilAttrib.Value(), "nil source slice stays nil")
		})
	}
}

// TestNewUUIDv4 protects that NewUUIDv4 creates a valid random version 4
// UUID for every call instead of a constant or zero value.
func TestNewUUIDv4(t *testing.T) {
	a := NewUUIDv4("id")
	defer a.Free()
	b := NewUUIDv4("id")
	defer b.Free()

	assert.Equal(t, "id", a.Key())
	assert.NotEqual(t, a.ValueUUID(), b.ValueUUID())
	parsed, err := ParseUUID(a.ValueString())
	require.NoError(t, err)
	assert.Equal(t, a.ValueUUID(), parsed)
	assert.Equal(t, byte(0x40), a.ValueUUID()[6]&0xf0, "version 4")
	assert.Equal(t, byte(0x80), a.ValueUUID()[8]&0xc0, "RFC 4122 variant")
}

// TestAttrib_EmptyValueEdgeCases pins attribs whose value is empty or only
// nil: Errors of only nil errors must not panic in ValueString (errors.Join
// returns nil for them), and an empty JSON value must append JSON null like
// Message.JSON logs it, instead of a key without a value that makes
// Attribs.MarshalJSON return invalid JSON.
func TestAttrib_EmptyValueEdgeCases(t *testing.T) {
	errs := NewErrors("errs", []error{nil, nil})
	require.Equal(t, "<nil>", errs.ValueString())
	require.Equal(t, `Errors{"errs": "<nil>"}`, errs.String())

	for _, attrib := range []Attrib{
		NewJSON("k", nil),
		NewJSON("k", json.RawMessage{}),
		NewAny("k", json.RawMessage(nil)),
		NewAny("k", json.RawMessage{}),
	} {
		data, err := Attribs{attrib}.MarshalJSON()
		require.NoError(t, err)
		require.Equal(t, `{"k":null}`, string(data), "%s", attrib)
	}
}

var attribTestBinary = []byte{0xff, 0xfe}

// TestAny_Bytes tests that an Any attrib with bytes uses the same meaning
// for them as Message.StrBytes in ValueString and AppendJSON: a string if
// they are valid UTF-8, else base64 encoded, also for named byte types.
func TestAny_Bytes(t *testing.T) {
	type namedBytes []byte
	for _, tc := range []struct {
		name        string
		val         any
		valueString string
		json        string
	}{
		{name: "text", val: []byte("ab"), valueString: "ab", json: `{"k":"ab"}`},
		{name: "binary", val: attribTestBinary, valueString: "__4", json: `{"k":"__4"}`},
		{name: "nil", val: []byte(nil), valueString: "<nil>", json: `{"k":null}`},
		{name: "named text", val: namedBytes("ab"), valueString: "ab", json: `{"k":"ab"}`},
		{name: "named binary", val: namedBytes(attribTestBinary), valueString: "__4", json: `{"k":"__4"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := NewAny("k", tc.val)
			require.Equal(t, tc.valueString, a.ValueString())
			data, err := Attribs{a}.MarshalJSON()
			require.NoError(t, err)
			require.Equal(t, tc.json, string(data))
		})
	}
}

// TestAny_ValueString_ownFormat tests that ValueString of a byte slice
// type with a format of its own uses it instead of printing its bytes,
// which could be a secret that the type redacts.
func TestAny_ValueString_ownFormat(t *testing.T) {
	assert.Equal(t, `"10.1.2.3"`, NewAny("k", net.IPv4(10, 1, 2, 3)).ValueString())
	assert.Equal(t, `"***"`, NewAny("k", testSecretJSONBytes("hunter2")).ValueString())
	assert.Equal(t, `["a",null]`, NewAny("k", []any{"a", nil}).ValueString())
	assert.Equal(t, `<nil>`, NewAny("k", testNullableBytes{}).ValueString())
}

// TestAny_ownFormatThroughPointers tests that a byte slice type that
// redacts itself only with Error, Log or String, which a pointer to a
// pointer to it doesn't have, is written with that method by the Any attrib
// like by Message.Any, instead of its secret bytes by encoding/json.
func TestAny_ownFormatThroughPointers(t *testing.T) {
	errSecret := testSecretErrorBytes("hunter2")
	errPtr := &errSecret
	logSecret := testSecretLogBytes("hunter2")
	logPtr := &logSecret
	mac := net.HardwareAddr{1, 2, 3, 4, 5, 6}
	macPtr := &mac
	var nilPtr *testSecretErrorBytes
	ptrSecret := testPtrSecretBytes("hunter2")
	ptrSecretPtr := &ptrSecret

	for _, tc := range []struct {
		name string
		val  any
		want string
	}{
		{name: "error", val: &errPtr, want: `"***"`},
		{name: "Loggable", val: &logPtr, want: `{"secret":"***"}`},
		{name: "String", val: &macPtr, want: `"01:02:03:04:05:06"`},
		{name: "nil inner pointer", val: &nilPtr, want: `null`},
		{name: "[]any element", val: []any{&errPtr}, want: `["***"]`},
		{name: "pointer receiver String", val: &ptrSecret, want: `"***"`},
		{name: "pointer to pointer receiver String", val: &ptrSecretPtr, want: `"***"`},
		{name: "nil pointer with pointer receiver String", val: (*testPtrSecretBytes)(nil), want: `null`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := NewAny("k", tc.val)
			defer a.Free()
			assert.Equal(t, `"k":`+tc.want, string(a.AppendJSON(nil)))
			assert.NotContains(t, a.ValueString(), "aHVudGVyMg", "no base64 of the secret")
		})
	}
}

// testRecursivePtr points to its own type
type testRecursivePtr *testRecursivePtr

// TestAny_recursivePointerType tests that checking a pointer type
// that points to itself for bytes ends instead of looping forever.
func TestAny_recursivePointerType(t *testing.T) {
	assert.Equal(t, `"k":null`, string(NewAny("k", testRecursivePtr(nil)).AppendJSON(nil)))
}

// TestUUID_AppendJSON_nil tests that the UUID attribs write a nil UUID
// as null like Message.UUID and Message.UUIDs, also inside a Loggable
// that is logged as JSON object with Message.Any.
func TestUUID_AppendJSON_nil(t *testing.T) {
	assert.Equal(t, `"u":null`, string(NewUUID("u", [16]byte{}).AppendJSON(nil)))
	assert.Equal(t, `"u":[null,"a547276f-b02b-4e7d-b67e-c6deb07567da"]`, string(NewUUIDs("u", [][16]byte{{}, attribTestUUID1}).AppendJSON(nil)))

	log, _, jsonOut := newTestLogger()
	log.Info("msg").Any("l", Attribs{NewUUID("u", [16]byte{})}).Log()
	assert.Contains(t, jsonOut.String(), `"l":{"u":null}`)
}
