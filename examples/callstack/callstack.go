package main

import (
	"github.com/domonda/golog"
	"github.com/domonda/golog/log"
)

func SubFunc() {
	log.Info("Logging in SubFunc").CallStack("stack").Log()
	log.Info("Logging in SubFunc but skip 1 frame").CallStackSkip("stack", 1).Log()
}

func TopLevelFunc() {
	log.Info("Logging in TopLevelFunc").CallStack("stack").Log()
	SubFunc()
}

func main() {
	log.Info("Logging in main").CallStack("stack").Log()
	TopLevelFunc()

	golog.TrimCallStackPathPrefix = ""
	log.Info("Logging in main with full path").CallStackSkip("stack", 0).Log()
}
