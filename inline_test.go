package dringbuf_test

import (
	"os/exec"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRingBuffer_InlineCost guards the inline cost of the RingBuffer hot path.
// Callers wrap these methods in their own small accessors, and the wrappers
// only inline while the methods leave them headroom under the compiler's
// budget of 80. Losing that once doubled some gotrade detector timings while
// every benchmark in this package stayed flat.
func TestRingBuffer_InlineCost(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the compiler")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not found")
	}

	out, err := exec.Command(goBin, "build", "-gcflags=-m=2", ".").CombinedOutput()
	require.NoError(t, err, string(out))

	re := regexp.MustCompile(`can inline \(\*RingBuffer\[go\.shape\.int\]\)\.(\w+) with cost (\d+)`)
	costs := map[string]int{}
	for _, m := range re.FindAllStringSubmatch(string(out), -1) {
		costs[m[1]], _ = strconv.Atoi(m[2])
	}

	limits := map[string]int{"Get": 30, "At": 30, "Last": 50, "Append": 70}
	for name, limit := range limits {
		cost, ok := costs[name]
		if assert.True(t, ok, "%s is not inlinable", name) {
			assert.LessOrEqual(t, cost, limit, "%s inline cost", name)
		}
	}
}
