package golog

import (
	"runtime"
	"strings"
	"sync"
)

type Registry struct {
	configs map[string]*DerivedConfig
	mutex   sync.Mutex
}

func (r *Registry) AddPackageConfig(config *DerivedConfig) (pkgImportPath string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.configs == nil {
		r.configs = make(map[string]*DerivedConfig)
	}

	pkgImportPath = getCallingPackageImportPath(2)

	if _, exists := r.configs[pkgImportPath]; exists {
		// Panicing because AddPackageConfig is one time global
		// setup before any other error handlers
		panic("package config already added: " + pkgImportPath)
	}

	r.configs[pkgImportPath] = config

	return pkgImportPath
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
