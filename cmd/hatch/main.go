// Command hatch generates rules, skills, commands, and agent definitions for
// multiple coding agents (Claude Code, Codex, Copilot, Cursor, OpenCode, Zed)
// from a single neutral source tree under `.hatch/`.
//
// Usage:
//
//	hatch gen   [-targets names]          write all target outputs
//	hatch list  [-targets names]          dry-run (print what would be written)
//	hatch check [-targets names]          verify generated files are up to date (for CI)
//	hatch clean [-targets names]          remove generated outputs
//	hatch init  [-examples] [-path p]    scaffold .hatch/ (optionally examples / nested scope)
//	hatch new <kind> [-path p] [title]   create a new source file
//	hatch version
//	hatch help
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/debug"

	"github.com/grafana/hatch/pkg/cli"
	"github.com/grafana/hatch/pkg/target"
	"github.com/grafana/hatch/pkg/target/claude"
	"github.com/grafana/hatch/pkg/target/codex"
	"github.com/grafana/hatch/pkg/target/copilot"
	"github.com/grafana/hatch/pkg/target/cursor"
	"github.com/grafana/hatch/pkg/target/opencode"
	"github.com/grafana/hatch/pkg/target/zed"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := run(ctx, os.Args, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "hatch: %s\n", err)
		os.Exit(1)
	}
}

// run is the testable entry point. It takes its dependencies explicitly so
// tests can drive the CLI with fake args and capture output.
func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	targets := target.NewSet(
		claude.New(),
		codex.New(),
		copilot.New(),
		cursor.New(),
		opencode.New(),
		zed.New(),
	)
	return cli.Run(ctx, version(), targets, args, stdin, stdout, stderr)
}

// version returns the hatch CLI version string, derived from Go build
// info. When installed via `go install github.com/grafana/hatch/cmd/hatch@vX.Y.Z`
// this is the tag (`v0.1.0`); when installed `@latest` off an untagged
// commit it's the pseudo-version (`v0.0.0-<date>-<sha>`). Local builds
// (`go build`, `go test`, `go run`) report `(devel)` in build info — for
// those we emit `dev` optionally suffixed with the short VCS commit so
// bug reports include enough to reproduce.
func version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	// Local/dev build — append short commit hash if the toolchain
	// recorded one. Tests running under `go test` typically won't.
	var rev, modified string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			modified = s.Value
		}
	}
	if rev == "" {
		return "dev"
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	if modified == "true" {
		return "dev+" + rev + "-dirty"
	}
	return "dev+" + rev
}
