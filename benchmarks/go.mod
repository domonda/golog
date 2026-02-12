module github.com/domonda/golog/benchmarks

go 1.24.0

// replaced with local golog package
require github.com/domonda/golog v0.0.0

replace github.com/domonda/golog => ../

require (
	github.com/rs/zerolog v1.34.0
	github.com/sirupsen/logrus v1.9.4
	go.uber.org/zap v1.27.1
)

require (
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/domonda/go-encjson v1.0.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/term v0.39.0 // indirect
)
