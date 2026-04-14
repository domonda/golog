package golog

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestParseStructFieldDirectives covers parseStructFieldDirectives directly
// without going through the Message/reflect pipeline. This is cheap enough to
// exhaustively cover whitespace, modifier spelling, edge cases, and forward
// compatibility (unknown modifiers).
func TestParseStructFieldDirectives(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  structFieldDirectives
	}{
		{"empty", "", structFieldDirectives{}},
		{"bare dash skips", "-", structFieldDirectives{skip: true}},
		{"dash with empty trailer is literal name", "-,", structFieldDirectives{key: "-"}},
		{"dash with modifier is literal name", "-,omitempty", structFieldDirectives{key: "-", omitempty: true}},
		{"name only", "name", structFieldDirectives{key: "name"}},
		{"name with omitempty", "name,omitempty", structFieldDirectives{key: "name", omitempty: true}},
		{"name with omitzero", "name,omitzero", structFieldDirectives{key: "name", omitzero: true}},
		{"name with omitnull", "name,omitnull", structFieldDirectives{key: "name", omitnull: true}},
		{"name with redact", "name,redact", structFieldDirectives{key: "name", redact: true}},
		{"name with redacted alias", "name,redacted", structFieldDirectives{key: "name", redact: true}},
		{"empty name with omitempty", ",omitempty", structFieldDirectives{omitempty: true}},
		{"all four modifiers", "name,redact,omitempty,omitzero,omitnull", structFieldDirectives{
			key: "name", redact: true, omitempty: true, omitzero: true, omitnull: true,
		}},
		{"unknown modifier ignored", "name,WIBBLE,omitempty", structFieldDirectives{key: "name", omitempty: true}},
		{"multiple unknown modifiers ignored", "name,foo,bar,baz", structFieldDirectives{key: "name"}},
		{"whitespace around name", "  name  ", structFieldDirectives{key: "name"}},
		{"whitespace around modifiers", "name ,  omitempty  ,  redact  ", structFieldDirectives{
			key: "name", omitempty: true, redact: true,
		}},
		{"whitespace-only name falls back to empty", "   ", structFieldDirectives{}},
		{"whitespace-only name with modifier", "   ,omitempty", structFieldDirectives{omitempty: true}},
		{"double comma tolerated", "name,,omitempty", structFieldDirectives{key: "name", omitempty: true}},
		{"trailing comma tolerated", "name,omitempty,", structFieldDirectives{key: "name", omitempty: true}},
		{"leading dash not followed by comma is skip", "-", structFieldDirectives{skip: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStructFieldDirectives(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ptrRcvNull has IsNull defined on the pointer receiver, so a plain
// ptrRcvNull value (not *ptrRcvNull) does NOT satisfy
// interface{ IsNull() bool } via direct assertion; detection requires
// taking the address of the value first.
type ptrRcvNull struct{ isnull bool }

func (p *ptrRcvNull) IsNull() bool { return p.isnull }

// ptrRcvZero has IsZero defined on the pointer receiver. Same story.
type ptrRcvZero struct{ n int }

func (p *ptrRcvZero) IsZero() bool { return p.n == 0 }

// TestShouldOmitStructField_PointerReceiver pins the detection of
// pointer-receiver IsNull and IsZero methods on non-addressable struct
// fields. Without the addressable-copy fallback in invokeIsNull and
// invokeIsZero, these would silently fall through to the generic
// reflect.Value.IsZero check and return wrong results for types whose
// IsNull / IsZero semantics differ from bit-for-bit zero.
func TestShouldOmitStructField_PointerReceiver(t *testing.T) {
	t.Run("IsNull pointer-receiver on non-addressable field, true", func(t *testing.T) {
		type S struct{ N ptrRcvNull }
		s := S{N: ptrRcvNull{isnull: true}}
		fv := reflect.ValueOf(s).Field(0)
		assert.False(t, fv.CanAddr(), "fv must be non-addressable for this test")
		got := shouldOmitStructField(fv, structFieldDirectives{omitnull: true})
		assert.True(t, got, "pointer-receiver IsNull must be detected and return true")
	})

	t.Run("IsNull pointer-receiver on non-addressable field, false", func(t *testing.T) {
		type S struct{ N ptrRcvNull }
		s := S{N: ptrRcvNull{isnull: false}}
		fv := reflect.ValueOf(s).Field(0)
		got := shouldOmitStructField(fv, structFieldDirectives{omitnull: true})
		assert.False(t, got, "IsNull false must NOT suppress")
	})

	t.Run("IsZero pointer-receiver on non-addressable field, true", func(t *testing.T) {
		type S struct{ Z ptrRcvZero }
		s := S{Z: ptrRcvZero{n: 0}}
		fv := reflect.ValueOf(s).Field(0)
		assert.False(t, fv.CanAddr(), "fv must be non-addressable for this test")
		got := isZeroValueForOmitzero(fv)
		assert.True(t, got, "pointer-receiver IsZero must be detected and return true")
	})

	t.Run("IsZero pointer-receiver on non-addressable field, false", func(t *testing.T) {
		type S struct{ Z ptrRcvZero }
		s := S{Z: ptrRcvZero{n: 42}}
		fv := reflect.ValueOf(s).Field(0)
		got := isZeroValueForOmitzero(fv)
		assert.False(t, got, "IsZero false must NOT report as zero (the method wins)")
	})

	t.Run("IsNull pointer-receiver on addressable field, true", func(t *testing.T) {
		// When the enclosing struct is reached via a pointer, fields
		// are addressable — the faster CanAddr path in invokeIsNull runs.
		type S struct{ N ptrRcvNull }
		s := &S{N: ptrRcvNull{isnull: true}}
		fv := reflect.ValueOf(s).Elem().Field(0)
		assert.True(t, fv.CanAddr(), "fv must be addressable for this test")
		got := shouldOmitStructField(fv, structFieldDirectives{omitnull: true})
		assert.True(t, got, "pointer-receiver IsNull on addressable field must work")
	})

	t.Run("IsZero pointer-receiver on addressable field, true", func(t *testing.T) {
		type S struct{ Z ptrRcvZero }
		s := &S{Z: ptrRcvZero{n: 0}}
		fv := reflect.ValueOf(s).Elem().Field(0)
		assert.True(t, fv.CanAddr(), "fv must be addressable for this test")
		got := isZeroValueForOmitzero(fv)
		assert.True(t, got, "pointer-receiver IsZero on addressable field must work")
	})

	t.Run("invokeIsNull returns ok=false when no method at all", func(t *testing.T) {
		type plain struct{ x int }
		fv := reflect.ValueOf(plain{x: 7})
		_, ok := invokeIsNull(fv)
		assert.False(t, ok, "plain type with no IsNull method must return ok=false")
	})

	t.Run("invokeIsZero returns ok=false when no method at all", func(t *testing.T) {
		type plain struct{ x int }
		fv := reflect.ValueOf(plain{x: 7})
		_, ok := invokeIsZero(fv)
		assert.False(t, ok, "plain type with no IsZero method must return ok=false")
	})
}

// TestShouldOmitStructField_NilPointer pins the nil-pointer short-circuit
// in shouldOmitStructField and isZeroValueForOmitzero. Without the short-
// circuit, a nil *Timestamp with ,omitnull (or a nil *time.Time with
// ,omitzero) panics because the value-receiver IsNull / IsZero method
// auto-dereferences the nil pointer.
func TestShouldOmitStructField_NilPointer(t *testing.T) {
	t.Run("nil *Timestamp with omitnull", func(t *testing.T) {
		type S struct{ P *Timestamp }
		fv := reflect.ValueOf(S{P: nil}).Field(0)
		// Pre-fix: this call panicked. Must now return true (suppress).
		got := shouldOmitStructField(fv, structFieldDirectives{omitnull: true})
		assert.True(t, got, "nil *Timestamp with omitnull must suppress, not panic")
	})

	t.Run("nil *time.Time with omitzero", func(t *testing.T) {
		type S struct{ T *time.Time }
		fv := reflect.ValueOf(S{T: nil}).Field(0)
		got := shouldOmitStructField(fv, structFieldDirectives{omitzero: true})
		assert.True(t, got, "nil *time.Time with omitzero must suppress, not panic")
	})

	t.Run("nil *time.Time with omitnull falls back via nil check", func(t *testing.T) {
		// *time.Time has no IsNull method, so omitnull falls through to the
		// omitzero path. The nil-pointer short-circuit in
		// isZeroValueForOmitzero must catch this before IsZero is called.
		type S struct{ T *time.Time }
		fv := reflect.ValueOf(S{T: nil}).Field(0)
		got := shouldOmitStructField(fv, structFieldDirectives{omitnull: true})
		assert.True(t, got, "nil *time.Time with omitnull must suppress via omitzero fallback, not panic")
	})

	t.Run("nil interface with omitnull", func(t *testing.T) {
		type S struct{ I any }
		fv := reflect.ValueOf(S{I: nil}).Field(0)
		got := shouldOmitStructField(fv, structFieldDirectives{omitnull: true})
		assert.True(t, got, "nil interface with omitnull must suppress, not panic")
	})

	t.Run("non-nil *Timestamp with omitnull preserves previous behavior", func(t *testing.T) {
		// Sanity: fix must not regress the non-nil path. A pointer to a
		// zero Timestamp (IsNull==true) must still suppress.
		ts := Timestamp{}
		type S struct{ P *Timestamp }
		fv := reflect.ValueOf(S{P: &ts}).Field(0)
		got := shouldOmitStructField(fv, structFieldDirectives{omitnull: true})
		assert.True(t, got, "pointer to zero Timestamp with omitnull must suppress via IsNull")

		// And a pointer to a non-zero Timestamp must NOT suppress.
		ts2 := Timestamp{Time: time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)}
		fv2 := reflect.ValueOf(S{P: &ts2}).Field(0)
		got2 := shouldOmitStructField(fv2, structFieldDirectives{omitnull: true})
		assert.False(t, got2, "pointer to non-zero Timestamp with omitnull must not suppress")
	})
}

// TestEncodingJSONIgnoresCustomModifiers pins a load-bearing assumption of
// the struct-field modifier design: `encoding/json.Marshal` must ignore
// golog-specific modifiers (redact, redacted, omitnull) silently, so that a
// tag like `json:"name,omitnull"` is valid for both packages — golog honors
// the modifier, encoding/json falls through. If a future Go release changes
// this, this test fails and the plan needs revisiting.
//
// The struct type is constructed at runtime via [reflect.StructOf] so that
// the json tags with golog-specific modifiers never appear as static struct
// tag literals in the source — both `go vet` (structtag) and staticcheck
// (SA5008) inspect AST-level struct tag declarations, and neither has any
// string literal to complain about when the tag lives in a
// [reflect.StructField] value at runtime.
func TestEncodingJSONIgnoresCustomModifiers(t *testing.T) {
	stringType := reflect.TypeFor[string]()
	structType := reflect.StructOf([]reflect.StructField{
		{Name: "A", Type: stringType, Tag: `json:"a,redact"`},
		{Name: "B", Type: stringType, Tag: `json:"b,redacted"`},
		{Name: "C", Type: stringType, Tag: `json:"c,omitnull"`},
		{Name: "D", Type: stringType, Tag: `json:"d,omitempty,omitnull,redact"`},
		{Name: "E", Type: stringType, Tag: `json:"e,omitempty"`},
	})

	fill := func(a, b, c, d, e string) any {
		v := reflect.New(structType).Elem()
		v.FieldByName("A").SetString(a)
		v.FieldByName("B").SetString(b)
		v.FieldByName("C").SetString(c)
		v.FieldByName("D").SetString(d)
		v.FieldByName("E").SetString(e)
		return v.Interface()
	}

	out, err := json.Marshal(fill("a_val", "b_val", "c_val", "d_val", "e_val"))
	if err != nil {
		t.Fatalf("encoding/json marshal failed: %v", err)
	}
	got := string(out)

	// All five fields appear with their tag names and original values —
	// the custom modifiers are no-ops for encoding/json. The only modifier
	// encoding/json actually honors here is omitempty on E, which has no
	// effect because E is non-empty.
	assert.Contains(t, got, `"a":"a_val"`)
	assert.Contains(t, got, `"b":"b_val"`)
	assert.Contains(t, got, `"c":"c_val"`)
	assert.Contains(t, got, `"d":"d_val"`)
	assert.Contains(t, got, `"e":"e_val"`)

	// Prove encoding/json would NOT emit an empty E (omitempty is real) —
	// our custom modifiers on D don't interfere with the omitempty that
	// encoding/json does understand on the same tag.
	out, err = json.Marshal(fill("a", "b", "c", "", ""))
	if err != nil {
		t.Fatalf("encoding/json marshal failed: %v", err)
	}
	got = string(out)
	assert.NotContains(t, got, `"e":`)
	// D has omitempty too — encoding/json suppresses it when empty.
	assert.NotContains(t, got, `"d":`)
}
