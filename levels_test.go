package golog

import (
	"reflect"
	"testing"
)

func TestLevels_NamesSorted(t *testing.T) {
	names := DefaultLevels.NamesSorted()
	expected := []string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL"}
	if !reflect.DeepEqual(names, expected) {
		t.Errorf("DefaultLevels.LevelNames() = %v, want %v", names, expected)
	}
}
