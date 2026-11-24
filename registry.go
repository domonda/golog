package golog

import (
	"maps"
	"slices"
	"strings"
	"sync"
)

type Registry struct {
	mutex          sync.RWMutex
	pkgPathNames   map[string]string
	pkgNameConfigs map[string]*DerivedConfig
	pkgPathConfigs map[string]*DerivedConfig
}

func (r *Registry) AddPackageConfig(pkgPath, pkgName string, config *DerivedConfig) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.pkgPathNames == nil {
		r.pkgPathNames = make(map[string]string)
		r.pkgNameConfigs = make(map[string]*DerivedConfig)
		r.pkgPathConfigs = make(map[string]*DerivedConfig)
	}

	if _, exists := r.pkgNameConfigs[pkgName]; exists {
		// Panicing because AddPackageConfig is one time global
		// setup before any other error handlers
		panic("config for package name already added: " + pkgName)
	}
	if _, exists := r.pkgPathConfigs[pkgPath]; exists {
		// Panicing because AddPackageConfig is one time global
		// setup before any other error handlers
		panic("config for package path already added: " + pkgPath)
	}

	r.pkgPathNames[pkgPath] = pkgName
	r.pkgNameConfigs[pkgName] = config
	r.pkgPathConfigs[pkgPath] = config
}

func (r *Registry) ConfigOrNilByPackageName(pkgName string) *DerivedConfig {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.pkgNameConfigs[pkgName]
}

func (r *Registry) ConfigOrNilByPackagePath(pkgPath string) *DerivedConfig {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.pkgPathConfigs[pkgPath]
}

// PackagesSortedByName returns the paths and names of the registered
// packages sorted by name, where paths[i] is the path of names[i].
func (r *Registry) PackagesSortedByName() (paths, names []string) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Package names are unique, see AddPackageConfig
	paths = slices.SortedFunc(maps.Keys(r.pkgPathNames), func(a, b string) int {
		return strings.Compare(r.pkgPathNames[a], r.pkgPathNames[b])
	})
	names = make([]string, len(paths))
	for i, path := range paths {
		names[i] = r.pkgPathNames[path]
	}
	return paths, names
}

func (r *Registry) Clear() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	clear(r.pkgPathNames)
	clear(r.pkgNameConfigs)
	clear(r.pkgPathConfigs)
}
