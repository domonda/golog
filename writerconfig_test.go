package golog

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_uniqueNonNilWriterConfigs(t *testing.T) {
	var (
		c0 = NopWriterConfig("c0")
		c1 = NopWriterConfig("c1")
		c2 = NopWriterConfig("c2")
	)
	tests := []struct {
		name          string
		w             []WriterConfig
		wantUnique    []WriterConfig
		wantSameSlice bool // expect returned slice to be the same as input
	}{
		// Nil and empty
		{name: "nil", w: nil, wantUnique: nil},
		{name: "empty", w: []WriterConfig{}, wantUnique: nil},
		{name: "nil only", w: []WriterConfig{nil}, wantUnique: nil},
		{name: "multiple nils", w: []WriterConfig{nil, nil, nil}, wantUnique: nil},

		// Already clean (should return same slice)
		{name: "single", w: []WriterConfig{c0}, wantUnique: []WriterConfig{c0}, wantSameSlice: true},
		{name: "two different", w: []WriterConfig{c0, c1}, wantUnique: []WriterConfig{c0, c1}, wantSameSlice: true},
		{name: "three different", w: []WriterConfig{c0, c1, c2}, wantUnique: []WriterConfig{c0, c1, c2}, wantSameSlice: true},

		// Duplicates (must allocate new slice)
		{name: "duplicate pair", w: []WriterConfig{c0, c0}, wantUnique: []WriterConfig{c0}},
		{name: "triplicate", w: []WriterConfig{c0, c0, c0}, wantUnique: []WriterConfig{c0}},
		{name: "duplicate preserves first", w: []WriterConfig{c0, c1, c0}, wantUnique: []WriterConfig{c0, c1}},
		{name: "all same except last", w: []WriterConfig{c0, c0, c1}, wantUnique: []WriterConfig{c0, c1}},

		// Nils mixed with values (must allocate new slice)
		{name: "nil at beginning", w: []WriterConfig{nil, c0, c1}, wantUnique: []WriterConfig{c0, c1}},
		{name: "nil in between", w: []WriterConfig{c0, nil, c1}, wantUnique: []WriterConfig{c0, c1}},
		{name: "nil at end", w: []WriterConfig{c0, c1, nil}, wantUnique: []WriterConfig{c0, c1}},
		{name: "nils and duplicates", w: []WriterConfig{nil, c0, nil, c0, c1, nil}, wantUnique: []WriterConfig{c0, c1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uniqueNonNilWriterConfigs(tt.w)
			require.Equal(t, tt.wantUnique, got)
			if tt.wantSameSlice && len(tt.w) > 0 {
				assert.True(t, &got[0] == &tt.w[0], "expected same underlying slice to be returned")
			}
		})
	}
}

func Test_mergeWriterConfigs(t *testing.T) {
	var (
		c0 = NopWriterConfig("c0")
		c1 = NopWriterConfig("c1")
		c2 = NopWriterConfig("c2")
		c3 = NopWriterConfig("c3")
	)
	tests := []struct {
		name        string
		a           []WriterConfig
		b           []WriterConfig
		want        []WriterConfig
		wantSameAsA bool // expect returned slice to be same as a
	}{
		// Both empty/nil
		{name: "nil nil", a: nil, b: nil, want: nil},
		{name: "empty empty", a: []WriterConfig{}, b: []WriterConfig{}, want: nil},
		{name: "nil empty", a: nil, b: []WriterConfig{}, want: nil},
		{name: "empty nil", a: []WriterConfig{}, b: nil, want: nil},

		// One side empty/nil
		{name: "a=nil b=single", a: nil, b: []WriterConfig{c0}, want: []WriterConfig{c0}},
		{name: "a=nil b=two", a: nil, b: []WriterConfig{c0, c1}, want: []WriterConfig{c0, c1}},
		{name: "a=single b=nil", a: []WriterConfig{c0}, b: nil, want: []WriterConfig{c0}, wantSameAsA: true},
		{name: "a=two b=nil", a: []WriterConfig{c0, c1}, b: nil, want: []WriterConfig{c0, c1}, wantSameAsA: true},
		{name: "a=single b=empty", a: []WriterConfig{c0}, b: []WriterConfig{}, want: []WriterConfig{c0}, wantSameAsA: true},

		// No overlap
		{name: "disjoint single", a: []WriterConfig{c0}, b: []WriterConfig{c1}, want: []WriterConfig{c0, c1}},
		{name: "disjoint multi", a: []WriterConfig{c0, c1}, b: []WriterConfig{c2, c3}, want: []WriterConfig{c0, c1, c2, c3}},
		{name: "disjoint asymmetric", a: []WriterConfig{c0}, b: []WriterConfig{c1, c2}, want: []WriterConfig{c0, c1, c2}},

		// Full overlap (b adds nothing new, a returned unchanged)
		{name: "same single", a: []WriterConfig{c0}, b: []WriterConfig{c0}, want: []WriterConfig{c0}, wantSameAsA: true},
		{name: "same two", a: []WriterConfig{c0, c1}, b: []WriterConfig{c0, c1}, want: []WriterConfig{c0, c1}, wantSameAsA: true},
		{name: "same reversed", a: []WriterConfig{c0, c1}, b: []WriterConfig{c1, c0}, want: []WriterConfig{c0, c1}, wantSameAsA: true},
		{name: "a superset", a: []WriterConfig{c0, c1, c2}, b: []WriterConfig{c1}, want: []WriterConfig{c0, c1, c2}, wantSameAsA: true},
		{name: "b subset of a", a: []WriterConfig{c0, c1, c2}, b: []WriterConfig{c2, c0}, want: []WriterConfig{c0, c1, c2}, wantSameAsA: true},

		// Partial overlap
		{name: "partial overlap", a: []WriterConfig{c0, c1}, b: []WriterConfig{c1, c2}, want: []WriterConfig{c0, c1, c2}},
		{name: "partial overlap reversed", a: []WriterConfig{c1, c2}, b: []WriterConfig{c0, c1}, want: []WriterConfig{c1, c2, c0}},

		// Nils in a
		{name: "nil in a no overlap", a: []WriterConfig{c0, nil}, b: []WriterConfig{c1}, want: []WriterConfig{c0, c1}},
		{name: "nil in a full overlap", a: []WriterConfig{nil, c0}, b: []WriterConfig{c0}, want: []WriterConfig{c0}},
		{name: "all nil a", a: []WriterConfig{nil, nil}, b: []WriterConfig{c0}, want: []WriterConfig{c0}},

		// Nils in b
		{name: "nil in b no overlap", a: []WriterConfig{c0}, b: []WriterConfig{nil, c1}, want: []WriterConfig{c0, c1}},
		{name: "nil in b full overlap", a: []WriterConfig{c0}, b: []WriterConfig{nil, c0}, want: []WriterConfig{c0}, wantSameAsA: true},
		{name: "all nil b", a: []WriterConfig{c0}, b: []WriterConfig{nil, nil}, want: []WriterConfig{c0}, wantSameAsA: true},

		// Nils in both
		{name: "nils both sides", a: []WriterConfig{nil, c0}, b: []WriterConfig{nil, c1}, want: []WriterConfig{c0, c1}},
		{name: "all nils both", a: []WriterConfig{nil}, b: []WriterConfig{nil}, want: nil},

		// Duplicates within a
		{name: "dup in a new in b", a: []WriterConfig{c0, c0}, b: []WriterConfig{c1}, want: []WriterConfig{c0, c1}},
		{name: "dup in a overlap b", a: []WriterConfig{c0, c0}, b: []WriterConfig{c0}, want: []WriterConfig{c0}},

		// Duplicates within b
		{name: "dup in b new", a: []WriterConfig{c0}, b: []WriterConfig{c1, c1}, want: []WriterConfig{c0, c1}},
		{name: "dup in b overlap", a: []WriterConfig{c0}, b: []WriterConfig{c0, c0}, want: []WriterConfig{c0}, wantSameAsA: true},

		// Duplicates in both
		{name: "dups everywhere", a: []WriterConfig{c0, c0, c1}, b: []WriterConfig{c1, c2, c2}, want: []WriterConfig{c0, c1, c2}},

		// Order preservation
		{name: "order a then b", a: []WriterConfig{c2, c0}, b: []WriterConfig{c3, c1}, want: []WriterConfig{c2, c0, c3, c1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeWriterConfigs(tt.a, tt.b)
			require.Equal(t, tt.want, got, "mergeWriterConfigs result")

			if tt.wantSameAsA && len(tt.a) > 0 {
				assert.True(t, &got[0] == &tt.a[0], "expected same underlying slice as a to be returned")
			}

			// Cross-validate against reference: uniqueNonNilWriterConfigs(append(a, b...))
			ref := uniqueNonNilWriterConfigs(append(slices.Clone(tt.a), tt.b...))
			require.Equal(t, ref, got, "mergeWriterConfigs should match uniqueNonNilWriterConfigs(append(a,b...))")
		})
	}
}

// TestTerminalDetection tests that the logger automatically switches between
// Text and JSON format based on whether it's running in a terminal or not.
// This test uses a Docker container to simulate both scenarios.
func TestTerminalDetection(t *testing.T) {
	// Check if Docker is available
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping terminal detection test")
	}

	const (
		imageName      = "golog-terminaldetection-test"
		dockerfilePath = "./tests/terminaldetection"
	)

	// Build the Docker image
	t.Log("Building Docker image...")
	buildCmd := exec.Command("docker", "build", "-t", imageName, "-f", dockerfilePath+"/Dockerfile", ".")
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build Docker image: %v\nOutput: %s", err, buildOutput)
	}

	// Clean up the image after the test
	defer func() {
		t.Log("Cleaning up Docker image...")
		cleanupCmd := exec.Command("docker", "rmi", "-f", imageName)
		_ = cleanupCmd.Run() // Ignore errors during cleanup
	}()

	t.Run("with terminal produces text format", func(t *testing.T) {
		// Run with -t flag to allocate a pseudo-TTY
		cmd := exec.Command("docker", "run", "--rm", "-t", imageName)
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stdout

		err := cmd.Run()
		require.NoError(t, err, "Failed to run Docker container with terminal")

		output := stdout.String()
		t.Logf("Output with terminal:\n%s", output)

		// Verify it's text format (not JSON)
		// Text format should contain readable timestamps and log levels
		// and should NOT be valid JSON
		lines := strings.Split(strings.TrimSpace(output), "\n")
		require.NotEmpty(t, lines, "Expected output")

		for _, line := range lines {
			// Each line should NOT be valid JSON (text format)
			var jsonData map[string]any
			err := json.Unmarshal([]byte(line), &jsonData)
			assert.Error(t, err, "Expected text format, but line is valid JSON: %s", line)

			// Text format should contain recognizable patterns
			// Look for timestamp-like patterns and log level indicators
			hasTimestamp := strings.Contains(line, ":") &&
				(strings.Contains(line, "INFO") ||
					strings.Contains(line, "WARN") ||
					strings.Contains(line, "ERROR") ||
					strings.Contains(line, "DEBUG"))
			assert.True(t, hasTimestamp, "Expected text format with timestamp and level, got: %s", line)
		}
	})

	t.Run("without terminal produces JSON format", func(t *testing.T) {
		// Run without -t flag (no pseudo-TTY)
		cmd := exec.Command("docker", "run", "--rm", imageName)
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stdout

		err := cmd.Run()
		require.NoError(t, err, "Failed to run Docker container without terminal")

		output := stdout.String()
		t.Logf("Output without terminal:\n%s", output)

		// Verify it's JSON format
		lines := strings.Split(strings.TrimSpace(output), "\n")
		require.NotEmpty(t, lines, "Expected output")

		// Expected log levels in order
		expectedLevels := []string{"INFO", "WARN", "ERROR", "DEBUG"}
		levelIndex := 0

		for i, line := range lines {
			if line == "" {
				continue
			}

			// Each line should be valid JSON
			var jsonData map[string]any
			err := json.Unmarshal([]byte(line), &jsonData)
			require.NoError(t, err, "Expected JSON format on line %d, got: %s", i+1, line)

			// Verify JSON structure contains expected fields
			assert.Contains(t, jsonData, "time", "JSON should contain 'time' field")
			assert.Contains(t, jsonData, "level", "JSON should contain 'level' field")
			assert.Contains(t, jsonData, "message", "JSON should contain 'message' field")

			// Verify log level matches expected order
			if levelIndex < len(expectedLevels) {
				level, ok := jsonData["level"].(string)
				require.True(t, ok, "Level should be a string")
				assert.Equal(t, expectedLevels[levelIndex], level,
					"Expected level %s on line %d", expectedLevels[levelIndex], i+1)
				levelIndex++
			}
		}

		assert.Equal(t, len(expectedLevels), levelIndex,
			"Expected %d log entries, got %d", len(expectedLevels), levelIndex)
	})
}

// isDockerAvailable checks if Docker is available on the system
func isDockerAvailable() bool {
	cmd := exec.Command("docker", "version")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run() == nil
}
