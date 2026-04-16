# 🥚 hatch

Write rules, skills, commands, and sub-agent definitions **once**, generate
the native files each coding agent expects.

Hatch reads a single source tree under `.hatch/` and produces the specific
files Claude Code, OpenAI Codex CLI, GitHub Copilot, Cursor, and OpenCode
each read to customise their behaviour.

## Install

```
go install github.com/grafana/hatch/cmd/hatch@latest
```

Or with mise inside the repo:

```
mise run install
```

## Quick start

```
$ cd my-project
$ hatch init -examples
$ hatch gen
wrote CLAUDE.md:1-3
wrote .claude/skills/review-pr/SKILL.md
wrote AGENTS.md:1-3
...
```

Edit files under `.hatch/_rules/`, `.hatch/_skills/`, `.hatch/_commands/`,
or `.hatch/_agents/`, then re-run `hatch gen`. Commit the `.hatch/` source
*and* the generated files together.

## Feature support

✓ means the agent has a native primitive; ⚠ means hatch emulates it.
See [docs/features.md](docs/features.md) for how each emulation works,
its output paths, and trade-offs.

| Feature                   | Claude Code | Codex | Copilot | Cursor | OpenCode |
| ------------------------- | :---------: | :---: | :-----: | :----: | :------: |
| Rules (always-on)         |      ✓      |   ✓   |    ✓    |   ✓    |    ✓     |
| Rules (scoped `applyTo`)  |      ⚠      |   ⚠   |    ✓    |   ✓    |    ⚠     |
| Skills                    |      ✓      |   ✓   |    ⚠    |   ⚠    |    ✓     |
| Slash commands            |      ✓      |   ⚠   |    ✓    |   ⚠    |    ✓     |
| Sub-agents                |      ✓      |   ⚠   |    ✓    |   ⚠    |    ✓     |
| Nested scopes (monorepo)  |      ✓      |   ✓   |    ⚠    |   ⚠    |    ✓     |

## CLI

| Command                                                                         | What it does                                                 |
| ------------------------------------------------------------------------------- | ------------------------------------------------------------ |
| [`hatch init`](docs/cli.md#hatch-init) `[-examples] [-path p]`                  | scaffold `.hatch/` (optionally with example files or nested) |
| [`hatch new`](docs/cli.md#hatch-new) `<kind> [-path p] [title]`                 | create a new rule, skill, command, or agent from a template  |
| [`hatch gen`](docs/cli.md#hatch-gen) `[-targets names]`                         | write every target's native files                            |
| [`hatch list`](docs/cli.md#hatch-list) `[-targets names]`                       | dry-run; print what `gen` would write                        |
| [`hatch clean`](docs/cli.md#hatch-clean) `[-targets names]`                     | remove everything hatch generated                            |
| [`hatch version`, `hatch help`](docs/cli.md#hatch-version-and-help)             | print version / usage                                        |

See [docs/cli.md](docs/cli.md) for the full flag reference, including
`-no-hatch-skill` and the auto-injected meta skill.

## Docs

- [CLI reference](docs/cli.md) — every subcommand and flag
- [Source layout, nesting, and frontmatter](docs/sources.md) — how `.hatch/` is organised and what frontmatter fields do
- [Feature support](docs/features.md) — what each target natively supports and how hatch emulates the gaps
- [Targets and output mapping](docs/targets.md) — per-agent file locations and nested-scope routing
- [Development](docs/development.md) — build and test the hatch binary itself
