package golog

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevels_NamesSorted(t *testing.T) {
	names := DefaultLevels.NamesSorted()
	expected := []string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL"}
	if !reflect.DeepEqual(names, expected) {
		t.Errorf("DefaultLevels.LevelNames() = %v, want %v", names, expected)
	}
}

func TestLevels_Name(t *testing.T) {
	t.Run("returns name for known level", func(t *testing.T) {
		assert.Equal(t, "TRACE", DefaultLevels.Name(DefaultLevels.Trace))
		assert.Equal(t, "DEBUG", DefaultLevels.Name(DefaultLevels.Debug))
		assert.Equal(t, "INFO", DefaultLevels.Name(DefaultLevels.Info))
		assert.Equal(t, "WARN", DefaultLevels.Name(DefaultLevels.Warn))
		assert.Equal(t, "ERROR", DefaultLevels.Name(DefaultLevels.Error))
		assert.Equal(t, "FATAL", DefaultLevels.Name(DefaultLevels.Fatal))
	})

	t.Run("returns integer string for unknown level", func(t *testing.T) {
		assert.Equal(t, "5", DefaultLevels.Name(Level(5)))
		assert.Equal(t, "-5", DefaultLevels.Name(Level(-5)))
		assert.Equal(t, "100", DefaultLevels.Name(Level(100)))
	})
}

func TestLevels_HasName(t *testing.T) {
	t.Run("returns true for known levels", func(t *testing.T) {
		assert.True(t, DefaultLevels.HasName(DefaultLevels.Trace))
		assert.True(t, DefaultLevels.HasName(DefaultLevels.Debug))
		assert.True(t, DefaultLevels.HasName(DefaultLevels.Info))
		assert.True(t, DefaultLevels.HasName(DefaultLevels.Warn))
		assert.True(t, DefaultLevels.HasName(DefaultLevels.Error))
		assert.True(t, DefaultLevels.HasName(DefaultLevels.Fatal))
	})

	t.Run("returns false for unknown levels", func(t *testing.T) {
		assert.False(t, DefaultLevels.HasName(Level(5)))
		assert.False(t, DefaultLevels.HasName(Level(-5)))
		assert.False(t, DefaultLevels.HasName(Level(100)))
	})
}

func TestLevels_SpecificNameMethods(t *testing.T) {
	assert.Equal(t, "FATAL", DefaultLevels.FatalName())
	assert.Equal(t, "ERROR", DefaultLevels.ErrorName())
	assert.Equal(t, "WARN", DefaultLevels.WarnName())
	assert.Equal(t, "INFO", DefaultLevels.InfoName())
	assert.Equal(t, "DEBUG", DefaultLevels.DebugName())
	assert.Equal(t, "TRACE", DefaultLevels.TraceName())
}

func TestLevels_LevelOfName(t *testing.T) {
	t.Run("returns level for known name", func(t *testing.T) {
		assert.Equal(t, DefaultLevels.Trace, DefaultLevels.LevelOfName("TRACE"))
		assert.Equal(t, DefaultLevels.Debug, DefaultLevels.LevelOfName("DEBUG"))
		assert.Equal(t, DefaultLevels.Info, DefaultLevels.LevelOfName("INFO"))
		assert.Equal(t, DefaultLevels.Warn, DefaultLevels.LevelOfName("WARN"))
		assert.Equal(t, DefaultLevels.Error, DefaultLevels.LevelOfName("ERROR"))
		assert.Equal(t, DefaultLevels.Fatal, DefaultLevels.LevelOfName("FATAL"))
	})

	t.Run("returns LevelInvalid for unknown name", func(t *testing.T) {
		assert.Equal(t, LevelInvalid, DefaultLevels.LevelOfName("UNKNOWN"))
		assert.Equal(t, LevelInvalid, DefaultLevels.LevelOfName("info")) // case sensitive
		assert.Equal(t, LevelInvalid, DefaultLevels.LevelOfName(""))
	})

	t.Run("parses integer level names", func(t *testing.T) {
		// Level 5 is within range but not named
		assert.Equal(t, Level(5), DefaultLevels.LevelOfName("5"))
		assert.Equal(t, Level(-5), DefaultLevels.LevelOfName("-5"))
		assert.Equal(t, Level(0), DefaultLevels.LevelOfName("0"))
	})

	t.Run("returns LevelInvalid for out of range integers", func(t *testing.T) {
		// LevelMin is -32, LevelMax is 31
		assert.Equal(t, LevelInvalid, DefaultLevels.LevelOfName("-100"))
		assert.Equal(t, LevelInvalid, DefaultLevels.LevelOfName("100"))
	})

	t.Run("returns LevelInvalid for non-integer strings", func(t *testing.T) {
		assert.Equal(t, LevelInvalid, DefaultLevels.LevelOfName("abc"))
		assert.Equal(t, LevelInvalid, DefaultLevels.LevelOfName("1.5"))
	})
}

func TestLevels_LevelOfNameOrDefault(t *testing.T) {
	defaultLevel := DefaultLevels.Warn

	t.Run("returns level for known name", func(t *testing.T) {
		assert.Equal(t, DefaultLevels.Info, DefaultLevels.LevelOfNameOrDefault("INFO", defaultLevel))
		assert.Equal(t, DefaultLevels.Error, DefaultLevels.LevelOfNameOrDefault("ERROR", defaultLevel))
	})

	t.Run("returns default for unknown name", func(t *testing.T) {
		assert.Equal(t, defaultLevel, DefaultLevels.LevelOfNameOrDefault("UNKNOWN", defaultLevel))
		assert.Equal(t, defaultLevel, DefaultLevels.LevelOfNameOrDefault("", defaultLevel))
	})
}

func TestLevels_NameLenRange(t *testing.T) {
	t.Run("default levels", func(t *testing.T) {
		minLen, maxLen := DefaultLevels.NameLenRange()
		assert.Equal(t, 4, minLen)
		assert.Equal(t, 5, maxLen)
	})

	t.Run("custom levels", func(t *testing.T) {
		customLevels := Levels{
			Names: map[Level]string{
				0: "A",
				1: "BCDEFGHIJ",
			},
		}
		minLen, maxLen := customLevels.NameLenRange()
		assert.Equal(t, 1, minLen)
		assert.Equal(t, 9, maxLen)
	})

	t.Run("empty names", func(t *testing.T) {
		// When Names is empty, (0, 0) is returned as a sentinel value
		// indicating no names exist. This is intentional behavior.
		customLevels := Levels{
			Names: map[Level]string{},
		}
		minLen, maxLen := customLevels.NameLenRange()
		assert.Equal(t, 0, minLen)
		assert.Equal(t, 0, maxLen)
	})
}

func TestLevels_CopyWithLeftPaddedNames(t *testing.T) {
	padded := DefaultLevels.CopyWithLeftPaddedNames()

	t.Run("preserves level values", func(t *testing.T) {
		assert.Equal(t, DefaultLevels.Trace, padded.Trace)
		assert.Equal(t, DefaultLevels.Debug, padded.Debug)
		assert.Equal(t, DefaultLevels.Info, padded.Info)
		assert.Equal(t, DefaultLevels.Warn, padded.Warn)
		assert.Equal(t, DefaultLevels.Error, padded.Error)
		assert.Equal(t, DefaultLevels.Fatal, padded.Fatal)
	})

	t.Run("pads shorter names on left", func(t *testing.T) {
		// Max name length is 5 (TRACE, DEBUG, ERROR, FATAL)
		// INFO and WARN are 4 chars, should be padded to 5
		assert.Equal(t, " INFO", padded.Names[DefaultLevels.Info])
		assert.Equal(t, " WARN", padded.Names[DefaultLevels.Warn])
	})

	t.Run("does not change longer names", func(t *testing.T) {
		assert.Equal(t, "TRACE", padded.Names[DefaultLevels.Trace])
		assert.Equal(t, "DEBUG", padded.Names[DefaultLevels.Debug])
		assert.Equal(t, "ERROR", padded.Names[DefaultLevels.Error])
		assert.Equal(t, "FATAL", padded.Names[DefaultLevels.Fatal])
	})

	t.Run("creates independent copy", func(t *testing.T) {
		// Modifying padded should not affect original
		padded.Names[DefaultLevels.Info] = "MODIFIED"
		assert.Equal(t, "INFO", DefaultLevels.Names[DefaultLevels.Info])
	})
}

func TestLevels_CopyWithRightPaddedNames(t *testing.T) {
	padded := DefaultLevels.CopyWithRightPaddedNames()

	t.Run("preserves level values", func(t *testing.T) {
		assert.Equal(t, DefaultLevels.Trace, padded.Trace)
		assert.Equal(t, DefaultLevels.Debug, padded.Debug)
		assert.Equal(t, DefaultLevels.Info, padded.Info)
		assert.Equal(t, DefaultLevels.Warn, padded.Warn)
		assert.Equal(t, DefaultLevels.Error, padded.Error)
		assert.Equal(t, DefaultLevels.Fatal, padded.Fatal)
	})

	t.Run("pads shorter names on right", func(t *testing.T) {
		// Max name length is 5 (TRACE, DEBUG, ERROR, FATAL)
		// INFO and WARN are 4 chars, should be padded to 5
		assert.Equal(t, "INFO ", padded.Names[DefaultLevels.Info])
		assert.Equal(t, "WARN ", padded.Names[DefaultLevels.Warn])
	})

	t.Run("does not change longer names", func(t *testing.T) {
		assert.Equal(t, "TRACE", padded.Names[DefaultLevels.Trace])
		assert.Equal(t, "DEBUG", padded.Names[DefaultLevels.Debug])
		assert.Equal(t, "ERROR", padded.Names[DefaultLevels.Error])
		assert.Equal(t, "FATAL", padded.Names[DefaultLevels.Fatal])
	})

	t.Run("creates independent copy", func(t *testing.T) {
		// Modifying padded should not affect original
		padded.Names[DefaultLevels.Info] = "MODIFIED"
		assert.Equal(t, "INFO", DefaultLevels.Names[DefaultLevels.Info])
	})
}

func TestDefaultLevels(t *testing.T) {
	t.Run("has correct level values", func(t *testing.T) {
		assert.Equal(t, Level(-20), DefaultLevels.Trace)
		assert.Equal(t, Level(-10), DefaultLevels.Debug)
		assert.Equal(t, Level(0), DefaultLevels.Info)
		assert.Equal(t, Level(10), DefaultLevels.Warn)
		assert.Equal(t, Level(20), DefaultLevels.Error)
		assert.Equal(t, Level(30), DefaultLevels.Fatal)
	})

	t.Run("has all level names", func(t *testing.T) {
		assert.Len(t, DefaultLevels.Names, 6)
	})
}

func TestLevelConstants(t *testing.T) {
	t.Run("LevelMin and LevelMax", func(t *testing.T) {
		assert.Equal(t, Level(-32), LevelMin)
		assert.Equal(t, Level(31), LevelMax)
	})

	t.Run("LevelInvalid", func(t *testing.T) {
		assert.Equal(t, Level(-128), LevelInvalid)
	})
}

func TestCustomLevels(t *testing.T) {
	customLevels := Levels{
		Trace: -30,
		Debug: -20,
		Info:  -10,
		Warn:  0,
		Error: 10,
		Fatal: 20,
		Names: map[Level]string{
			-30: "TRC",
			-20: "DBG",
			-10: "INF",
			0:   "WRN",
			10:  "ERR",
			20:  "FTL",
		},
	}

	t.Run("uses custom names", func(t *testing.T) {
		assert.Equal(t, "TRC", customLevels.Name(customLevels.Trace))
		assert.Equal(t, "DBG", customLevels.Name(customLevels.Debug))
		assert.Equal(t, "INF", customLevels.Name(customLevels.Info))
		assert.Equal(t, "WRN", customLevels.Name(customLevels.Warn))
		assert.Equal(t, "ERR", customLevels.Name(customLevels.Error))
		assert.Equal(t, "FTL", customLevels.Name(customLevels.Fatal))
	})

	t.Run("NamesSorted returns sorted names", func(t *testing.T) {
		names := customLevels.NamesSorted()
		expected := []string{"TRC", "DBG", "INF", "WRN", "ERR", "FTL"}
		assert.Equal(t, expected, names)
	})

	t.Run("LevelOfName works with custom names", func(t *testing.T) {
		assert.Equal(t, customLevels.Trace, customLevels.LevelOfName("TRC"))
		assert.Equal(t, customLevels.Warn, customLevels.LevelOfName("WRN"))
	})
}
