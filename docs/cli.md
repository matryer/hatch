# hatch CLI reference

All subcommands operate on the current working directory — `cd` into your
project first. Every command is safe to re-run.

## hatch init

Scaffolds a `.hatch/` source tree with the four primitive container
subdirectories (`_rules/`, `_skills/`, `_commands/`, `_agents/`).
Pass `-examples` to additionally write one working example of each
primitive. Pass `-path` to scaffold under a nested scope (see
[sources.md](sources.md)).

```
hatch init                    # empty .hatch/ scaffold
hatch init -examples          # + one sample rule, skill, command, and agent
hatch init -path backend      # → .hatch/backend/_rules/, _skills/, _commands/, _agents/
```

Existing files are left alone.

## hatch new

Creates a single new source file from a template. `kind` is one of `rule`,
`skill`, `command`, or `agent`. The title is slugged into a filesystem-safe
name; if you omit it, you'll be prompted on stdin. Pass `-path` to drop
the new file under a nested scope.

```
hatch new rule "Always write tests"           # → .hatch/_rules/always-write-tests.md
hatch new skill "review pr"                   # → .hatch/_skills/review-pr/SKILL.md
hatch new command commit                      # → .hatch/_commands/commit.md
hatch new agent "security auditor"            # → .hatch/_agents/security-auditor.md
hatch new rule -path backend "Database style" # → .hatch/backend/_rules/database-style.md
```

Skill, command, and agent templates include a `description:` frontmatter
field pre-filled with a `TODO` — fill it in before running `hatch gen`.

## hatch gen

Reads `.hatch/` and writes the native files each coding agent expects.
With no flags, every registered target is generated. Narrow the run with
`-targets`, a comma-separated list of target names:

```
hatch gen                         # every target
hatch gen -targets claude         # only Claude Code
hatch gen -targets claude,codex   # Claude Code and Codex
```

Output uses editor-jumpable `path:begin-end` notation for block-injected
files so you can click straight to the hatch block; file-owned outputs
print just the path:

```
$ hatch gen
wrote CLAUDE.md:1-3
wrote .claude/skills/review-pr/SKILL.md
wrote AGENTS.md:1-3
...
```

## hatch list

Dry-run: prints every path `hatch gen` would write, without touching the
filesystem. Useful for previewing changes before committing, or in CI to
assert the checked-in generated files match the current `.hatch/` source.

```
hatch list
hatch list -targets claude
```

## hatch check

CI gate. Re-derives what a fresh `hatch gen` would write and compares each
result to the file already on disk, without touching the filesystem. Exits
non-zero when any file is missing, stale, or has the wrong exec bit, and
prints one line per drifted path so the failure is actionable.

```
$ hatch check
out-of-date  CLAUDE.md  (stale)
out-of-date  .claude/skills/review-pr/SKILL.md  (missing)
hatch: 2 file(s) out of date; run `hatch gen` to update
```

On a clean tree it prints a one-line summary and exits 0:

```
$ hatch check
all 7 generated file(s) up to date
```

Accepts the same `-targets` and `-no-hatch-skill` flags as `gen`.

## hatch clean

Removes everything hatch generated. File-owned outputs are deleted;
the hatch-managed block is stripped from block-injected files (and the
file itself is deleted if it becomes empty). Your `.hatch/` source tree
is never touched.

## Auto-injected meta skill

Every `hatch gen` run automatically writes a `hatch` SKILL.md into each
target's native skill location, teaching the coding agent how `.hatch/`
is structured. Two ways to customise:

- **Override** — write your own `.hatch/_skills/hatch/SKILL.md`; the user
  version wins.
- **Opt out** — pass `-no-hatch-skill` to `gen`, `list`, and `clean`.

## hatch version and help

Print the version or the built-in usage summary. `-v`/`--version` and
`-h`/`--help` are accepted as aliases.
