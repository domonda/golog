package golog

var (
	_ Colorizer = noColorizer(0)        // make sure noColorizer implements Colorizer
	_ Colorizer = new(ConsoleColorizer) // make sure ConsoleColorizer implements Colorizer
)
