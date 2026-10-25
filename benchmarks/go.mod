module github.com/domonda/golog/benchmarks

go 1.26.0

// replaced with local golog package
require github.com/domonda/golog v0.0.0-00010101000000-000000000000

replace github.com/domonda/golog => ../

require (
	github.com/rs/zerolog v1.35.1
	github.com/sirupsen/logrus v1.10.2
	go.uber.org/zap v1.28.0
)

require (
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/domonda/go-encjson v1.0.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.1 // indirect
	github.com/mattn/go-colorable v0.1.16 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/term v0.47.0 // indirect
)
