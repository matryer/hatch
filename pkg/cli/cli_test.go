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

func TestRun_HelpTopLevel_MentionsPerCommandHelp(t *testing.T) {
	// The bare `hatch help` overview should tell users how to drill into
	// a specific subcommand, otherwise the per-command help is undiscoverable.
	is := is.New(t)
	stdout, _, err := invoke(t, "help")
	is.NoErr(err)
	is.True(strings.Contains(stdout, "hatch help <command>"))
}

func TestRun_HelpForEachCommand_PrintsDetailedHelp(t *testing.T) {
	// Per-command help must exist for every dispatchable subcommand. The
	// output must contain the command name in its synopsis AND must NOT be
	// the top-level overview (which lists every command and the registered
	// targets) — otherwise we're silently falling through to the overview.
	for _, cmd := range []string{"gen", "list", "check", "clean", "init", "new", "version", "help"} {
		t.Run(cmd, func(t *testing.T) {
			is := is.New(t)
			stdout, _, err := invoke(t, "help", cmd)
			is.NoErr(err)
			is.True(strings.Contains(stdout, "hatch "+cmd))    // synopsis must mention the command
			is.True(!strings.Contains(stdout, "Registered targets")) // must not be the top-level overview
		})
	}
}

func TestRun_HelpForGen_IncludesFlagsAndExamples(t *testing.T) {
	// `hatch help gen` should be useful: list flags and at least one
	// example invocation. Pin the most important bits so we don't ship a
	// per-command help block that's empty fluff.
	is := is.New(t)
	stdout, _, err := invoke(t, "help", "gen")
	is.NoErr(err)
	is.True(strings.Contains(stdout, "-targets"))
	is.True(strings.Contains(stdout, "-no-hatch-skill"))
	is.True(strings.Contains(stdout, "hatch gen -targets"))
}

func TestRun_HelpUnknownCommand(t *testing.T) {
	is := is.New(t)
	_, _, err := invoke(t, "help", "nope")
	is.True(err != nil)
	is.True(strings.Contains(err.Error(), "nope"))
}

func TestRun_HelpTooManyArgs(t *testing.T) {
	is := is.New(t)
	_, _, err := invoke(t, "help", "gen", "extra")
	is.True(err != nil)
}
