package golog

import (
	"runtime"
	"sort"
	"strings"
	"sync"
)

type Registry struct {
	mutex          sync.RWMutex
	pkgPathNames   map[string]string
	pkgNameConfigs map[string]*DerivedConfig
	pkgPathConfigs map[string]*DerivedConfig
}

func (r *Registry) AddPackageConfig(pkgName string, config *DerivedConfig) (pkgPath string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.pkgPathNames == nil {
		r.pkgPathNames = make(map[string]string)
		r.pkgNameConfigs = make(map[string]*DerivedConfig)
		r.pkgPathConfigs = make(map[string]*DerivedConfig)
	}

	pkgPath = getCallingPackageImportPath(2)

	if _, exists := r.pkgNameConfigs[pkgName]; exists {
		// Panicing because AddPackageConfig is one time global
		// setup before any other error handlers
		panic("package name config already added: " + pkgName)
	}
	if _, exists := r.pkgPathConfigs[pkgPath]; exists {
		// Panicing because AddPackageConfig is one time global
		// setup before any other error handlers
		panic("package path config already added: " + pkgPath)
	}

	r.pkgPathNames[pkgPath] = pkgName
	r.pkgNameConfigs[pkgName] = config
	r.pkgPathConfigs[pkgPath] = config

	return pkgPath
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

func (r *Registry) PackagesSortedByName() (paths, names []string) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	names = make([]string, 0, len(r.pkgPathNames))
	for _, name := range r.pkgPathNames {
		names = append(names, name)
	}
	sort.Strings(names)

	paths = make([]string, len(r.pkgPathNames))
	for i, name := range names {
		paths[i] = r.pkgPathNames[name]
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

func getCallingPackageImportPath(skip int) string {
	stack := make([]uintptr, 1)
	num := runtime.Callers(skip+2, stack)
	if num != len(stack) {
		panic("insufficient call stack")
	}
	frame, _ := runtime.CallersFrames(stack).Next()
	name := frame.Func.Name()
	return name[:strings.LastIndexByte(name, '.')]
}
