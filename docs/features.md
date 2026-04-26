# Feature support

This page expands the matrix in the
[README](../README.md#feature-support) with the specific output paths
and trade-offs for each emulated cell. For the full per-primitive
output mapping (every cell, not just the emulated ones), see
[targets.md](targets.md).

✓ means the agent has a native primitive; ⚠ means hatch emulates it.
The superscript points to the matching section below.

| Feature                   | Claude Code | Codex | Copilot | Cursor | OpenCode |   Zed   |
| ------------------------- | :---------: | :---: | :-----: | :----: | :------: | :-----: |
| Rules (always-on)         |      ✓      |   ✓   |    ✓    |   ✓    |    ✓     |   ✓     |
| Rules (scoped `applyTo`)  |     ⚠¹      |  ⚠¹   |    ✓    |   ✓    |    ⚠¹    |   ⚠¹    |
| Skills                    |      ✓      |   ✓   |    ⚠²   |   ⚠³   |    ✓     |   ⚠²    |
| Slash commands            |      ✓      |  ⚠⁴   |    ✓    |   ⚠³   |    ✓     |   ⚠⁴    |
| Sub-agents                |      ✓      |  ⚠⁴   |    ✓    |   ⚠³   |    ✓     |   ⚠⁴    |
| Nested scopes (monorepo)  |      ✓      |   ✓   |    ⚠⁵   |   ⚠⁶   |    ✓     |   ⚠⁷    |

## 1. Scoped rules inlined into the main instructions block

Applies to: Claude Code, Codex, OpenCode, Zed.

Claude's `CLAUDE.md` and Codex/OpenCode's `AGENTS.md` have no native
way to scope a rule to a glob. Rather than drop the scope, hatch emits
the rule as a sub-section of the hatch-managed block, with the
`applyTo` glob surfaced in the heading so the agent can see the intended
scope while reading:

```markdown
## testing.md (applies to: **/*_test.go)
…rule body…
```

The agent loads the rule unconditionally but has the scope in context.
Copilot and Cursor, which have native `applyTo` / `globs` fields, get
true path-based scoping with no inlining.

## 2. Skills inlined into the rules file

Applies to: Copilot (`.github/copilot-instructions.md`), Zed (`.rules`).

Neither Copilot nor Zed has a native skill primitive. Hatch merges
each skill's body into the hatch-managed block inside the target's
rules file under a `## Skills` section.

Trade-offs:

- Skills are loaded every turn, not on-demand when the description
  matches the task — they consume context whether relevant or not.
- Sibling assets (`scripts/`, `reference/`, etc.) are not copied,
  because the instructions file is the only thing the agent reads.

## 3. Cursor non-rule primitives as kind-prefixed rules

Applies to: Cursor skills, commands, sub-agents.

Cursor's only stable primitive is the rule (`.cursor/rules/*.mdc`).
Skills, commands, and sub-agents are emitted as additional rule files
with a kind prefix in the filename, so you can see at a glance which
rule files came from which hatch primitive:

- skill   → `.cursor/rules/skill-<n>.mdc`
- command → `.cursor/rules/command-<n>.mdc`
- agent   → `.cursor/rules/agent-<n>.mdc`

The content fires as always-on rule context. Consequences:

- Commands can't be user-invoked via a slash — the body is just
  advice the agent always sees.
- Sub-agents can't be dispatched to as a separate context — the
  description is read as inline guidance.

## 4. Commands and sub-agents inlined into the rules file

Applies to: Codex (`AGENTS.md`), Zed (`.rules`).

Neither Codex nor Zed has a first-class markdown primitive for slash
commands or sub-agents. Hatch appends each one to the hatch-managed
block in the target's rules file under a `## Commands` or
`## Sub-agents` section, with per-entry instructions telling the agent
how to invoke them. They're not reachable via a slash UI — the agent
sees them as written instructions only.

## 5. Copilot nested scopes via `applyTo` rewrites

Applies to: Copilot.

Copilot only reads `.github/` at the repo root. For a nested scope like
`backend/_rules/style.md`, hatch emits
`.github/instructions/backend-style.instructions.md` with
`applyTo: backend/**`. The filename carries a slug prefix
(`<scope>-<n>`) so nested scopes from different sub-trees don't
collide at the root.

Skills, commands, and sub-agents under a nested scope follow the same
path-flattening strategy within their respective `.github/` directories
(`prompts/`, `agents/`).

## 6. Cursor nested scopes via `globs` rewrites

Applies to: Cursor.

Cursor only reads `.cursor/` at the repo root. Nested scopes use the
same strategy as Copilot, through Cursor's native `globs` field: a
`.cursor/rules/<scope>-<n>.mdc` file with `globs: <scope>/**`, and
kind-prefixed filenames for non-rule primitives
(`<scope>-skill-<n>.mdc`, etc.).

## 7. Zed: everything inlined into a single rules file

Applies to: Zed.

Zed reads project guidance from a single rules file at the repo root
(it picks the first match in a priority list — `.rules`,
`.cursorrules`, …, `AGENTS.md`, `CLAUDE.md`). Hatch writes to
`.rules`, Zed's highest-priority filename, to avoid colliding with
Codex / OpenCode (both own `AGENTS.md`).

Zed has no project-level concept of skills, slash commands, or
sub-agents (native skill support is in flight — issue
[zed-industries/zed#49057](https://github.com/zed-industries/zed/issues/49057)).
Hatch inlines all four primitives — rules, skills, commands, and
sub-agents — into `.rules` using the same patterns as sections 1, 2,
and 4 above. Sibling skill assets are not copied because `.rules` is
the only file Zed reads.

Zed only reads the project root, so nested scopes flatten into the
same root `.rules` file. Each non-root scope's content gets a
`# Scope: <path>/` heading so the agent sees what each section relates
to. Without a glob field there is no path-based enforcement; the agent
loads everything every turn.
