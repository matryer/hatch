# Source layout, nesting, and frontmatter

## Layout

```
.hatch/
  _rules/
    coding-style.md         # always-on project instructions
    testing.md              # may have an applyTo: "**/*_test.go" glob
  _skills/
    review-pr/              # skills are directories, not single files
      SKILL.md
      scripts/review.sh     # sibling assets copy through verbatim
  _commands/
    commit.md               # user-invoked slash prompts
  _agents/
    security-auditor.md     # delegated sub-agents
```

Source files are plain markdown. Skills, commands, and agents carry a
YAML frontmatter header (see below).

The underscore prefix on the four primitive container directories
distinguishes them from ordinary path components in a nested `.hatch/`
tree. Only those four exact names (`_rules`, `_skills`, `_commands`,
`_agents`) are recognised — anything else is a scope path component.

## Nesting for monorepos

Drop a path component between `.hatch/` and a primitive container, and
its outputs are emitted under the same path:

```
.hatch/
  _rules/                          # root scope: applies repo-wide
    coding-style.md
  backend/
    _rules/
      database.md                  # only loaded when working in backend/
    _skills/
      check-migrations/
        SKILL.md
  services/api/
    _commands/
      deploy.md
```

Generates:

```
CLAUDE.md
AGENTS.md
backend/CLAUDE.md
backend/AGENTS.md
backend/.claude/skills/check-migrations/SKILL.md
services/api/CLAUDE.md
services/api/.claude/commands/deploy.md
…
```

Claude Code, Codex, and OpenCode pick up nested `CLAUDE.md` / `AGENTS.md`
natively. See [targets.md](targets.md) for how Copilot and Cursor
handle nested scopes.

Path components must avoid the four primitive container names. Other
`_`-prefixed names work but are discouraged in case future hatch
versions add new primitive containers.

## Namespaced commands

Claude Code reads a subdirectory under `.claude/commands/` as a
namespaced slash command: `.claude/commands/opsx/apply.md` is invoked
as `/opsx:apply`. To express that in hatch source, drop the file into
a subdirectory inside `_commands/`:

```
.hatch/
  _commands/
    commit.md           # /commit
    opsx/
      apply.md          # /opsx:apply in Claude Code
      verify.md         # /opsx:verify in Claude Code
```

The relative path becomes the command's name (forward-slash joined,
`opsx/apply`). Only Claude Code preserves the namespace in its output
path. The other targets have no namespace convention, so hatch
flattens `/` → `-` for their filenames and any textual identifier:

| Target      | Output for `opsx/apply.md`                         |
| ----------- | -------------------------------------------------- |
| Claude Code | `.claude/commands/opsx/apply.md` (native subdir)   |
| Codex       | `### Command: opsx-apply` in `AGENTS.md`           |
| Copilot     | `.github/prompts/opsx-apply.prompt.md`             |
| Cursor      | `.cursor/rules/command-opsx-apply.mdc`             |
| OpenCode    | `.opencode/commands/opsx-apply.md`                 |

Nesting works for any depth: `_commands/foo/bar/baz.md` loads with
name `foo/bar/baz` and produces `.claude/commands/foo/bar/baz.md` for
Claude and `foo-bar-baz` everywhere else.

`hatch new command "opsx/apply"` slugifies the title and creates a
flat file (`opsx-apply.md`); to scaffold a namespaced command, create
the subdirectory manually and drop the `.md` in.

## Frontmatter

Skills, commands, and agents start with a YAML frontmatter block
delimited by `---`. Only `description` is required.

```markdown
---
description: Review a PR for correctness, style, and tests.
name: review-pr
targets: [claude, opencode]
applyTo: "**/*.go"
claude:
  allowed-tools: [Read, Grep]
copilot:
  model: gpt-4.1
---

Body markdown. Two template vars substitute at generation time:
- {{agent}}  → the agent display name, e.g. "Claude Code"
- {{target}} → the target short name, e.g. "claude"
```

| Field         | Applies to              | Required | Meaning                                                                                           |
| ------------- | ----------------------- | -------- | ------------------------------------------------------------------------------------------------- |
| `description` | skill, command, agent   | yes      | One-sentence summary.                                                                             |
| `name`        | skill, command, agent   | no       | Overrides the filename/dirname slug.                                                              |
| `targets`     | all                     | no       | List of target names. Empty/absent means every target. A single string is accepted.               |
| `applyTo`     | rule                    | no       | Glob pattern limiting the rule to matching paths.                                                 |
| `claude:` etc | all                     | no       | Per-target passthrough block — its keys merge into the generated file's own frontmatter.          |

Rules are plain markdown and don't need frontmatter. If they have one,
only `targets` and `applyTo` are meaningful.

## Generated metadata

Every file-owned output with frontmatter carries a `metadata:` block
so a reader of the generated file can see which hatch built it and
where to go to edit the source:

```yaml
metadata:
  generated: hatch@v0.4.0
  source: .hatch/_skills/review-pr/SKILL.md
```

The shape follows the
[agentskills.io `metadata` convention](https://agentskills.io/specification#metadata-field)
— a free-form string/string map — so hatch's keys can't collide with
spec-defined fields like `license` or `compatibility`. The same shape
is used on non-SKILL.md outputs (commands, agents, `.mdc` rules,
Copilot instructions files) for consistency.

Block-injected files (`CLAUDE.md`, `AGENTS.md`,
`.github/copilot-instructions.md`, `.rules`) have no frontmatter of
their own and don't carry this metadata. The hatch-injected meta skill omits
`source` because no `.hatch/_skills/hatch/SKILL.md` exists until you
create one.
