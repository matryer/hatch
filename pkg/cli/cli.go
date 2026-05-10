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
		return cmdHelp(version, targets, rest, stdout)
	default:
		return fmt.Errorf("unknown command %q (try `hatch help`)", cmd)
	}
}

// cmdHelp powers `hatch help [command]`. With no argument it prints the
// top-level overview; with one argument it prints a per-command help block
// (synopsis, flags, examples) so users can drill into a specific subcommand.
func cmdHelp(version string, targets *target.Set, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		printUsage(stdout, version, targets)
		return nil
	}
	if len(args) > 1 {
		return fmt.Errorf("hatch help: too many arguments (want at most one command name)")
	}
	cmd := args[0]
	text, ok := commandHelp[cmd]
	if !ok {
		return fmt.Errorf("hatch help: unknown command %q (try `hatch help`)", cmd)
	}
	fmt.Fprint(stdout, text)
	return nil
}

// commandHelp holds the per-subcommand help text shown by `hatch help <cmd>`.
// Each entry should be useful on its own: synopsis, one-paragraph description,
// flags (if any), and at least one runnable example.
var commandHelp = map[string]string{
	"gen": `hatch gen — write all target outputs

Reads .hatch/ and writes the native files each registered coding agent
expects. With no flags, every registered target is generated. Safe to
re-run; output is deterministic.

Usage:
  hatch gen [flags]

Flags:
  -targets names     comma-separated target names (default: all)
  -no-hatch-skill    skip the auto-injected hatch meta SKILL.md for this run

Examples:
  hatch gen
  hatch gen -targets claude
  hatch gen -targets claude,codex
`,
	"list": `hatch list — dry-run; print what gen would write

Prints every path hatch gen would write, without touching the filesystem.
Useful for previewing changes before committing, or in CI to assert the
checked-in generated files match the current .hatch/ source.

Usage:
  hatch list [flags]

Flags:
  -targets names     comma-separated target names (default: all)
  -no-hatch-skill    skip the auto-injected hatch meta SKILL.md for this run

Examples:
  hatch list
  hatch list -targets claude
`,
	"clean": `hatch clean — remove generated outputs

Re-derives what a fresh hatch gen would write from the current source tree,
then removes those files (for file-owned outputs) or strips just the
hatch-managed block (for block-injected files). Your .hatch/ source tree
is never touched.

Usage:
  hatch clean [flags]

Flags:
  -targets names     comma-separated target names (default: all)
  -no-hatch-skill    skip the auto-injected hatch meta SKILL.md for this run

Examples:
  hatch clean
  hatch clean -targets claude
`,
	"init": `hatch init — scaffold .hatch/

Creates the four primitive container subdirectories (_rules/, _skills/,
_commands/, _agents/) under .hatch/. Existing files are left alone.

Usage:
  hatch init [-examples] [-path p]

Flags:
  -examples          also write one example rule, skill, command, and agent
  -path p            scaffold under a nested scope path (e.g. backend or services/api)

Examples:
  hatch init
  hatch init -examples
  hatch init -path backend
`,
	"new": `hatch new — create a new source file from a template

Writes one new primitive source file under .hatch/. kind is one of rule,
skill, command, or agent. The title is slugged into a filesystem-safe
name; if omitted, you are prompted on stdin.

Usage:
  hatch new <kind> [-path p] [title]

Flags:
  -path p            create the new primitive under a nested scope path

Examples:
  hatch new rule "Always write tests"
  hatch new skill "review pr"
  hatch new command commit
  hatch new agent "security auditor"
  hatch new rule -path backend "Database style"
`,
	"version": `hatch version — print the hatch binary version

Usage:
  hatch version

Aliases: -v, --version
`,
	"help": `hatch help — print usage information

With no argument, prints the top-level overview of every subcommand. With
one argument, prints the detailed help block for that subcommand.

Usage:
  hatch help [command]

Examples:
  hatch help
  hatch help gen
  hatch help new

Aliases: -h, --help
`,
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

Run 'hatch help <command>' for detailed help on a specific subcommand.

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
