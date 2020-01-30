package golog

// Colorizer enables styling strings for color terminals
type Colorizer interface {
	ColorizeMsg(string) string
	ColorizeTimestamp(string) string
	ColorizeLevel(*Levels, Level) string
	ColorizeKey(string) string
	ColorizeNil(string) string
	ColorizeTrue(string) string
	ColorizeFalse(string) string
	ColorizeInt(string) string
	ColorizeUint(string) string
	ColorizeFloat(string) string
	ColorizeString(string) string
	ColorizeError(string) string
	ColorizeUUID(string) string
}

// NoColorizer is a no-op Colorizer returning all strings unchanged
const NoColorizer noColorizer = 0

type noColorizer int // int so it can be used as const

func (noColorizer) ColorizeMsg(str string) string                    { return str }
func (noColorizer) ColorizeTimestamp(str string) string              { return str }
func (noColorizer) ColorizeLevel(levels *Levels, level Level) string { return levels.Name(level) }
func (noColorizer) ColorizeKey(str string) string                    { return str }
func (noColorizer) ColorizeNil(str string) string                    { return str }
func (noColorizer) ColorizeTrue(str string) string                   { return str }
func (noColorizer) ColorizeFalse(str string) string                  { return str }
func (noColorizer) ColorizeInt(str string) string                    { return str }
func (noColorizer) ColorizeUint(str string) string                   { return str }
func (noColorizer) ColorizeFloat(str string) string                  { return str }
func (noColorizer) ColorizeString(str string) string                 { return str }
func (noColorizer) ColorizeError(str string) string                  { return str }
func (noColorizer) ColorizeUUID(str string) string                   { return str }

var _ Colorizer = noColorizer(0) // make sure noColorizer implements Colorizer
