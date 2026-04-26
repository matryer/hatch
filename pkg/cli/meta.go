package cli

import "github.com/grafana/hatch/pkg/source"

// loadSource is the gen/list/clean shared entry point. It loads .hatch/
// from the current working directory and, unless includeMeta is false,
// injects the hatch meta skill into the root scope so the three
// commands see the same primitives — which in particular means
// `hatch clean` removes the same meta-skill outputs that `hatch gen`
// wrote. Callers pass includeMeta from a `-no-hatch-skill` flag: pass
// false to opt out of the auto-injected meta skill for this run.
//
// hatchVersion is stamped into the returned Source so targets can
// embed it as metadata.generated in every generated file's frontmatter.
func loadSource(includeMeta bool, hatchVersion string) (*source.Source, error) {
	s, err := source.Load(".")
	if err != nil {
		return nil, err
	}
	s.HatchVersion = hatchVersion
	if includeMeta {
		injectMetaSkill(s)
	}
	return s, nil
}

// injectMetaSkill adds the hatch meta skill to the root scope of src so
// every `hatch gen` run automatically writes a SKILL.md teaching coding
// agents how `.hatch/` is structured and how to extend it. This is the
// successor to the (removed) `hatch meta skill` subcommand: instead of
// asking users to drop the meta skill into their tree manually, hatch
// includes it in every generation pass.
//
// If the user already has a root-scope skill named "hatch", that one
// wins — injectMetaSkill is a no-op so the user can override the
// content of the meta skill by writing their own `.hatch/_skills/hatch/`.
func injectMetaSkill(s *source.Source) {
	root := s.Root()
	if root == nil {
		return
	}
	for _, sk := range root.Skills {
		if sk.Name == metaSkillName {
			return
		}
	}
	root.Skills = append(root.Skills, source.Primitive{
		Kind:        source.KindSkill,
		Name:        metaSkillName,
		Description: metaSkillDescription,
		Body:        metaSkillBody,
	})
}

// metaSkillName, metaSkillDescription, metaSkillBody are the three
// pieces that together form the hatch meta SKILL.md that gets injected
// into every `hatch gen` run. Keeping them as separate constants lets
// injectMetaSkill build a Primitive without re-parsing.
const (
	metaSkillName        = "hatch"
	metaSkillDescription = "Authoring rules, skills, commands, and sub-agents for this project via hatch — write once, generate for every coding agent."
)

const metaSkillBody = `# hatch

This project uses **hatch** (` + "`github.com/grafana/hatch`" + `) to keep a single
source of truth for the guidance it sends to coding agents. Hatch reads a
directory under ` + "`.hatch/`" + ` and produces the native files each agent expects
(Claude Code, OpenAI Codex, GitHub Copilot, Cursor, OpenCode, Zed).

When you are asked to add, change, or remove rules, skills, slash commands,
or sub-agent definitions in this project, edit files under ` + "`.hatch/`" + ` — not
the generated files in ` + "`CLAUDE.md`" + `, ` + "`AGENTS.md`" + `, ` + "`.rules`" + `,
` + "`.claude/`" + `, ` + "`.agents/`" + `, ` + "`.github/`" + `, ` + "`.cursor/`" + `, or ` + "`.opencode/`" + `.
After editing, run ` + "`hatch gen`" + ` from the project root to regenerate
everything.

## Install

` + "```" + `
go install github.com/grafana/hatch/cmd/hatch@latest
` + "```" + `

## The four primitives

Hatch primitive containers use an underscore prefix to distinguish them
from user-authored path components in nested layouts. The four exact
container names are ` + "`_rules`" + `, ` + "`_skills`" + `, ` + "`_commands`" + `, and ` + "`_agents`" + ` —
any other directory under ` + "`.hatch/`" + ` is a scope path component (see
"Nested paths" below).

| Kind      | Purpose                                                            | Source path                            |
|-----------|--------------------------------------------------------------------|----------------------------------------|
| ` + "`rule`" + `    | Always-on project instructions; optionally scoped with a glob      | ` + "`.hatch/_rules/<slug>.md`" + `              |
| ` + "`skill`" + `   | Model-invoked in-session capability; supports sibling assets       | ` + "`.hatch/_skills/<slug>/SKILL.md`" + `       |
| ` + "`command`" + ` | User-invoked slash prompt                                          | ` + "`.hatch/_commands/<slug>.md`" + `           |
| ` + "`agent`" + `   | Delegated sub-agent definition                                     | ` + "`.hatch/_agents/<slug>.md`" + `             |

## Nested paths

For monorepos that need different guidance per area, put a path
component between ` + "`.hatch/`" + ` and a primitive container. The path becomes
a prefix on the generated output:

` + "```" + `
.hatch/backend/_rules/style.md     → backend/CLAUDE.md, backend/AGENTS.md
.hatch/services/api/_skills/check  → services/api/.claude/skills/check/SKILL.md
` + "```" + `

Claude Code, Codex, and OpenCode all read nested ` + "`CLAUDE.md`" + ` /
` + "`AGENTS.md`" + ` files from subdirectories natively. Copilot, Cursor, and
Zed only read their config from the repo root, so hatch routes scoped
output for those targets through their native scoping mechanism — for
example ` + "`.github/instructions/<scope-slug>-<name>.instructions.md`" + ` with
an auto-generated ` + "`applyTo`" + ` glob for Copilot, ` + "`.cursor/rules/<scope-slug>-<name>.mdc`" + `
with a ` + "`globs`" + ` field for Cursor, and a ` + "`# Scope: <path>/`" + ` heading inside
the root ` + "`.rules`" + ` file for Zed.

Path components must not match one of the four primitive container names
(` + "`_rules`" + `, ` + "`_skills`" + `, ` + "`_commands`" + `, ` + "`_agents`" + `). Other ` + "`_`" + `-prefixed
names work, but it's good practice to avoid them — future hatch versions
may add new primitive container names.

## Creating a new source file

Use ` + "`hatch new`" + `:

` + "```" + `
hatch new rule "Always run gofmt before committing"
hatch new skill "Review pull requests"
hatch new command "Commit with generated message"
hatch new agent "Security auditor"
` + "```" + `

Add ` + "`-path <relative-path>`" + ` to write under a nested scope:

` + "```" + `
hatch new rule -path backend "Database access patterns"
hatch new skill -path services/api "Smoke test the API"
` + "```" + `

For skills, the file is ` + "`SKILL.md`" + ` inside a directory — place sibling
assets (scripts, references) alongside and they copy through verbatim.

You can also write files by hand under ` + "`.hatch/[<path>/]_<kind>s/...`" + `
if you prefer — ` + "`hatch new`" + ` is just a scaffolding helper.

## Frontmatter

Skills, commands, and agents carry a YAML frontmatter header. Only
` + "`description`" + ` is required.

| Field         | Required | Meaning                                                                                              |
|---------------|----------|------------------------------------------------------------------------------------------------------|
| ` + "`description`" + ` | yes      | One-sentence summary. Downstream agents either require or recommend this.                            |
| ` + "`name`" + `        | no       | Overrides the slug. Defaults to the source filename/dirname.                                         |
| ` + "`targets`" + `     | no       | List of target names (` + "`claude`" + `, ` + "`codex`" + `, ` + "`copilot`" + `, ` + "`cursor`" + `, ` + "`opencode`" + `, ` + "`zed`" + `). Empty/absent → all.   |
| ` + "`applyTo`" + `     | no       | Rules only. Glob pattern scoping the rule to matching files.                                         |
| ` + "`claude:`" + ` etc | no       | Per-target passthrough block — its contents merge into the generated file's own frontmatter.        |

Rule files are plain markdown and don't need frontmatter at all. Bodies may
use ` + "`{{agent}}`" + ` and ` + "`{{target}}`" + ` — these are substituted at generation time
with the agent's display name (e.g., "Claude Code") and short name (e.g.,
"claude").

## Workflow

1. ` + "`cd`" + ` to the project root.
2. Edit or create files under ` + "`.hatch/`" + ` — either by hand or with ` + "`hatch new`" + `.
3. Run ` + "`hatch gen`" + ` to regenerate every agent's files.
4. Commit both the ` + "`.hatch/`" + ` source changes and the regenerated output
   files together.

## Useful commands

` + "```" + `
hatch init                        # scaffold an empty .hatch/ tree
hatch init -examples              # …plus one starter example of each primitive
hatch init -path backend          # …scaffold under a nested scope
hatch new <kind> <title>          # create a new source file
hatch new <kind> -path <p> <t>    # create one under a nested scope
hatch gen                         # regenerate all target files
hatch gen -targets claude         # regenerate only one agent's files
hatch list                        # dry-run: show what gen would write
hatch clean                       # remove everything hatch generated
` + "```" + `

## Never edit generated files

` + "`CLAUDE.md`" + `, ` + "`AGENTS.md`" + `, ` + "`.rules`" + `, ` + "`.github/copilot-instructions.md`" + `,
and everything under ` + "`.claude/`" + `, ` + "`.agents/`" + `, ` + "`.cursor/`" + `, ` + "`.opencode/`" + `,
` + "`.github/prompts/`" + `, ` + "`.github/instructions/`" + `, and ` + "`.github/agents/`" + ` is
regenerated by hatch. Any edits you make there will be overwritten on the
next ` + "`hatch gen`" + `. Put your changes in ` + "`.hatch/`" + ` instead.

For the block-injected files (` + "`CLAUDE.md`" + `, ` + "`AGENTS.md`" + `, ` + "`.rules`" + `,
` + "`.github/copilot-instructions.md`" + `), hatch only rewrites content between
` + "`<!-- hatch:begin v1 -->`" + ` and ` + "`<!-- hatch:end v1 -->`" + ` markers — surrounding
content you have written by hand is preserved across regeneration.

For nested scopes, the same rules apply to the prefixed paths
(` + "`backend/CLAUDE.md`" + `, ` + "`services/api/AGENTS.md`" + `, etc.).
`
