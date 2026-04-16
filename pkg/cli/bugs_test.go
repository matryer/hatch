package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/grafana/hatch/pkg/cli"
	"github.com/matryer/is"
)

// Bug 5: `hatch init` writes its "created X" log lines by iterating a Go
// map, which has randomized iteration order. Running init across fresh dirs
// produces non-deterministic stdout — breaks reproducibility and makes
// golden-file tests flaky. Medium severity.
func TestBug_InitOutputIsDeterministic(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	run := func() []string {
		dir := t.TempDir()
		t.Chdir(dir)
		var stdout, stderr bytes.Buffer
		err := cli.Run(ctx, "test", allTargets(), []string{"hatch", "init"}, strings.NewReader(""), &stdout, &stderr)
		is.NoErr(err)
		return extractCreatedLines(stdout.String())
	}

	first := run()
	// Map iteration is randomized per-range-call, so even within a single
	// process successive runs can differ. Run several times to give the bug
	// a chance to surface.
	for i := 0; i < 20; i++ {
		got := run()
		is.Equal(got, first)
	}
}

// extractCreatedLines returns the path portion of each "created X" line.
func extractCreatedLines(stdout string) []string {
	var out []string
	for _, line := range strings.Split(stdout, "\n") {
		if !strings.HasPrefix(line, "created ") {
			continue
		}
		out = append(out, strings.TrimPrefix(line, "created "))
	}
	return out
}
