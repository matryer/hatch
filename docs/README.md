# 🥚 hatch documentation

Welcome! Hatch turns one source tree under `.hatch/` into the native
config files every coding agent reads. Pick a topic below — or, if you
want the elevator pitch first, head back to the [project README](../README.md).

## Authoring

How to write rules, skills, commands, and sub-agents in `.hatch/`, and
how to drive hatch from the command line.

- **[Source layout, nesting, and frontmatter](sources.md)** — the four
  primitives, the `_rules` / `_skills` / `_commands` / `_agents`
  containers, nested scopes for monorepos, and the frontmatter fields
  that change generation behaviour.
- **[CLI reference](cli.md)** — every subcommand (`gen`, `list`,
  `clean`, `init`, `new`, `version`, `help`) and flag, with examples.

## Targets

Where each generated file goes, what each target natively supports,
and how hatch closes the gaps.

- **[Targets and output mapping](targets.md)** — per-agent file
  locations, the file-owned vs. block-injected distinction, and the
  block-marker format. Includes design references for each agent's
  own docs.
- **[Feature support](features.md)** — the support matrix (`✓` native,
  `⚠` emulated) plus a section per emulated cell explaining the
  trade-offs.

## Contributing

- **[Development](development.md)** — build and test the hatch binary
  itself.

---

Found a doc bug, or something missing? Open an issue at
[github.com/grafana/hatch/issues](https://github.com/grafana/hatch/issues).
