package golog

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistry_PackagesSortedByName(t *testing.T) {
	var registry Registry
	config := NewDerivedConfig(new(NewConfig(&DefaultLevels, AllLevelsActive, NewJSONWriterConfig(new(lockedBuffer), nil))))
	registry.AddPackageConfig("example.com/b/pkgb", "pkgb", config)
	registry.AddPackageConfig("example.com/a/pkgc", "pkgc", config)
	registry.AddPackageConfig("example.com/c/pkga", "pkga", config)

	paths, names := registry.PackagesSortedByName()
	require.Equal(t, []string{"pkga", "pkgb", "pkgc"}, names)
	require.Equal(t, []string{"example.com/c/pkga", "example.com/b/pkgb", "example.com/a/pkgc"}, paths)
}

// TestRegistry_Concurrent adds and looks up package configs
// from many goroutines at once, for go test -race.
func TestRegistry_Concurrent(t *testing.T) {
	const numGoroutines = 16
	var registry Registry
	config := NewDerivedConfig(new(NewConfig(&DefaultLevels, AllLevelsActive, NewJSONWriterConfig(new(lockedBuffer), nil))))

	var wg sync.WaitGroup
	for g := range numGoroutines {
		wg.Go(func() {
			name := fmt.Sprintf("pkg%02d", g)
			registry.AddPackageConfig("example.com/"+name, name, config)
			require.Same(t, config, registry.ConfigOrNilByPackageName(name))
			require.Same(t, config, registry.ConfigOrNilByPackagePath("example.com/"+name))
			registry.PackagesSortedByName()
		})
	}
	wg.Wait()

	paths, names := registry.PackagesSortedByName()
	require.Len(t, names, numGoroutines)
	for i, name := range names {
		require.Equal(t, fmt.Sprintf("pkg%02d", i), name)
		require.Equal(t, "example.com/"+name, paths[i])
	}
}
