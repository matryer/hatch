package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafana/hatch/pkg/cli"
	"github.com/grafana/hatch/pkg/target"
	"github.com/grafana/hatch/pkg/target/claude"
	"github.com/grafana/hatch/pkg/target/codex"
	"github.com/grafana/hatch/pkg/target/copilot"
	"github.com/grafana/hatch/pkg/target/cursor"
	"github.com/grafana/hatch/pkg/target/opencode"
	"github.com/matryer/is"
)

// allTargets builds the same target set the real binary does.
func allTargets() *target.Set {
	return target.NewSet(
		claude.New(),
		codex.New(),
		copilot.New(),
		cursor.New(),
		opencode.New(),
	)
}

// invoke runs the CLI with the given argv and returns stdout, stderr.
func invoke(t *testing.T, args ...string) (string, string, error) {
	return invokeWithStdin(t, "", args...)
}

// invokeWithStdin is invoke's stdin-aware variant — used by tests that
// drive an interactive subcommand (currently only `hatch new`).
func invokeWithStdin(t *testing.T, stdin string, args ...string) (string, string, error) {
	t.Helper()
	return invokeWithCtx(t, context.Background(), stdin, args...)
}

// invokeWithCtx drives the CLI with a caller-supplied context — used by
// tests that need to assert cancellation behavior.
func invokeWithCtx(t *testing.T, ctx context.Context, stdin string, args ...string) (string, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	err := cli.Run(ctx, "test", allTargets(), append([]string{"hatch"}, args...), strings.NewReader(stdin), &stdout, &stderr)
	return stdout.String(), stderr.String(), err
}

// scaffoldSource writes a minimal .hatch/ tree with one rule, one skill,
// one command, and one agent into the given root.
func scaffoldSource(t *testing.T, root string) {
	t.Helper()
	files := map[string]string{
		".hatch/_rules/style.md": "Be concise.\n",
		".hatch/_skills/review-pr/SKILL.md": `---
description: Review a PR.
---
body
`,
		".hatch/_commands/commit.md": `---
description: Commit changes.
---
commit body
`,
		".hatch/_agents/security.md": `---
description: Security review.
---
security body
`,
	}
	for path, body := range files {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRun_Help(t *testing.T) {
	is := is.New(t)
	stdout, _, err := invoke(t, "help")
	is.NoErr(err)
	is.True(strings.Contains(stdout, "hatch test"))
	is.True(strings.Contains(stdout, "generate"))
	is.True(strings.Contains(stdout, "claude"))
}

func TestRun_NoArgsPrintsUsage(t *testing.T) {
	is := is.New(t)
	stdout, _, err := invoke(t)
	is.NoErr(err)
	is.True(strings.Contains(stdout, "Usage:"))
}

func TestRun_Version(t *testing.T) {
	is := is.New(t)
	stdout, _, err := invoke(t, "version")
	is.NoErr(err)
	is.Equal(strings.TrimSpace(stdout), "test")
}

func TestRun_UnknownCommand(t *testing.T) {
	is := is.New(t)
	_, _, err := invoke(t, "nope")
	is.True(err != nil)
}
