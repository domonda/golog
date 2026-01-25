package golog

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
