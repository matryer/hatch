// Package cli holds the subcommand dispatch and per-command handlers for
// the hatch binary. It's public so external tools can embed the same CLI —
// pass your own target.Set into Run to extend the set of supported agents.
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/grafana/hatch/pkg/target"
)

// Run is the testable CLI entry point. It dispatches on args[1] and delegates
// to a per-subcommand handler. The caller supplies the target set explicitly;
// there is no global registry.
//
// `stdin` is used only by interactive subcommands (currently `hatch new` when
// no title argument is supplied). Callers that never invoke those may pass a
// nil or empty reader.
func Run(ctx context.Context, version string, targets *target.Set, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) < 2 {
		printUsage(stdout, version, targets)
		return nil
	}
	cmd, rest := args[1], args[2:]
	switch cmd {
	case "gen":
		return cmdGen(ctx, version, targets, rest, stdout, stderr)
	case "list":
		return cmdList(ctx, version, targets, rest, stdout, stderr)
	case "clean":
		return cmdClean(ctx, version, targets, rest, stdout, stderr)
	case "init":
		return cmdInit(ctx, rest, stdout, stderr)
	case "new":
		return cmdNew(ctx, rest, stdin, stdout, stderr)
	case "version", "--version", "-v":
		fmt.Fprintln(stdout, version)
		return nil
	case "help", "--help", "-h":
		printUsage(stdout, version, targets)
		return nil
	default:
		return fmt.Errorf("unknown command %q (try `hatch help`)", cmd)
	}
}

func printUsage(w io.Writer, version string, targets *target.Set) {
	fmt.Fprintf(w, `hatch %s

Generate rules, skills, commands, and sub-agent definitions for multiple
coding agents from a single source at .hatch/.

Usage:
  hatch gen   [flags]                  write all target outputs
  hatch list  [flags]                  dry-run; print what would be written
  hatch clean [flags]                  remove generated outputs
  hatch init  [-examples] [-path p]    scaffold .hatch/ (optionally with example files or under a nested scope)
  hatch new <kind> [-path p] [title]   create a new source file
  hatch version                        print version and exit
  hatch help                           this message

Flags (gen/list/clean):
  -targets names     comma-separated target names (default: all)
  -no-hatch-skill    skip the auto-injected hatch meta SKILL.md for this run

Registered targets: %s
`, version, strings.Join(targets.Names(), ", "))
}

// commonFlagSet bundles the flags shared by generate/list/clean. All
// subcommands operate on the current working directory.
type commonFlagSet struct {
	fs           *flag.FlagSet
	targetsList  *string
	noHatchSkill *bool
}

func commonFlags(name string, stderr io.Writer) *commonFlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return &commonFlagSet{
		fs:           fs,
		targetsList:  fs.String("targets", "", "comma-separated target names (default: all)"),
		noHatchSkill: fs.Bool("no-hatch-skill", false, "skip the auto-injected hatch meta SKILL.md for this run"),
	}
}

// ensureNoPositional returns an error if any positional arguments remain
// after fs.Parse. The subcommands that call it don't accept any positional
// arguments — target selection is `-targets` only — so a stray word like
// `hatch gen claude` must fail loudly rather than be silently ignored and
// run against every target.
func ensureNoPositional(fs *flag.FlagSet, cmd string) error {
	if fs.NArg() == 0 {
		return nil
	}
	return fmt.Errorf("hatch %s: unexpected argument %q (use -targets to narrow targets)", cmd, fs.Arg(0))
}

// selectTargets resolves a comma-separated target-name list against the
// available set. An empty value means "all".
func selectTargets(available *target.Set, list string) (*target.Set, error) {
	list = strings.TrimSpace(list)
	if list == "" {
		return available, nil
	}
	var names []string
	for _, n := range strings.Split(list, ",") {
		n = strings.TrimSpace(n)
		if n != "" {
			names = append(names, n)
		}
	}
	return available.Select(names)
}
