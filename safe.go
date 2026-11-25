package golog

import (
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"
)

// Logging never panics or hangs because of a logged value:
//
//   - The methods that golog calls on a logged value, like Error, String,
//     Log, IsNull, MarshalJSON and MarshalText, are called by the safe...
//     functions below, which recover a panic and log it as error instead.
//   - A value that references itself is logged with a cycle error instead
//     of recursing until the stack overflows, which is a fatal error that
//     can't be recovered. See valuePath and checkFormattable.

// panicError returns the error that is logged instead of a value
// whose method panicked with p.
func panicError(method string, val, p any) error {
	return fmt.Errorf("golog: recovered panic in %s method of %T: %s", method, val, describePanic(p))
}

// describePanic returns the recovered panic value p as string:
// the result of its Error or String method, or p formatted by fmt.
// A method that panics again, also with p, is recovered and p is
// described by its type instead.
func describePanic(p any) (s string) {
	defer func() {
		if recover() != nil {
			s = fmt.Sprintf("panic value of type %T", p)
		}
	}()
	switch x := p.(type) {
	case string:
		return x
	case error:
		return x.Error()
	case fmt.Stringer:
		return x.String()
	}
	if err := checkFormattable(reflect.ValueOf(p), printVerb, new(valueCheckBudget)); err != nil {
		return err.Error()
	}
	return fmt.Sprint(p)
}

// cycleError returns the error that is logged instead
// of a value of type t that references itself.
func cycleError(t reflect.Type) error {
	return fmt.Errorf("golog: encountered a cycle via %s", t)
}

// safeErrorString returns err.Error(),
// or the error of a recovered panic.
func safeErrorString(err error) (s string) {
	defer func() {
		if p := recover(); p != nil {
			s = panicError("Error", err, p).Error()
		}
	}()
	return err.Error()
}

// safeString returns s.String(),
// or the error of a recovered panic.
func safeString(s fmt.Stringer) (str string) {
	defer func() {
		if p := recover(); p != nil {
			str = panicError("String", s, p).Error()
		}
	}()
	return s.String()
}

// safeIsNull returns n.IsNull(), or false if it panics,
// so the value is logged instead, which reports a panic of its methods.
func safeIsNull(n interface{ IsNull() bool }) (isNull bool) {
	defer func() {
		if recover() != nil {
			isNull = false
		}
	}()
	return n.IsNull()
}

// safeIsZero returns z.IsZero(), or false if it panics,
// so the value is logged instead, which reports a panic of its methods.
func safeIsZero(z interface{ IsZero() bool }) (isZero bool) {
	defer func() {
		if recover() != nil {
			isZero = false
		}
	}()
	return z.IsZero()
}

// safeLog calls loggable.Log(m) and returns
// the error of a recovered panic.
func safeLog(loggable Loggable, m *Message) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = panicError("Log", loggable, p)
		}
	}()
	loggable.Log(m)
	return nil
}

// safeMarshalJSON returns m.MarshalJSON(),
// or the error of a recovered panic.
func safeMarshalJSON(m json.Marshaler) (j []byte, err error) {
	defer func() {
		if p := recover(); p != nil {
			j, err = nil, panicError("MarshalJSON", m, p)
		}
	}()
	return m.MarshalJSON()
}

// safeMarshalText returns m.MarshalText(),
// or the error of a recovered panic.
func safeMarshalText(m encoding.TextMarshaler) (text []byte, err error) {
	defer func() {
		if p := recover(); p != nil {
			text, err = nil, panicError("MarshalText", m, p)
		}
	}()
	return m.MarshalText()
}

// safeJSONMarshal returns json.Marshal(val), or the error of a recovered
// panic, because encoding/json passes on a panic of a MarshalJSON method,
// or an error if encoding/json would write val forever or in exponential
// time, see checkJSONSize, which counts the values of val with budget.
func safeJSONMarshal(val any, budget *valueCheckBudget) (j []byte, err error) {
	if err := checkJSONSize(reflect.ValueOf(val), budget); err != nil {
		return nil, err
	}
	defer func() {
		if p := recover(); p != nil {
			j, err = nil, fmt.Errorf("golog: recovered panic while marshalling %T as JSON: %s", val, safeSprint(p))
		}
	}()
	return json.Marshal(val)
}

// writeError writes err to w, or null for a nil error or a nil pointer,
// whose Error method with value receiver would panic.
// A panic of the Error method is written as error instead.
func writeError(w Writer, err error) {
	if err == nil || isNilPointer(err) {
		w.WriteNil()
		return
	}
	defer func() {
		if p := recover(); p != nil {
			w.WriteError(panicError("Error", err, p))
		}
	}()
	w.WriteError(err)
}

// isNilPointer returns true if val is a nil pointer.
func isNilPointer(val any) bool {
	v := reflect.ValueOf(val)
	return v.Kind() == reflect.Pointer && v.IsNil()
}

// valueRef identifies a pointer, slice or map by its address and type,
// and a slice also by its length, because a sub-slice and an embedded
// struct at offset zero have the same address as what contains them.
type valueRef struct {
	ptr uintptr
	len int
	typ reflect.Type
}

// valuePath holds the pointers, slices and maps that are being written
// around a value, to find a value that references itself, and counts
// the Log calls of the Loggables that are logged nested in each other
// and the slices, arrays and maps that are written in it.
type valuePath struct {
	refs []valueRef
	// inElements is true for the values that are elements of slices,
	// arrays and maps, nested in no Loggable inside them,
	// see appendElementsJSON
	inElements bool
	// work is shared by all paths of the values of a log message,
	// see Message.loggablePath, or of an outermost value written
	// without a message, like by an Any attrib
	work *valuePathWork
}

// valuePathWork counts what is written for the paths that share it,
// see valuePath.work, to limit it.
type valuePathWork struct {
	// logs counts the Log calls of Loggables
	logs int
	// elems counts the slices, arrays and maps written element by element
	elems int
	// values counts the elements of the slices, arrays and maps
	values int
	// marshalDepth counts the Attribs.MarshalJSON calls nested in
	// each other, marshaled holds the Attribs that were marshalled,
	// and repeats counts the calls for Attribs that were marshalled
	// before, see Attribs.MarshalJSON
	marshalDepth int
	marshaled    map[valueRef]struct{}
	repeats      int
	// checked is the budget of checkFormattable and checkJSONSize
	checked valueCheckBudget
}

// maxValuePathLen is the maximum number of pointers, slices, arrays,
// maps and Loggables that are written nested in each other, so a value that is
// nested deeper is written with an error instead of exhausting the stack.
const maxValuePathLen = 256

// maxNestedLogs is the maximum number of Log calls of the Loggables
// that share a valuePathWork, like the Loggables of a log message,
// so Loggables that log themselves several times end instead of taking
// exponential time, but a Loggable can log many other Loggables.
const maxNestedLogs = 100_000

// nestingError returns the error that is logged instead of
// a value of type t nested deeper than maxValuePathLen.
func nestingError(t reflect.Type) error {
	return fmt.Errorf("golog: %s nested more than %d deep", t, maxValuePathLen)
}

// enter returns the path with v, or an error if v is already on the
// path, so it references itself, or the path is too long.
// A value that is no non-nil pointer, slice or map returns p.
func (p valuePath) enter(v reflect.Value) (valuePath, error) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Map:
		if v.IsNil() {
			return p, nil
		}
	default:
		return p, nil
	}
	ref := valueRef{ptr: v.Pointer(), typ: v.Type()}
	if v.Kind() == reflect.Slice {
		ref.len = v.Len()
	}
	if slices.Contains(p.refs, ref) {
		return p, cycleError(v.Type())
	}
	if len(p.refs) >= maxValuePathLen {
		return p, nestingError(v.Type())
	}
	p.refs = append(p.refs, ref)
	return p, nil
}

// enterLoggable returns the path with the Loggable v, whose Log method
// is about to be called, or an error if v is already on the path, the
// path is too long, or more than maxNestedLogs Log calls were made on the
// paths that share its valuePathWork, like by a log message. A Loggable that is no pointer, slice or map,
// like a struct or a LoggableFunc, can't be told apart from another one
// of its type, so it only counts for the length of the path and for the
// Log calls, which end a Loggable that logs itself.
func (p valuePath) enterLoggable(v reflect.Value) (valuePath, error) {
	if p.work == nil {
		p.work = new(valuePathWork)
	}
	if p.work.logs >= maxNestedLogs {
		return p, fmt.Errorf("golog: more than %d Loggables logged inside one logged value or message, at %s", maxNestedLogs, v.Type())
	}
	p.work.logs++
	p.inElements = false
	switch v.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Map:
		if !v.IsNil() {
			return p.enter(v)
		}
	}
	if len(p.refs) >= maxValuePathLen {
		return p, nestingError(v.Type())
	}
	// len -1 never equals the ref of a pointer, slice or map
	p.refs = append(p.refs, valueRef{len: -1, typ: v.Type()})
	return p, nil
}

// enterElements returns the path with the slice, array or map v, whose
// elements are about to be written, or an error like enter, or if more
// than maxValueCheckNodes slices, arrays and maps or maxValueCheckElems
// of their elements were written on the paths that share its
// valuePathWork, like by the values of a log message, because
// a value that references the same slices many times, like through
// interfaces, would take exponential time.
func (p valuePath) enterElements(v reflect.Value) (valuePath, error) {
	if p.work == nil {
		p.work = new(valuePathWork)
	}
	if p.work.elems >= maxValueCheckNodes || p.work.values > maxValueCheckElems-v.Len() {
		return p, fmt.Errorf("golog: more than %d slices, arrays and maps or %d of their elements written inside one logged value or message, at %s", maxValueCheckNodes, maxValueCheckElems, v.Type())
	}
	p.work.elems++
	p.work.values += v.Len()
	p.inElements = true
	if v.Kind() == reflect.Array {
		// An array behind a pointer is on the path by its address, so a
		// pointer to an array that contains the pointer is a cycle.
		// Another array counts for the length of the path,
		// so arrays nested in arrays end.
		ref := valueRef{len: -2, typ: v.Type()} // len -2 never equals another ref
		if v.CanAddr() {
			ref.ptr = v.Addr().Pointer()
			if slices.Contains(p.refs, ref) {
				return p, cycleError(v.Type())
			}
		}
		if len(p.refs) >= maxValuePathLen {
			return p, nestingError(v.Type())
		}
		p.refs = append(p.refs, ref)
		return p, nil
	}
	return p.enter(v)
}

// The safe... formatting functions below pass their format and arguments
// on to the fmt functions unchanged when they don't replace an argument,
// so go vet checks the calls of the functions that call them, like
// Logger.Infof, like the calls of the fmt functions.

// safeSprint returns fmt.Sprint(vals...) with the arguments of
// safeSprintArgs, or an error string for a panic that fmt passes on.
func safeSprint(vals ...any) string {
	return sprintChecked(new(valueCheckBudget), vals...)
}

// sprintChecked is safeSprint that checks vals with budget.
func sprintChecked(budget *valueCheckBudget, vals ...any) (s string) {
	defer recoverFormatPanic(&s)
	if args := safeSprintArgs(vals, budget); args != nil {
		return fmt.Sprint(args...)
	}
	return fmt.Sprint(vals...)
}

// safeSprintln returns fmt.Sprintln(vals...) like safeSprint.
func safeSprintln(vals ...any) (s string) {
	defer recoverFormatPanic(&s)
	if args := safeSprintArgs(vals, new(valueCheckBudget)); args != nil {
		return fmt.Sprintln(args...)
	}
	return fmt.Sprintln(vals...)
}

// safeSprintf returns fmt.Sprintf(format, args...), but with an error
// for an argument that fmt could format forever or in exponential time,
// see withUnformattableArgsReplaced, and for a panic that fmt passes on.
func safeSprintf(format string, args ...any) (s string) {
	defer recoverFormatPanic(&s)
	if checked := withUnformattableArgsReplaced(format, args, false); checked != nil {
		return fmt.Sprintf(format, checked...)
	}
	return fmt.Sprintf(format, args...)
}

// safeErrorf returns fmt.Errorf(format, args...) like safeSprintf.
func safeErrorf(format string, args ...any) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("golog: recovered panic while formatting %q: %s", format, describePanic(p))
		}
	}()
	if checked := withUnformattableArgsReplaced(format, args, true); checked != nil {
		return fmt.Errorf(format, checked...)
	}
	return fmt.Errorf(format, args...)
}

// recoverFormatPanic, called deferred, sets *s to an error string
// for a panic that fmt passes on, like a panic of a String method
// with a value that panics again.
func recoverFormatPanic(s *string) {
	if p := recover(); p != nil {
		*s = "golog: recovered panic while formatting: " + describePanic(p)
	}
}

// safeSprintArgs returns a copy of vals for fmt.Sprint with the results
// of the Error and String methods, which Sprint would call, called safely,
// so a panic or a value that panics again can't escape fmt,
// and an error for a value that fmt would format forever or in
// exponential time, or nil if no value was replaced.
// A reflect.Value is replaced by the value it holds like fmt formats it.
// The values are checked with budget.
func safeSprintArgs(vals []any, budget *valueCheckBudget) []any {
	var args []any
	set := func(i int, val any) {
		if args == nil {
			args = slices.Clone(vals)
		}
		args[i] = val
	}
	// fmt.Sprint adds spaces between operands when neither is a string,
	// so a value that is no string is replaced by a formatted value,
	// which is no string either
	replace := func(i int, val any, str string) {
		if reflect.ValueOf(val).Kind() == reflect.String {
			set(i, str)
		} else {
			set(i, formatted{str})
		}
	}
	for i, val := range vals {
		if rv, ok := val.(reflect.Value); ok {
			if !rv.IsValid() || !rv.CanInterface() {
				// Formatted by fmt without methods, like an unexported field
				if err := checkFormattable(reflect.ValueOf(val), printVerb, budget); err != nil {
					replace(i, val, err.Error())
				}
				continue
			}
			val = rv.Interface()
			set(i, val)
		}
		if val == nil || isNilPointer(val) {
			// fmt formats a nil pointer as <nil>
			// if its methods panic for it
			continue
		}
		switch x := val.(type) {
		case fmt.Formatter:
			// Used by fmt instead of Error or String
		case error:
			replace(i, val, safeErrorString(x))
			continue
		case fmt.Stringer:
			replace(i, val, safeString(x))
			continue
		}
		if err := checkFormattable(reflect.ValueOf(val), printVerb, budget); err != nil {
			replace(i, val, err.Error())
		}
	}
	return args
}

// formatted is a formatted value that the fmt package formats like a
// value that is no string, because its kind is no string, so fmt.Sprint
// adds spaces around it like around the value that it replaces.
type formatted struct{ s string }

func (f formatted) String() string { return f.s }

// withUnformattableArgsReplaced returns a copy of args with an error instead
// of each value that fmt would format forever or in exponential time,
// see checkFormattable, with the verbs of format that format it,
// or nil if no value was replaced. wrapErrs is true for fmt.Errorf,
// which formats an error argument of %w like %v.
func withUnformattableArgsReplaced(format string, args []any, wrapErrs bool) []any {
	needsCheck := false
	for _, arg := range args {
		if _, ok := arg.(reflect.Value); ok || (arg != nil && typeNeedsFormatCheck(formattedType(reflect.TypeOf(arg)), true)) {
			// fmt formats a reflect.Value as the value it holds
			needsCheck = true
			break
		}
	}
	if !needsCheck {
		// Most arguments are numbers, strings and the like
		return nil
	}
	var replaced []any
	var buf [8]fmtVerbs
	var budget valueCheckBudget // Shared by all args, which fmt formats as one string
	for i, verbs := range formatArgVerbs(format, args, wrapErrs, buf[:0]) {
		for _, verb := range allFmtVerbs {
			if verbs&verb.bit() == 0 {
				continue
			}
			if err := checkFormattable(reflect.ValueOf(args[i]), verb, &budget); err != nil {
				if replaced == nil {
					replaced = slices.Clone(args)
				}
				replaced[i] = err.Error()
				break
			}
		}
	}
	return replaced
}

// formatArgVerbs returns for each of the args of the fmt format the set
// of the ways in which fmt formats it with its verbs, see fmtVerb,
// which is empty for an argument whose value isn't formatted, like by
// %T or a * width, or by a verb with an invalid argument index.
// It parses format like the fmt package: an argument that no verb
// formats is formatted like with %v as %!(EXTRA ...), unless a verb
// has an argument index. The result is appended to buf.
func formatArgVerbs(format string, args []any, wrapErrs bool, buf []fmtVerbs) []fmtVerbs {
	verbs := buf
	for range args {
		verbs = append(verbs, 0)
	}
	use := func(arg int, methods fmtMethods, followPointers bool) {
		verbs[arg] |= fmtVerb{methods: methods, followPointers: followPointers}.bit()
	}
	argNum := 0
	reordered := false
	for i := 0; i < len(format); {
		if format[i] != '%' {
			i++
			continue
		}
		i++
		sharp := false
		for ; i < len(format) && strings.IndexByte("#0+- ", format[i]) >= 0; i++ {
			sharp = sharp || format[i] == '#'
		}
		goodArgNum := true
		afterIndex := false
		// argIndex parses an explicit argument index like [2]
		// at format[i:] like the argNumber method of fmt
		argIndex := func() {
			if i >= len(format) || format[i] != '[' {
				return
			}
			reordered = true
			index, length, ok := parseFormatArgIndex(format[i:])
			i += length
			if ok && index >= 0 && index < len(args) {
				argNum = index
			} else {
				goodArgNum = false
			}
			afterIndex = ok
		}
		argIndex()
		if i < len(format) && format[i] == '*' {
			// The width is read from the argument, not formatted
			i++
			if argNum < len(args) {
				argNum++
			}
			afterIndex = false
		} else {
			var widthPresent bool
			_, widthPresent, i = parseFormatNum(format, i)
			if afterIndex && widthPresent {
				goodArgNum = false // Like %[3]2d
			}
		}
		if i+1 < len(format) && format[i] == '.' {
			i++
			if afterIndex {
				goodArgNum = false // Like %[3].2d
			}
			argIndex()
			if i < len(format) && format[i] == '*' {
				// The precision is read from the argument, not formatted
				i++
				if argNum < len(args) {
					argNum++
				}
				afterIndex = false
			} else {
				_, _, i = parseFormatNum(format, i)
			}
		}
		if !afterIndex {
			argIndex()
		}
		if i >= len(format) {
			break
		}
		verb, size := utf8.DecodeRuneInString(format[i:])
		i += size
		if verb == '%' || !goodArgNum || argNum >= len(args) {
			// Written as %, %!v(BADINDEX) or %!v(MISSING)
			// without formatting or using an argument
			continue
		}
		switch verb {
		case 'T':
			// The type of the value
		case 'p':
			// The address of a pointer, but fmt formats any other
			// value as error like %!p(T=...) without its methods
			switch reflect.ValueOf(args[argNum]).Kind() {
			case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
			default:
				use(argNum, fmtNoMethods, false)
			}
		case 'v':
			if sharp {
				use(argNum, fmtGoSyntax, false)
			} else {
				use(argNum, fmtV, false)
			}
		case 'x', 'X':
			use(argNum, fmtV, false)
		case 's', 'q':
			use(argNum, fmtV, true)
		case 'w':
			// fmt.Errorf formats an error like %v, anything else
			// is formatted as error like %!w(T=...) without methods.
			// For %#w, fmt uses only the Format and GoString methods
			// of the error and formats its fields with the verb w,
			// which formats them as error without methods.
			switch _, isErr := args[argNum].(error); {
			case !wrapErrs || !isErr:
				use(argNum, fmtNoMethods, false)
			case !sharp:
				use(argNum, fmtV, false)
			case formatsWithMethod(reflect.ValueOf(args[argNum]), fmtGoSyntax):
				use(argNum, fmtGoSyntax, false)
			default:
				use(argNum, fmtNoMethods, true)
			}
		case 'b', 'o', 'd':
			use(argNum, fmtFormatter, false)
		default:
			use(argNum, fmtFormatter, true)
		}
		argNum++
	}
	if !reordered {
		// Formatted as %!(EXTRA type=value, ...)
		for ; argNum < len(args); argNum++ {
			use(argNum, fmtV, false)
		}
	}
	return verbs
}

// parseFormatNum parses the decimal number at format[i:] like the
// parsenum function of fmt, which ends the format at a number above 1e6,
// and returns it, whether there is one, and the index after it.
func parseFormatNum(format string, i int) (num int, isNum bool, newi int) {
	for newi = i; newi < len(format) && '0' <= format[newi] && format[newi] <= '9'; newi++ {
		if num > 1e6 {
			return 0, false, len(format)
		}
		num = num*10 + int(format[newi]-'0')
		isNum = true
	}
	return num, isNum, newi
}

// parseFormatArgIndex parses the argument index like [2] at the start of
// s like the parseArgNumber function of fmt and returns it from zero, the
// length of the bracketed index, and false if it is no valid number.
func parseFormatArgIndex(s string) (index, length int, ok bool) {
	if len(s) < 3 {
		return 0, 1, false
	}
	for i := 1; i < len(s); i++ {
		if s[i] == ']' {
			num, isNum, newi := parseFormatNum(s[:i], 1)
			if !isNum || newi != i {
				return 0, i + 1, false
			}
			return num - 1, i + 1, true
		}
	}
	return 0, 1, false
}

// fmtMethods are the methods of a value that the fmt package
// uses instead of formatting the value's elements.
type fmtMethods uint8

const (
	// fmtNoMethods is for a value that fmt formats as error like
	// %!w(T=...), because it can't format it with the verb,
	// without the methods of the value and of the values in it.
	fmtNoMethods fmtMethods = iota
	// fmtFormatter is for verbs like %d: only the Format method.
	fmtFormatter
	// fmtV is for Print and verbs like %v and %s:
	// the Format, Error and String methods.
	fmtV
	// fmtGoSyntax is for %#v: the Format and GoString methods.
	fmtGoSyntax
)

// fmtVerb tells checkFormattable how the fmt package formats
// a value with a verb.
type fmtVerb struct {
	methods fmtMethods
	// followPointers is true for a verb that fmt can't format a pointer
	// with, so it formats a pointer nested in the value as error like
	// %!s(*T=&{...}) with the value that it points to, which it formats
	// like %v without methods. fmt formats other nested pointers as address.
	followPointers bool
}

// printVerb is how fmt.Print and %v format a value.
var printVerb = fmtVerb{methods: fmtV}

// allFmtVerbs holds every fmtVerb.
var allFmtVerbs = []fmtVerb{
	{fmtNoMethods, false}, {fmtNoMethods, true},
	{fmtFormatter, false}, {fmtFormatter, true},
	{fmtV, false}, {fmtV, true},
	{fmtGoSyntax, false}, {fmtGoSyntax, true},
}

// fmtVerbs is a set of fmtVerb, see fmtVerb.bit.
type fmtVerbs uint8

// bit returns the set that holds only v.
func (v fmtVerb) bit() fmtVerbs {
	b := uint(v.methods) * 2
	if v.followPointers {
		b++
	}
	return 1 << b
}

// fmtFollowsPointerTo returns true if fmt formats a pointer to a value
// of kind k, as argument or when following a nested pointer, like &{...}
// with the value that it points to instead of as address.
func fmtFollowsPointerTo(k reflect.Kind) bool {
	switch k {
	case reflect.Array, reflect.Slice, reflect.Struct, reflect.Map:
		return true
	}
	return false
}

// maxValueCheckNodes is the maximum number of slices, arrays, maps and
// structs that checkFormattable and checkJSONSize count, and of slices,
// arrays and maps that Message.Any and the Any attrib write element by
// element, because a value that references the same values many times,
// like through interfaces, would take exponential time to write. A value
// is counted for every reference, like the fmt package and encoding/json
// write it for every reference.
const maxValueCheckNodes = 1_000_000

// maxValueCheckElems is the maximum number of elements of the slices,
// arrays and maps that are counted like by maxValueCheckNodes, so slices
// with many elements that reference the same values can't take time
// and space that is the product of their lengths, but a slice of a
// million numbers is still written.
const maxValueCheckElems = 1 << 24

// maxValueCheckDepth is the maximum depth of nested values that
// checkFormattable and checkJSONSize check, so a value nested deeper,
// which the fmt package or encoding/json could only format by exhausting
// the stack, is reported as error, like encoding/json's decoder limits
// the nesting depth of the values that it decodes.
const maxValueCheckDepth = 10_000

// valueCheckBudget counts the values that checkFormattable and
// checkJSONSize checked, see maxValueCheckNodes and maxValueCheckElems.
// The zero value has checked none. The checks of the values of a log
// message share a budget, see valuePath.checkBudget, so the limits apply
// to the whole message instead of to each of its values or elements.
type valueCheckBudget struct {
	nodes, elems int
}

// spend counts a node with numElems elements and returns
// false if the budget is exhausted.
func (b *valueCheckBudget) spend(numElems int) bool {
	b.nodes++
	b.elems += numElems
	return b.nodes <= maxValueCheckNodes && b.elems <= maxValueCheckElems
}

// spendElemsOf counts the elements that are written for v, which is
// not walked, because its type can't reference values, but whose
// elements are written, which can be the same slices, arrays and maps
// many times, and returns false if the budget is exhausted.
// See numWrittenElems for forJSON and followPointer.
func (b *valueCheckBudget) spendElemsOf(v reflect.Value, forJSON, followPointer bool) bool {
	b.elems += numWrittenElems(v, forJSON, followPointer)
	return b.elems <= maxValueCheckElems
}

// numCheckedElems returns the number of elements of v that are counted
// by valueCheckBudget for a slice, array or map that is walked, whose
// elements are checked one by one: its length, but one for a byte slice
// if forJSON is true, which encoding/json writes as one string.
func numCheckedElems(v reflect.Value, forJSON bool) int {
	switch v.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		if forJSON && isPlainByteSlice(v.Type()) {
			return 1
		}
		return v.Len()
	}
	return 0
}

// numWrittenElems returns the number of elements that encoding/json, if
// forJSON is true, or else the fmt package writes for v, which is not
// walked: the elements of a slice, array or map, including the elements
// of the arrays in it, and of the arrays in a struct, see numArrayElems.
// encoding/json follows every pointer, which can't lead back to v,
// because its type is not walked, fmt one if followPointer is true.
// A value that is written by one of its methods counts as one element.
func numWrittenElems(v reflect.Value, forJSON, followPointer bool) int {
	switch {
	case forJSON:
		for v.Kind() == reflect.Pointer && !v.IsNil() {
			v = v.Elem()
		}
	case followPointer:
		if v.Kind() == reflect.Pointer && !v.IsNil() && fmtFollowsPointerTo(v.Elem().Kind()) {
			v = v.Elem()
		}
	}
	switch v.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		if writtenByMethod(v.Type(), forJSON) || (forJSON && isPlainByteSlice(v.Type())) {
			return 1
		}
		return saturatingMul(v.Len(), max(1, numArrayElems(v.Type().Elem(), forJSON)))
	case reflect.Struct:
		n := numArrayElems(v.Type(), forJSON)
		if forJSON && !writtenByMethod(v.Type(), true) {
			// encoding/json follows the non-nil pointers in the fields
			for _, index := range pointerFields(v.Type()) {
				if f, err := v.FieldByIndexErr(index); err == nil && !f.IsNil() {
					n = min(n+numWrittenElems(f, true, false), maxValueCheckElems+1)
				}
			}
		}
		return n
	}
	return 0
}

// typesPointerFields caches the results of pointerFields.
var typesPointerFields sync.Map // map[reflect.Type][][]int

// pointerFields returns the indexes of the fields of the struct type t,
// also of the struct fields in it, that are pointers to slices, arrays,
// maps or structs that encoding/json writes for every struct.
func pointerFields(t reflect.Type) [][]int {
	if fields, ok := typesPointerFields.Load(t); ok {
		return fields.([][]int)
	}
	var fields [][]int
	for i := range t.NumField() {
		f := t.Field(i)
		if !isJSONField(f) {
			continue
		}
		if writtenByMethod(f.Type, true) || writtenByMethod(derefType(f.Type), true) {
			// encoding/json writes the field with the method
			continue
		}
		switch f.Type.Kind() {
		case reflect.Pointer:
			switch derefType(f.Type).Kind() {
			case reflect.Slice, reflect.Array, reflect.Map, reflect.Struct:
				fields = append(fields, []int{i})
			}
		case reflect.Struct:
			for _, index := range pointerFields(f.Type) {
				fields = append(fields, append([]int{i}, index...))
			}
		}
	}
	typesPointerFields.Store(t, fields)
	return fields
}

// writtenByMethod returns true if encoding/json, if forJSON is true,
// or else the fmt package with %v writes a value of type t with one of
// its methods instead of element by element.
func writtenByMethod(t reflect.Type, forJSON bool) bool {
	if forJSON {
		return implementsJSONOrTextFormat(t) || implementsJSONOrTextFormat(reflect.PointerTo(t))
	}
	return t.NumMethod() > 0 &&
		(t.Implements(fmtFormatterType) || t.Implements(errorType) || t.Implements(fmtStringerType))
}

// isPlainByteSlice returns true for a slice of bytes without methods,
// which encoding/json writes as one base64 string.
func isPlainByteSlice(t reflect.Type) bool {
	return t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 && t.Elem().NumMethod() == 0
}

// numArrayElemsKey is the key of typesNumArrayElems.
type numArrayElemsKey struct {
	t       reflect.Type
	forJSON bool
}

// typesNumArrayElems caches the results of numArrayElems.
var typesNumArrayElems sync.Map // map[numArrayElemsKey]int

// numArrayElems returns the number of elements of the arrays in a value
// of type t, which is an array or a struct, also of the arrays nested in
// them, which are written for every reference to the value, by
// encoding/json if forJSON is true, or else by the fmt package.
// The arrays that are written by a method, or as struct fields that
// encoding/json leaves out, are not counted.
// Pointers are not followed, see numWrittenElems.
func numArrayElems(t reflect.Type, forJSON bool) int {
	return numArrayElemsOnPath(t, forJSON, nil)
}

func numArrayElemsOnPath(t reflect.Type, forJSON bool, path []reflect.Type) int {
	switch t.Kind() {
	case reflect.Array, reflect.Struct:
	default:
		return 0
	}
	key := numArrayElemsKey{t: t, forJSON: forJSON}
	if n, ok := typesNumArrayElems.Load(key); ok {
		return n.(int)
	}
	n := 0
	switch {
	case writtenByMethod(t, forJSON):
		// Written as one value
	case slices.Contains(path, t):
		// A type that contains itself through pointers
	case t.Kind() == reflect.Array:
		n = saturatingMul(t.Len(), max(1, numArrayElemsOnPath(t.Elem(), forJSON, append(path, t))))
	default:
		for i := range t.NumField() {
			if f := t.Field(i); !forJSON || isJSONField(f) {
				n = min(n+numArrayElemsOnPath(f.Type, forJSON, append(path, t)), maxValueCheckElems+1)
			}
		}
	}
	typesNumArrayElems.Store(key, n)
	return n
}

// saturatingMul returns a*b for non-negative a and b,
// but at most maxValueCheckElems+1, which exhausts any budget.
func saturatingMul(a, b int) int {
	if a == 0 || b == 0 {
		return 0
	}
	if a > (maxValueCheckElems+1)/b {
		return maxValueCheckElems + 1
	}
	return min(a*b, maxValueCheckElems+1)
}

// checkBudget returns the budget of the checks of the values written
// on the paths that share its valuePathWork, see valueCheckBudget.
func (p valuePath) checkBudget() *valueCheckBudget {
	if p.work == nil {
		return new(valueCheckBudget)
	}
	return &p.work.checked
}

// checkJSONSize returns an error if encoding/json would write v forever
// or in exponential time: if v references itself through pointers, maps
// or slices, which encoding/json reports as cycle, or has more values
// than maxValueCheckNodes and maxValueCheckElems allow, or is nested
// deeper than maxValueCheckDepth. Only values of types that can reference
// values of their own type or hold interfaces or nested slices or maps
// are counted, an interface is counted by the value that it holds, so a
// large value of a simple type is never checked. Like encoding/json, it
// doesn't look inside a value with a MarshalJSON or MarshalText method,
// also with pointer receiver of an addressable value, and skips the
// struct fields that encoding/json leaves out.
// The values are counted with budget.
func checkJSONSize(v reflect.Value, budget *valueCheckBudget) error {
	c := jsonCheck{budget: budget}
	switch c.check(v, 0, 0, nil) {
	case formatCyclic:
		return cycleError(c.cycleType)
	case formatTooLarge:
		return fmt.Errorf("golog: %s has too many values to marshal as JSON, more than %d nested values or %d elements", v.Type(), maxValueCheckNodes, maxValueCheckElems)
	case formatTooDeep:
		return fmt.Errorf("golog: %s is nested more than %d deep to marshal as JSON", v.Type(), maxValueCheckDepth)
	}
	return nil
}

// startDetectingCyclesAfter is the number of nested pointers, maps and
// slices after which checkJSONSize tracks them to find a cycle, like
// encoding/json does, so most values are checked without tracking.
const startDetectingCyclesAfter = 1000

// jsonCheck holds the state of checkJSONSize.
type jsonCheck struct {
	budget *valueCheckBudget
	// seen holds the pointers, maps and slices that are nested deeper
	// than startDetectingCyclesAfter around the checked value
	seen map[valueRef]struct{}
	// cycleType is the type of the value that closes a cycle
	cycleType reflect.Type
}

// check checks v nested depth deep, in ptrLevel pointers, maps and slices.
// embeddedIn holds the struct types that embed v through embedded fields,
// which encoding/json doesn't embed again.
func (c *jsonCheck) check(v reflect.Value, depth, ptrLevel int, embeddedIn []reflect.Type) formatResult {
	if !v.IsValid() {
		return formatOK
	}
	if !typeMayRecurse(v.Type()) {
		if !c.budget.spendElemsOf(v, true, false) {
			return formatTooLarge
		}
		return formatOK
	}
	if v.Kind() == reflect.Interface {
		// Counted by the value that it holds, so a large []any
		// of numbers and strings is not counted twice
		if v.IsNil() {
			return formatOK
		}
		return c.check(v.Elem(), depth, ptrLevel, nil)
	}
	if depth++; depth > maxValueCheckDepth {
		return formatTooDeep
	}
	if !c.budget.spend(numCheckedElems(v, true)) {
		return formatTooLarge
	}
	// Tested on the type instead of the value, because the Interface
	// method would copy a struct to the heap
	if v.CanInterface() && (implementsJSONOrTextFormat(v.Type()) ||
		(v.Kind() != reflect.Pointer && v.CanAddr() && implementsJSONOrTextFormat(reflect.PointerTo(v.Type())))) {
		return formatOK
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Map:
		if v.IsNil() {
			return formatOK
		}
		if ptrLevel++; ptrLevel > startDetectingCyclesAfter {
			ref := valueRef{ptr: v.Pointer(), typ: v.Type()}
			if v.Kind() == reflect.Slice {
				ref.len = v.Len()
			}
			if _, ok := c.seen[ref]; ok {
				c.cycleType = v.Type()
				return formatCyclic
			}
			if c.seen == nil {
				c.seen = make(map[valueRef]struct{})
			}
			c.seen[ref] = struct{}{}
			defer delete(c.seen, ref)
		}
	}
	switch v.Kind() {
	case reflect.Pointer:
		// An embedded pointer keeps embeddedIn
		return c.check(v.Elem(), depth, ptrLevel, embeddedIn)
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			if r := c.check(v.Index(i), depth, ptrLevel, nil); r != formatOK {
				return r
			}
		}
	case reflect.Map:
		for iter := v.MapRange(); iter.Next(); {
			if r := c.check(iter.Value(), depth, ptrLevel, nil); r != formatOK {
				return r
			}
		}
	case reflect.Struct:
		t := v.Type()
		for i := range v.NumField() {
			f := t.Field(i)
			if !isJSONField(f) {
				continue
			}
			// Like encoding/json, an embedded field with a name in its
			// json tag, or of a type that is no struct, is a regular field
			name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			if !f.Anonymous || name != "" || derefType(f.Type).Kind() != reflect.Struct {
				if r := c.check(v.Field(i), depth, ptrLevel, nil); r != formatOK {
					return r
				}
				continue
			}
			// Like encoding/json, the fields of an embedded struct type
			// that embeds this one, also through other embedded types,
			// are not embedded again
			if slices.Contains(embeddedIn, derefType(f.Type)) || derefType(f.Type) == t {
				continue
			}
			if r := c.check(v.Field(i), depth, ptrLevel, append(embeddedIn, t)); r != formatOK {
				return r
			}
		}
	}
	return formatOK
}

// isJSONField returns true if encoding/json marshals the struct field f:
// an exported field, or an embedded struct, without the tag json:"-".
func isJSONField(f reflect.StructField) bool {
	if f.Anonymous {
		t := f.Type
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if !f.IsExported() && t.Kind() != reflect.Struct {
			return false
		}
	} else if !f.IsExported() {
		return false
	}
	return f.Tag.Get("json") != "-"
}

// typesThatMayRecurse caches the results of typeMayRecurse.
var typesThatMayRecurse sync.Map // map[reflect.Type]bool

// typeMayRecurse returns true if a value of type t can hold an interface,
// or a value of a type that contains itself, through struct fields,
// pointers, slices, arrays and maps, or slices, arrays or maps whose
// elements hold slices, maps or pointers, so it can reference values
// that encoding/json writes many times.
func typeMayRecurse(t reflect.Type) bool {
	return typeMayRecurseOnPath(t, nil)
}

// typeMayRecurseOnPath is typeMayRecurse for t nested in the types of
// path. Its result doesn't depend on path: it is true only if t can
// reference a value of its own type, also through types on path, which
// reference t, so it is cached for every type that it walks.
func typeMayRecurseOnPath(t reflect.Type, path []reflect.Type) bool {
	if mayRecurse, ok := typesThatMayRecurse.Load(t); ok {
		return mayRecurse.(bool)
	}
	mayRecurse := typeMayRecurseUncached(t, path)
	typesThatMayRecurse.Store(t, mayRecurse)
	return mayRecurse
}

func typeMayRecurseUncached(t reflect.Type, path []reflect.Type) bool {
	switch t.Kind() {
	case reflect.Interface:
		return true
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map, reflect.Struct:
		if slices.Contains(path, t) {
			return true
		}
		path = append(path, t)
	default:
		return false
	}
	switch t.Kind() {
	case reflect.Map:
		return typeHoldsReferences(t.Key(), true) || typeHoldsReferences(t.Elem(), true) ||
			typeMayRecurseOnPath(t.Key(), path) || typeMayRecurseOnPath(t.Elem(), path)
	case reflect.Struct:
		// A struct that holds slices or maps is walked, they can be
		// the same slice in many structs. The arrays behind pointers
		// are counted by type, see numArrayElems.
		if typeHoldsReferences(t, false) {
			return true
		}
		for i := range t.NumField() {
			if typeMayRecurseOnPath(t.Field(i).Type, path) {
				return true
			}
		}
		return false
	case reflect.Slice, reflect.Array:
		return typeHoldsReferences(t.Elem(), true) || typeMayRecurseOnPath(t.Elem(), path)
	default:
		return typeMayRecurseOnPath(t.Elem(), path)
	}
}

// typeHoldsReferences returns true if a value of type t holds slices,
// maps or interfaces, or pointers if pointers is true, also in arrays
// and struct fields, so the elements of a slice, array or map of type
// t can reference the same values, which are written for every element.
func typeHoldsReferences(t reflect.Type, pointers bool) bool {
	switch t.Kind() {
	case reflect.Slice, reflect.Map, reflect.Interface:
		return true
	case reflect.Pointer:
		return pointers
	case reflect.Array:
		return typeHoldsReferences(t.Elem(), pointers)
	case reflect.Struct:
		for i := range t.NumField() {
			if typeHoldsReferences(t.Field(i).Type, pointers) {
				return true
			}
		}
	}
	return false
}

// checkFormattable returns an error if the fmt package would format v
// with verb forever, because v contains itself through slices, maps,
// arrays, structs and interfaces, which fmt follows, or in exponential
// time, because v has more values than maxValueCheckNodes and
// maxValueCheckElems allow, counting a value for every reference like
// fmt formats it, or if v is nested deeper than maxValueCheckDepth.
// Pointers in v are formatted as addresses, see fmtVerb.followPointers
// for the exception, a pointer argument to an array, slice, struct or
// map is followed, and a reflect.Value is formatted as the value that
// it holds. A value whose method fmt uses with verb ends the search.
// The values are counted with budget.
func checkFormattable(v reflect.Value, verb fmtVerb, budget *valueCheckBudget) error {
	if v.IsValid() && v.Type() == reflect.TypeFor[reflect.Value]() && v.CanInterface() {
		v = v.Interface().(reflect.Value)
	}
	if !v.IsValid() || formatsWithMethod(v, verb.methods) {
		return nil
	}
	t := v.Type()
	if v.Kind() == reflect.Pointer && !v.IsNil() && fmtFollowsPointerTo(v.Elem().Kind()) {
		v = v.Elem()
	}
	c := formatCheck{verb: verb, budget: budget}
	var path [8]valueRef // The slices and maps of most values fit without allocating
	switch c.check(v, path[:0], 0) {
	case formatCyclic:
		return cycleError(t)
	case formatTooLarge:
		return fmt.Errorf("golog: %s has too many values to format, more than %d nested values or %d elements", t, maxValueCheckNodes, maxValueCheckElems)
	case formatTooDeep:
		return fmt.Errorf("golog: %s is nested more than %d deep to format", t, maxValueCheckDepth)
	}
	return nil
}

type formatResult int

const (
	formatOK formatResult = iota
	formatCyclic
	formatTooLarge
	formatTooDeep
)

// formatCheck holds the state of checkFormattable.
type formatCheck struct {
	verb   fmtVerb
	budget *valueCheckBudget
	// deep holds the slices and maps on the path
	// after the first maxFormatCheckPathLen ones
	deep map[valueRef]struct{}
}

// maxFormatCheckPathLen is the number of slices and maps
// that formatCheck searches on its path before using a map.
const maxFormatCheckPathLen = 64

// check checks v nested depth deep in the slices and maps on path.
func (c *formatCheck) check(v reflect.Value, path []valueRef, depth int) formatResult {
	if !v.IsValid() {
		return formatOK
	}
	if !typeNeedsFormatCheck(v.Type(), c.verb.followPointers) {
		if !c.budget.spendElemsOf(v, false, c.verb.followPointers) {
			return formatTooLarge
		}
		return formatOK
	}
	if formatsWithMethod(v, c.verb.methods) {
		return formatOK
	}
	if v.Kind() == reflect.Interface {
		// Counted by the value that it holds, so a large []any
		// of numbers and strings is not counted twice
		if v.IsNil() {
			return formatOK
		}
		return c.check(v.Elem(), path, depth)
	}
	if depth++; depth > maxValueCheckDepth {
		return formatTooDeep
	}
	switch v.Kind() {
	case reflect.Pointer:
		// Followed once without methods, see fmtVerb.followPointers.
		// fmt formats the pointers in it as addresses, so a value on
		// path is formatted only once more, without following pointers,
		// which is checked with a new path.
		if !c.verb.followPointers || v.IsNil() || !fmtFollowsPointerTo(v.Elem().Kind()) {
			return formatOK
		}
		outer, outerDeep := c.verb, c.deep
		c.verb, c.deep = fmtVerb{methods: fmtNoMethods}, nil
		r := c.check(v.Elem(), nil, depth)
		c.verb, c.deep = outer, outerDeep
		return r
	}
	if !c.budget.spend(numCheckedElems(v, false)) {
		return formatTooLarge
	}
	switch v.Kind() {
	case reflect.Slice, reflect.Map:
		if v.IsNil() {
			return formatOK
		}
		ref := valueRef{ptr: v.Pointer(), typ: v.Type()}
		if v.Kind() == reflect.Slice {
			ref.len = v.Len()
		}
		if _, ok := c.deep[ref]; ok || slices.Contains(path, ref) {
			return formatCyclic
		}
		if len(path) < maxFormatCheckPathLen {
			path = append(path, ref)
		} else {
			// Looked up in a map instead of searching a long path
			if c.deep == nil {
				c.deep = make(map[valueRef]struct{})
			}
			c.deep[ref] = struct{}{}
			defer delete(c.deep, ref)
		}
		if v.Kind() == reflect.Map {
			for iter := v.MapRange(); iter.Next(); {
				if r := c.check(iter.Key(), path, depth); r != formatOK {
					return r
				}
				if r := c.check(iter.Value(), path, depth); r != formatOK {
					return r
				}
			}
		} else {
			for i := range v.Len() {
				if r := c.check(v.Index(i), path, depth); r != formatOK {
					return r
				}
			}
		}

	case reflect.Array:
		for i := range v.Len() {
			if r := c.check(v.Index(i), path, depth); r != formatOK {
				return r
			}
		}

	case reflect.Struct:
		for i := range v.NumField() {
			if r := c.check(v.Field(i), path, depth); r != formatOK {
				return r
			}
		}
	}
	return formatOK
}

// formatsWithMethod returns true if fmt formats v with one of
// its methods, which it can't call for an unexported field.
// It tests the type of v instead of copying v with its Interface method,
// and the value that an interface holds, which fmt formats with its methods.
func formatsWithMethod(v reflect.Value, methods fmtMethods) bool {
	if methods == fmtNoMethods || !v.CanInterface() {
		return false
	}
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}
	t := v.Type()
	if t.NumMethod() == 0 {
		return false // Fast path for most types
	}
	if t.Implements(fmtFormatterType) {
		return true
	}
	switch methods {
	case fmtV:
		return t.Implements(errorType) || t.Implements(fmtStringerType)
	case fmtGoSyntax:
		return t.Implements(fmtGoStringerType)
	}
	return false
}

var (
	fmtFormatterType  = reflect.TypeFor[fmt.Formatter]()
	fmtStringerType   = reflect.TypeFor[fmt.Stringer]()
	fmtGoStringerType = reflect.TypeFor[fmt.GoStringer]()
)

// formattedType returns the type of the value that fmt formats for an
// argument of type t: the type that a pointer argument points to,
// because fmt formats a pointer to a struct, slice, array or map
// like &{...} with the value, but only the pointer of a pointer
// that is nested in a value or points to another pointer.
func formattedType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Pointer {
		return t.Elem()
	}
	return t
}

// formatCheckTypeKey is the key of typesThatNeedFormatCheck.
type formatCheckTypeKey struct {
	t              reflect.Type
	followPointers bool
}

// typesThatNeedFormatCheck caches the results of typeNeedsFormatCheck
// for the types that it walks.
var typesThatNeedFormatCheck sync.Map // map[formatCheckTypeKey]bool

// typeNeedsFormatCheck returns false if no value of type t can contain
// itself in a way that fmt follows: through an interface, or a slice
// or map type that contains itself, or through a pointer if fmt follows
// pointers, see fmtVerb.followPointers, and if no value of type t has
// slices, arrays or maps whose elements can reference the same values,
// which fmt formats for every reference. It returns true for any type
// that has to be checked by value, so a []int or a struct of numbers
// and strings is not walked.
func typeNeedsFormatCheck(t reflect.Type, followPointers bool) bool {
	switch t.Kind() {
	case reflect.Interface:
		return true
	case reflect.Pointer:
		if !followPointers {
			return false
		}
	case reflect.Slice, reflect.Map, reflect.Array, reflect.Struct:
	default:
		return false
	}
	// Cached, because it is called for every argument and value
	// that is checked, and walks the fields of the type
	key := formatCheckTypeKey{t: t, followPointers: followPointers}
	if needsCheck, ok := typesThatNeedFormatCheck.Load(key); ok {
		return needsCheck.(bool)
	}
	needsCheck := typeNeedsFormatCheckUncached(t, followPointers)
	typesThatNeedFormatCheck.Store(key, needsCheck)
	return needsCheck
}

// typeNeedsFormatCheckUncached is typeNeedsFormatCheck without cache.
// A slice, array, map or struct needs to be checked if it holds slices,
// maps or interfaces, or pointers if fmt follows them, through which it
// can contain itself or reference the same values many times.
func typeNeedsFormatCheckUncached(t reflect.Type, followPointers bool) bool {
	switch t.Kind() {
	case reflect.Interface:
		return true
	case reflect.Pointer:
		// Followed once, formatted without following pointers
		return followPointers && typeNeedsFormatCheckUncached(t.Elem(), false)
	case reflect.Slice, reflect.Array:
		return typeHoldsReferences(t.Elem(), followPointers)
	case reflect.Map:
		return typeHoldsReferences(t.Key(), followPointers) || typeHoldsReferences(t.Elem(), followPointers)
	case reflect.Struct:
		return typeHoldsReferences(t, followPointers)
	}
	return false
}
