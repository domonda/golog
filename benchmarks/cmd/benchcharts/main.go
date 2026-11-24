// Command benchcharts draws the benchmark charts of the README
// as SVG files for a light and a dark theme from the output of
// the comparative benchmarks, using the median of all runs:
//
//	go test -run '^$' -bench=. -benchmem -benchtime=2s -count=6 > results.txt
//	go run ./cmd/benchcharts results.txt charts
package main

import (
	"bufio"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// libs in the fixed order of the allocation grid
var libs = []string{"golog", "zerolog", "zap", "slog", "logrus"}

type scenario struct {
	bench string // benchmark name without the Benchmark prefix
	label string
}

// speedScenarios are the panels of the speed chart
var speedScenarios = []scenario{
	{"WithFields", "JSON, 4 fields"},
	{"WithManyFields", "JSON, 10 fields"},
	{"TextOutput", "Text, no colors"},
	{"Disabled", "Level disabled"},
}

// allocScenarios are the columns of the allocation grid
var allocScenarios = []scenario{
	{"SimpleMessage", "Message only"},
	{"WithFields", "4 fields"},
	{"WithManyFields", "10 fields"},
	{"WithAccumulatedContext", "Sub-logger"},
	{"ComplexFields", "Error + time"},
	{"TextOutput", "Text"},
	{"Disabled", "Disabled"},
}

type result struct {
	nsPerOp     float64
	bytesPerOp  int
	allocsPerOp int
}

type theme struct {
	name      string
	surface   string
	border    string
	primary   string
	secondary string
	muted     string
	accent    string // golog
	context   string // the other libraries
	ramp      [4]string
	rampInk   [4]string
}

var themes = []theme{
	{
		name:      "light",
		surface:   "#ffffff",
		border:    "#d1d9e0",
		primary:   "#0b0b0b",
		secondary: "#52514e",
		muted:     "#6e6d68",
		accent:    "#2a78d6",
		context:   "#b4b2a9",
		ramp:      [4]string{"#f09c74", "#e0703e", "#b8501f", "#863812"},
		rampInk:   [4]string{"#0b0b0b", "#0b0b0b", "#ffffff", "#ffffff"},
	},
	{
		name:      "dark",
		surface:   "#0d1117",
		border:    "#3d444d",
		primary:   "#f0efec",
		secondary: "#c3c2b7",
		muted:     "#9198a1",
		accent:    "#3987e5",
		context:   "#5c5b56",
		ramp:      [4]string{"#7d3719", "#a8481d", "#d6672f", "#f4a27b"},
		rampInk:   [4]string{"#ffffff", "#ffffff", "#0b0b0b", "#0b0b0b"},
	},
}

const font = `system-ui, -apple-system, 'Segoe UI', Helvetica, Arial, sans-serif`

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: benchcharts <results.txt> <output dir>")
		os.Exit(2)
	}
	results, header, err := readResults(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, t := range themes {
		files := map[string]string{
			"speed-" + t.name + ".svg":  speedChart(results, header, t),
			"allocs-" + t.name + ".svg": allocChart(results, header, t),
		}
		for name, svg := range files {
			err = os.WriteFile(filepath.Join(os.Args[2], name), []byte(svg), 0o644) // #nosec G306 G703 -- public chart written to the output dir of the command line
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	}
}

var lineRegex = regexp.MustCompile(`^Benchmark(\w+)/(\w+)-\d+\s+\d+\s+([\d.]+) ns/op\s+(\d+) B/op\s+(\d+) allocs/op`)

// readResults returns the median result per benchmark and library
// and a header line naming the CPU, Go version and number of runs.
func readResults(path string) (map[string]result, string, error) {
	file, err := os.Open(path) // #nosec G304 G703 -- results file of the command line
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	runs := map[string][]result{}
	cpu, goVersion := "", ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if v, ok := strings.CutPrefix(line, "cpu: "); ok {
			cpu = v
		}
		if v, ok := strings.CutPrefix(line, "go version "); ok {
			goVersion = strings.Fields(v)[0]
		}
		m := lineRegex.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		ns, _ := strconv.ParseFloat(m[3], 64)
		b, _ := strconv.Atoi(m[4])
		allocs, _ := strconv.Atoi(m[5])
		key := m[1] + "/" + m[2]
		runs[key] = append(runs[key], result{ns, b, allocs})
	}
	if err = scanner.Err(); err != nil {
		return nil, "", err
	}
	if len(runs) == 0 {
		return nil, "", fmt.Errorf("no benchmark results in %s", path)
	}
	medians := map[string]result{}
	numRuns := 0
	for key, rs := range runs {
		ns := make([]float64, len(rs))
		for i, r := range rs {
			ns[i] = r.nsPerOp
		}
		slices.Sort(ns)
		median := ns[len(ns)/2]
		if len(ns)%2 == 0 {
			median = (ns[len(ns)/2-1] + ns[len(ns)/2]) / 2
		}
		medians[key] = result{median, rs[0].bytesPerOp, rs[0].allocsPerOp}
		numRuns = max(numRuns, len(rs))
	}
	header := fmt.Sprintf("%s · %s · median of %d runs", cpu, strings.TrimPrefix(goVersion, "go"), numRuns)
	if goVersion != "" {
		header = fmt.Sprintf("%s · Go %s · median of %d runs", cpu, strings.TrimPrefix(goVersion, "go"), numRuns)
	}
	return medians, header, nil
}

func formatNs(ns float64) string {
	if ns < 10 {
		return strconv.FormatFloat(ns, 'f', 1, 64) + " ns"
	}
	return strconv.FormatFloat(ns, 'f', 0, 64) + " ns"
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return strconv.Itoa(n) + " " + word + "s"
}

// barPath returns a horizontal bar anchored at x with a rounded data end.
func barPath(x, y, w, h float64) string {
	r := min(4, w/2, h/2)
	return fmt.Sprintf("M%.1f,%.1fh%.1fa%.1f,%.1f 0 0 1 %.1f,%.1fv%.1fa%.1f,%.1f 0 0 1 %.1f,%.1fh%.1fz",
		x, y, w-r, r, r, r, r, h-2*r, r, r, -r, r, -(w - r))
}

func svgStart(b *strings.Builder, t theme, width, height int, title, desc string) {
	fmt.Fprintf(b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-labelledby="title desc" font-family="%s">`+"\n",
		width, height, width, height, font)
	fmt.Fprintf(b, "<title id=\"title\">%s</title>\n<desc id=\"desc\">%s</desc>\n", html.EscapeString(title), html.EscapeString(desc))
	fmt.Fprintf(b, `<rect x="0.5" y="0.5" width="%d" height="%d" rx="8" fill="%s" stroke="%s"/>`+"\n", width-1, height-1, t.surface, t.border)
}

func text(b *strings.Builder, x, y float64, size int, weight int, fill, anchor, s string) {
	fmt.Fprintf(b, `<text x="%.1f" y="%.1f" font-size="%d" font-weight="%d" fill="%s" text-anchor="%s">%s</text>`+"\n",
		x, y, size, weight, fill, anchor, html.EscapeString(s))
}

func speedChart(results map[string]result, header string, t theme) string {
	const (
		width     = 880
		margin    = 24.0
		panelGap  = 40.0
		rowHeight = 26.0
		barHeight = 14.0
		nameWidth = 64.0
		labelRoom = 150.0
		top       = 84.0
		panelHead = 30.0
		panelGapY = 26.0
	)
	panelWidth := (width - 2*margin - panelGap) / 2
	panelHeight := panelHead + float64(len(libs))*rowHeight
	height := int(top + 2*panelHeight + panelGapY + 44)

	var desc strings.Builder
	for _, s := range speedScenarios {
		fmt.Fprintf(&desc, "%s: ", s.label)
		for i, lib := range libs {
			r := results[s.bench+"/"+lib]
			if i > 0 {
				desc.WriteString(", ")
			}
			fmt.Fprintf(&desc, "%s %s with %s", lib, formatNs(r.nsPerOp), plural(r.allocsPerOp, "allocation"))
		}
		desc.WriteString(". ")
	}

	var b strings.Builder
	svgStart(&b, t, width, height, "Nanoseconds per log message, shorter is faster", desc.String())
	text(&b, margin, 36, 20, 600, t.primary, "start", "Nanoseconds per log message")
	text(&b, margin, 60, 13, 400, t.secondary, "start", "Shorter is faster. "+header+".")

	for i, s := range speedScenarios {
		px := margin + float64(i%2)*(panelWidth+panelGap)
		py := top + float64(i/2)*(panelHeight+panelGapY)
		text(&b, px, py+16, 15, 600, t.primary, "start", s.label)

		ranked := slices.Clone(libs)
		slices.SortFunc(ranked, func(a, c string) int {
			ra, rc := results[s.bench+"/"+a].nsPerOp, results[s.bench+"/"+c].nsPerOp
			switch {
			case ra < rc:
				return -1
			case ra > rc:
				return 1
			}
			return 0
		})
		maxNs := results[s.bench+"/"+ranked[len(ranked)-1]].nsPerOp
		barMax := panelWidth - nameWidth - labelRoom
		// Baseline of the bars
		fmt.Fprintf(&b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="1"/>`+"\n",
			px+nameWidth, py+panelHead-4, px+nameWidth, py+panelHead+float64(len(libs))*rowHeight-6, t.border)
		for j, lib := range ranked {
			r := results[s.bench+"/"+lib]
			y := py + panelHead + float64(j)*rowHeight
			fill, ink, weight := t.context, t.secondary, 400
			if lib == "golog" {
				fill, ink, weight = t.accent, t.primary, 700
			}
			text(&b, px+nameWidth-10, y+barHeight-2, 13, weight, ink, "end", lib)
			w := max(2, barMax*r.nsPerOp/maxNs)
			fmt.Fprintf(&b, `<path d="%s" fill="%s"/>`+"\n", barPath(px+nameWidth, y, w, barHeight), fill)
			label := formatNs(r.nsPerOp) + " · " + plural(r.allocsPerOp, "alloc")
			text(&b, px+nameWidth+w+8, y+barHeight-2, 12, weight, ink, "start", label)
		}
	}
	text(&b, margin, float64(height)-18, 12, 400, t.muted, "start",
		"github.com/domonda/golog/benchmarks · go test -bench=. -benchmem -benchtime=2s -count=6")
	b.WriteString("</svg>\n")
	return b.String()
}

// allocBin returns the ramp step for a number of allocations, or -1 for none.
func allocBin(n int) int {
	switch {
	case n == 0:
		return -1
	case n == 1:
		return 0
	case n < 10:
		return 1
	case n < 30:
		return 2
	default:
		return 3
	}
}

func allocChart(results map[string]result, header string, t theme) string {
	const (
		width     = 880
		margin    = 24.0
		nameWidth = 84.0
		top       = 84.0
		headRow   = 30.0
		cellH     = 34.0
		gap       = 4.0
	)
	cellW := (width - 2*margin - nameWidth - gap*float64(len(allocScenarios)-1)) / float64(len(allocScenarios))
	gridBottom := top + headRow + float64(len(libs))*(cellH+gap)
	height := int(gridBottom + 76)

	var desc strings.Builder
	for _, lib := range libs {
		fmt.Fprintf(&desc, "%s: ", lib)
		for i, s := range allocScenarios {
			if i > 0 {
				desc.WriteString(", ")
			}
			fmt.Fprintf(&desc, "%s %d", s.label, results[s.bench+"/"+lib].allocsPerOp)
		}
		desc.WriteString(". ")
	}

	var b strings.Builder
	svgStart(&b, t, width, height, "Heap allocations per log message, fewer is better", desc.String())
	text(&b, margin, 36, 20, 600, t.primary, "start", "Heap allocations per log message")
	text(&b, margin, 60, 13, 400, t.secondary, "start", "Fewer is better. "+header+".")

	for i, s := range allocScenarios {
		x := margin + nameWidth + float64(i)*(cellW+gap) + cellW/2
		text(&b, x, top+16, 12, 600, t.secondary, "middle", s.label)
	}
	for j, lib := range libs {
		y := top + headRow + float64(j)*(cellH+gap)
		ink, weight := t.secondary, 400
		if lib == "golog" {
			ink, weight = t.primary, 700
		}
		text(&b, margin+nameWidth-12, y+cellH/2+5, 14, weight, ink, "end", lib)
		for i, s := range allocScenarios {
			n := results[s.bench+"/"+lib].allocsPerOp
			x := margin + nameWidth + float64(i)*(cellW+gap)
			bin := allocBin(n)
			cellInk := t.muted
			if bin >= 0 {
				fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="4" fill="%s"/>`+"\n", x, y, cellW, cellH, t.ramp[bin])
				cellInk = t.rampInk[bin]
			} else {
				fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="4" fill="none" stroke="%s"/>`+"\n", x+0.5, y+0.5, cellW-1, cellH-1, t.border)
			}
			text(&b, x+cellW/2, y+cellH/2+5, 14, weight, cellInk, "middle", strconv.Itoa(n))
		}
	}

	// Legend of the bins
	legend := []string{"none", "1", "2–9", "10–29", "30+"}
	x := margin + nameWidth
	ly := gridBottom + 12
	for i, l := range legend {
		if i == 0 {
			fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="14" height="14" rx="3" fill="none" stroke="%s"/>`+"\n", x+0.5, ly+0.5, t.border)
		} else {
			fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="14" height="14" rx="3" fill="%s"/>`+"\n", x, ly, t.ramp[i-1])
		}
		text(&b, x+20, ly+12, 12, 400, t.secondary, "start", l)
		x += 20 + float64(len(l))*7 + 28
	}
	text(&b, margin, float64(height)-18, 12, 400, t.muted, "start",
		"github.com/domonda/golog/benchmarks · go test -bench=. -benchmem -benchtime=2s -count=6")
	b.WriteString("</svg>\n")
	return b.String()
}
